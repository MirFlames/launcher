// tcp-flood — нагрузочное тестирование mc-proxy
//
// Использование:
//
//	tcp-flood -target localhost:25565 -connections 100 -concurrent 20
//	tcp-flood -target localhost:25565 -mode status -connections 500
package main

import (
	"flag"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	pk "github.com/Tnze/go-mc/net/packet"
	"github.com/google/uuid"
)

func main() {
	target := flag.String("target", "localhost:25565", "host:port mc-proxy")
	connections := flag.Int("connections", 100, "число подключений")
	concurrent := flag.Int("concurrent", 20, "параллельных горутин")
	mode := flag.String("mode", "login", "login или status")
	nickname := flag.String("nickname", "LoadTest", "никнейм для login")
	sessionUUID := flag.String("uuid", "00000000-0000-0000-0000-000000000001", "session_uuid для login (должен быть в sessions.db)")
	timeout := flag.Duration("timeout", 10*time.Second, "таймаут на подключение")
	flag.Parse()

	var ok, fail int64
	start := time.Now()

	var wg sync.WaitGroup
	sem := make(chan struct{}, *concurrent)

	for i := 0; i < *connections; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func(n int) {
			defer wg.Done()
			defer func() { <-sem }()

			err := runConnection(*target, *mode, *nickname, *sessionUUID, *timeout)
			if err != nil {
				atomic.AddInt64(&fail, 1)
				return
			}
			atomic.AddInt64(&ok, 1)
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	fmt.Printf("\n--- Результаты ---\n")
	fmt.Printf("Целевой адрес: %s\n", *target)
	fmt.Printf("Режим: %s\n", *mode)
	fmt.Printf("Успешно: %d\n", ok)
	fmt.Printf("Ошибок: %d\n", fail)
	fmt.Printf("Время: %v\n", elapsed.Round(time.Millisecond))
	if elapsed.Seconds() > 0 {
		fmt.Printf("Скорость: %.1f conn/s\n", float64(ok+fail)/elapsed.Seconds())
	}
}

func runConnection(target, mode, nickname, sessionUUID string, timeout time.Duration) error {
	conn, err := net.DialTimeout("tcp", target, timeout)
	if err != nil {
		return err
	}
	defer conn.Close()

	mcConn := wrapConn(conn)
	const threshold = -1

	handshake := buildHandshake(mode == "status")
	if err := handshake.Pack(conn, threshold); err != nil {
		return err
	}

	if mode == "status" {
		statusReq := pk.Marshal(0x00) // Status Request — пустой пакет
		if err := statusReq.Pack(conn, threshold); err != nil {
			return err
		}
		var statusResp pk.Packet
		if err := mcConn.ReadPacket(&statusResp); err != nil {
			return err
		}
		return nil
	}

	loginStart := buildLoginStart(nickname, sessionUUID)
	if err := loginStart.Pack(conn, threshold); err != nil {
		return err
	}

	var p pk.Packet
	if err := mcConn.ReadPacket(&p); err != nil {
		return err
	}
	return nil
}

func buildHandshake(status bool) pk.Packet {
	nextState := int32(2)
	if status {
		nextState = 1
	}
	return pk.Marshal(
		0x00,
		pk.VarInt(767),
		pk.String("localhost"),
		pk.UnsignedShort(25565),
		pk.VarInt(nextState),
	)
}

func buildLoginStart(nickname, sessionUUID string) pk.Packet {
	id, _ := uuid.Parse(sessionUUID)
	return pk.Marshal(
		0x00,
		pk.String(nickname),
		pk.UUID(id),
	)
}

type conn struct {
	net.Conn
}

func wrapConn(c net.Conn) *conn {
	return &conn{Conn: c}
}

func (c *conn) ReadPacket(p *pk.Packet) error {
	return p.UnPack(c.Conn, -1)
}
