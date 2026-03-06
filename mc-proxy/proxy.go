package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"github.com/Tnze/go-mc/data/packetid"
	mcnet "github.com/Tnze/go-mc/net"
	pk "github.com/Tnze/go-mc/net/packet"
	"github.com/google/uuid"
)

func handleTCPConn(client net.Conn, backendAddr string) {
	defer client.Close()

	if !tryAcquireGlobalConn() {
		ip := clientIP(client.RemoteAddr())
		logConnStart(client.RemoteAddr()).logReject("GLOBAL_LIMIT", "too many connections")
		log.Printf("[MC] GLOBAL_LIMIT ip=%s", ip)
		return
	}
	defer releaseGlobalConn()

	ip := clientIP(client.RemoteAddr())
	cl := logConnStart(client.RemoteAddr())
	cl.log("CONNECT", "")

	if isBanned(ip) {
		cl.logReject("BANNED", "ip already in ban list")
		return
	}
	if !rateLimitAllow(ip) {
		cl.logReject("RATE_LIMIT", "too many connection attempts")
		return
	}

	client.SetReadDeadline(time.Now().Add(30 * time.Second))

	mcConn := mcnet.WrapConn(client)
	const threshold = -1 // без сжатия на этапе login

	// Handshake
	var handshake pk.Packet
	if err := mcConn.ReadPacket(&handshake); err != nil {
		cl.logSuspicious("HANDSHAKE_ERROR", "err="+err.Error())
		banIP(ip)
		log.Printf("[MC] BAN ip=%s reason=handshake_error", ip)
		return
	}

	nextState, err := handshakeNextState(handshake.Data)
	if err != nil {
		cl.logSuspicious("HANDSHAKE_PARSE_ERROR", "err="+err.Error())
		banIP(ip)
		log.Printf("[MC] BAN ip=%s reason=handshake_parse_error", ip)
		return
	}

	if nextState == handshakeStateStatus {
		// Ping для списка серверов — пробрасываем без авторизации
		cl.log("STATUS_PING", "")
		if !tryAcquirePerIP(ip) {
			cl.logReject("PER_IP_LIMIT", "too many connections from this IP")
			return
		}
		defer releasePerIP(ip)
		forwardConn(client, mcConn, backendAddr, handshake, nil, threshold)
		return
	}

	if nextState != handshakeStateLogin {
		cl.logSuspicious("UNEXPECTED_HANDSHAKE_STATE", fmt.Sprintf("next_state=%d", nextState))
		return
	}

	// Login Start
	var loginStart pk.Packet
	if err := mcConn.ReadPacket(&loginStart); err != nil {
		cl.logSuspicious("LOGIN_START_ERROR", "err="+err.Error())
		banIP(ip)
		log.Printf("[MC] BAN ip=%s reason=login_start_error", ip)
		return
	}

	client.SetReadDeadline(time.Time{})

	if packetid.ServerboundPacketID(loginStart.ID) != packetid.ServerboundLoginHello {
		cl.logSuspicious("UNEXPECTED_PACKET", fmt.Sprintf("packet_id=%d", loginStart.ID))
		banIP(ip)
		log.Printf("[MC] BAN ip=%s reason=unexpected_packet_id=%d", ip, loginStart.ID)
		return
	}

	var name pk.String
	var id pk.UUID
	if err := loginStart.Scan(&name, &id); err != nil {
		cl.logSuspicious("PARSE_LOGIN_ERROR", "err="+err.Error())
		banIP(ip)
		log.Printf("[MC] BAN ip=%s reason=parse_login_error", ip)
		return
	}

	sessionUUID := uuid.UUID(id).String()

	// Проверка авторизации (sessionsDB обязательна — без неё mc-proxy не стартует)
	if !sessionVerify(string(name), sessionUUID) {
		cl.logReject("AUTH_FAIL", "nickname="+string(name)+" session_uuid="+sessionUUID)
		disconnectPk := buildLoginDisconnectPacket(MsgAuthFail)
		_ = mcConn.WritePacket(disconnectPk)
		return
	}
	cl.logOk(string(name), sessionUUID)

	if !tryAcquirePerIP(ip) {
		cl.logReject("PER_IP_LIMIT", "too many connections from this IP")
		disconnectPk := buildLoginDisconnectPacket(MsgPerIPLimit)
		_ = mcConn.WritePacket(disconnectPk)
		return
	}
	defer releasePerIP(ip)

	cl.log("SUCCESS", "nickname="+string(name)+" session_uuid="+sessionUUID+" -> backend")
	forwardConn(client, mcConn, backendAddr, handshake, &loginStart, threshold)
	cl.log("DISCONNECT", "nickname="+string(name))
}

func forwardConn(client net.Conn, mcConn *mcnet.Conn, backendAddr string, handshake pk.Packet, loginStart *pk.Packet, threshold int) {
	isLogin := loginStart != nil

	if !tryAcquireBackendDial(5 * time.Second) {
		log.Printf("[MC] BACKEND_DIAL_LIMIT backend=%s", backendAddr)
		sendBackendUnavailable(client, mcConn, isLogin)
		return
	}
	defer releaseBackendDial()

	backend, err := net.DialTimeout("tcp", backendAddr, 10*time.Second)
	if err != nil {
		logBackendDialError(err)
		sendBackendUnavailable(client, mcConn, isLogin)
		return
	}
	defer backend.Close()

	if err := writePacket(backend, handshake, threshold); err != nil {
		log.Printf("[MC] BACKEND_WRITE_ERROR handshake: %v", err)
		sendBackendUnavailable(client, mcConn, isLogin)
		return
	}
	if loginStart != nil {
		if err := writePacket(backend, *loginStart, threshold); err != nil {
			log.Printf("[MC] BACKEND_WRITE_ERROR login_start: %v", err)
			sendBackendUnavailable(client, mcConn, isLogin)
			return
		}
	}

	go io.Copy(backend, client)
	io.Copy(client, backend)
}

func sendBackendUnavailable(client net.Conn, mcConn *mcnet.Conn, isLogin bool) {
	if isLogin {
		if err := sendLoginDisconnect(client, MsgBackendUnavailable); err != nil {
			log.Printf("[MC] BACKEND_UNAVAILABLE_WRITE_ERROR: %v", err)
		}
		return
	}

	// Status: читаем Status Request, отправляем Status Response с сообщением, читаем Ping, отправляем Pong
	client.SetReadDeadline(time.Now().Add(5 * time.Second))
	var statusReq pk.Packet
	if err := mcConn.ReadPacket(&statusReq); err != nil {
		return
	}
	client.SetReadDeadline(time.Time{})

	statusResp := buildStatusResponseJSON(MsgBackendUnavailable)
	respPk := pk.Marshal(packetid.ClientboundStatusStatusResponse, pk.String(statusResp))
	_ = mcConn.WritePacket(respPk)

	client.SetReadDeadline(time.Now().Add(5 * time.Second))
	var pingReq pk.Packet
	if err := mcConn.ReadPacket(&pingReq); err != nil {
		return
	}
	client.SetReadDeadline(time.Time{})

	var payload pk.Long
	if err := pingReq.Scan(&payload); err != nil {
		return
	}
	pongPk := pk.Marshal(packetid.ClientboundStatusPongResponse, payload)
	_ = mcConn.WritePacket(pongPk)
}

func writePacket(w io.Writer, p pk.Packet, threshold int) error {
	return p.Pack(w, threshold)
}
