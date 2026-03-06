package main

import (
	"log"
	"net"
	"time"
)

type connLog struct {
	ip        string
	addr      string
	timestamp time.Time
}

func logConnStart(addr net.Addr) connLog {
	ip := clientIP(addr)
	return connLog{
		ip:        ip,
		addr:      addr.String(),
		timestamp: time.Now(),
	}
}

func (c connLog) log(reason, details string) {
	log.Printf("[MC] CONNECT ip=%s addr=%s reason=%s ts=%s %s",
		c.ip, c.addr, reason, c.timestamp.Format(time.RFC3339), details)
}

func (c connLog) logOk(nickname, sessionUUID string) {
	c.log("AUTH_OK", "nickname="+nickname+" session_uuid="+sessionUUID)
}

func (c connLog) logReject(reason, details string) {
	c.log(reason, details)
}

func (c connLog) logSuspicious(kind, details string) {
	c.log("SUSPICIOUS", "kind="+kind+" "+details)
}
