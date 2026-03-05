package main

import (
	"strconv"
	"time"

	mcpinger "github.com/Raqbit/mc-pinger"
)

const minecraftStatusTimeout = 5 * time.Second

// MinecraftStatusResult — результат проверки статуса Minecraft-сервера
type MinecraftStatusResult struct {
	Online      bool
	Count       int      // количество игроков онлайн
	PlayerNames []string // ники из Players.Sample (подвыборка, может быть неполной)
}

// CheckMinecraftServerStatus проверяет доступность сервера и количество игроков.
func CheckMinecraftServerStatus() MinecraftStatusResult {
	port, err := strconv.ParseUint(config.ServerPort, 10, 16)
	if err != nil {
		return MinecraftStatusResult{Online: false}
	}
	p := mcpinger.New(config.ServerHost, uint16(port), mcpinger.WithTimeout(minecraftStatusTimeout))
	info, err := p.Ping()
	if err != nil {
		return MinecraftStatusResult{Online: false}
	}
	names := make([]string, 0, len(info.Players.Sample))
	for _, player := range info.Players.Sample {
		if player.Name != "" {
			names = append(names, player.Name)
		}
	}
	return MinecraftStatusResult{
		Online:      true,
		Count:       int(info.Players.Online),
		PlayerNames: names,
	}
}
