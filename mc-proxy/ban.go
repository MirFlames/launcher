package main

import (
	"bufio"
	"log"
	"net"
	"os"
	"strings"
	"sync"
)

var banList = struct {
	sync.RWMutex
	ips  map[string]struct{}
	path string
}{ips: make(map[string]struct{})}

func initBanList() {
	banList.path = getEnv("BAN_FILE", "bans.txt")
	if err := loadBansFromFile(); err != nil {
		// файл может не существовать при первом запуске
		if !os.IsNotExist(err) {
			log.Printf("[Ban] загрузка %s: %v", banList.path, err)
		}
	}
	if s := getEnv("BAN_IP", ""); s != "" {
		for _, ip := range strings.Split(s, ",") {
			ip = strings.TrimSpace(ip)
			if ip != "" {
				banList.ips[ip] = struct{}{}
			}
		}
	}
}

func loadBansFromFile() error {
	f, err := os.Open(banList.path)
	if err != nil {
		return err
	}
	defer f.Close()
	banList.Lock()
	defer banList.Unlock()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		ip := strings.TrimSpace(sc.Text())
		if ip != "" && !strings.HasPrefix(ip, "#") {
			banList.ips[ip] = struct{}{}
		}
	}
	return sc.Err()
}

func saveBanToFile(ip string) error {
	f, err := os.OpenFile(banList.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(ip + "\n")
	return err
}

func isBanned(ip string) bool {
	banList.RLock()
	_, ok := banList.ips[ip]
	banList.RUnlock()
	return ok
}

func banIP(ip string) {
	banList.Lock()
	if _, exists := banList.ips[ip]; exists {
		banList.Unlock()
		return
	}
	banList.ips[ip] = struct{}{}
	banList.Unlock()

	if err := saveBanToFile(ip); err != nil {
		log.Printf("[Ban] запись в %s: %v", banList.path, err)
	}
}

func clientIP(addr net.Addr) string {
	if addr == nil {
		return ""
	}
	host, _, _ := net.SplitHostPort(addr.String())
	return host
}
