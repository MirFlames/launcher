package main

import (
	"log"
	"net"
	"os"
	"sync"
	"time"
)

func main() {
	initSessionsDB()
	initBanList()
	initLimits()

	mcBackend := getEnv("MC_BACKEND", "127.0.0.1:25567")
	voiceBackend := getEnv("VOICE_BACKEND", "127.0.0.1:24456")
	mcListen := getEnv("MC_LISTEN", ":25565")
	voiceListen := getEnv("VOICE_LISTEN", ":24454")

	var wg sync.WaitGroup

	// TCP proxy для Minecraft
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := runTCPProxy(mcListen, mcBackend); err != nil {
			log.Fatalf("[MC] %v", err)
		}
	}()

	// UDP proxy для Simple Voice Chat
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := runUDPProxy(voiceListen, voiceBackend); err != nil {
			log.Fatalf("[Voice] %v", err)
		}
	}()

	log.Printf("mc-proxy: MC %s -> %s, Voice %s -> %s", mcListen, mcBackend, voiceListen, voiceBackend)
	wg.Wait()
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func runTCPProxy(listenAddr, backendAddr string) error {
	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}
	defer ln.Close()

	for {
		client, err := ln.Accept()
		if err != nil {
			return err
		}
		go handleTCPConn(client, backendAddr)
	}
}

func runUDPProxy(listenAddr, backendAddr string) error {
	addr, err := net.ResolveUDPAddr("udp", listenAddr)
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	backendAddrResolved, err := net.ResolveUDPAddr("udp", backendAddr)
	if err != nil {
		return err
	}

	type clientSession struct {
		backend *net.UDPConn
	}
	sessions := make(map[string]*clientSession)
	var mu sync.RWMutex

	buf := make([]byte, 65535)
	for {
		n, clientAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("[Voice] read: %v", err)
			continue
		}
		if n == 0 {
			continue
		}

		key := clientAddr.String()
		mu.RLock()
		s := sessions[key]
		mu.RUnlock()

		if s == nil {
			backendConn, err := net.DialUDP("udp", nil, backendAddrResolved)
			if err != nil {
				log.Printf("[Voice] dial %s: %v", backendAddr, err)
				continue
			}
			s = &clientSession{backend: backendConn}
			mu.Lock()
			sessions[key] = s
			mu.Unlock()

			go func(clientAddr *net.UDPAddr, backendConn *net.UDPConn) {
				replyBuf := make([]byte, 65535)
				for {
					backendConn.SetReadDeadline(time.Now().Add(5 * time.Minute))
					m, err := backendConn.Read(replyBuf)
					if err != nil {
						break
					}
					if m > 0 {
						conn.WriteToUDP(replyBuf[:m], clientAddr)
					}
				}
				backendConn.Close()
				mu.Lock()
				delete(sessions, clientAddr.String())
				mu.Unlock()
			}(clientAddr, backendConn)
		}

		_, err = s.backend.Write(buf[:n])
		if err != nil {
			log.Printf("[Voice] write to backend: %v", err)
		}
	}
}
