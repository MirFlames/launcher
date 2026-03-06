package main

import (
	"bytes"
	"encoding/binary"
	"io"

	pk "github.com/Tnze/go-mc/net/packet"
)

// handshakeNextState извлекает next state из пакета Handshake
// Формат: VarInt protocolVersion, String serverAddress, Short port, VarInt nextState
func handshakeNextState(data []byte) (int32, error) {
	br := bytes.NewReader(data)
	var protocolVersion pk.VarInt
	if _, err := protocolVersion.ReadFrom(br); err != nil {
		return 0, err
	}
	var serverAddr pk.String
	if _, err := serverAddr.ReadFrom(br); err != nil {
		return 0, err
	}
	var port uint16
	if err := binary.Read(br, binary.BigEndian, &port); err != nil {
		return 0, err
	}
	var nextState pk.VarInt
	if _, err := nextState.ReadFrom(br); err != nil && err != io.EOF {
		return 0, err
	}
	return int32(nextState), nil
}

const (
	handshakeStateStatus = 1
	handshakeStateLogin  = 2
)
