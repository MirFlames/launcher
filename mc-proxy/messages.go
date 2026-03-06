package main

import (
	"encoding/json"
	"io"

	"github.com/Tnze/go-mc/data/packetid"
	pk "github.com/Tnze/go-mc/net/packet"
)

// Сообщения об ошибках, отправляемые клиенту
const (
	MsgAuthFail           = "Доступ запрещён. Войдите через лаунчер.\nИнформация тут: t.me/mc_fam"
	MsgPerIPLimit         = "Слишком много соединений с вашего IP."
	MsgBackendUnavailable = "Упс! Кажется сервер прилёг поспать...\nНавряд ли что-то серьезное, скорее всего админ снова решил потестить какую-то фичу... \nСмотри телеграм канал: t.me/mc_fam"
)

// buildLoginDisconnectPacket создаёт пакет Login Disconnect с текстом (красный цвет)
func buildLoginDisconnectPacket(text string) pk.Packet {
	desc := map[string]string{"text": text, "color": "red"}
	descBytes, _ := json.Marshal(desc)
	return pk.Marshal(packetid.ClientboundLoginLoginDisconnect, pk.String(string(descBytes)))
}

// sendLoginDisconnect отправляет Login Disconnect клиенту
func sendLoginDisconnect(w io.Writer, text string) error {
	return writePacket(w, buildLoginDisconnectPacket(text), -1)
}

// buildStatusResponseJSON создаёт JSON для Status Response (description с красным текстом)
func buildStatusResponseJSON(description string) string {
	desc := map[string]string{"text": description, "color": "red"}
	body := map[string]interface{}{
		"version":     map[string]interface{}{"name": "1.21", "protocol": 767},
		"players":     map[string]int{"max": 0, "online": 0},
		"description": desc,
	}
	b, _ := json.Marshal(body)
	return string(b)
}
