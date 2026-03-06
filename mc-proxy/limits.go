package main

import (
	"log"
	"os"
	"strconv"
	"sync"
	"time"
)

var (
	// Глобальный лимит одновременных соединений
	globalConnLimit int
	globalConnSem   chan struct{}

	// Семафор на dial к mc-серверу
	backendDialLimit int
	backendDialSem   chan struct{}

	// Лимит одновременных соединений с одного IP (2–3 для игрока)
	perIPLimit      int
	perIPConnCount  map[string]int
	perIPConnMutex  sync.Mutex
)

func initLimits() {
	globalConnLimit = getEnvInt("MC_MAX_CONNECTIONS", 200)
	globalConnSem = make(chan struct{}, globalConnLimit)

	backendDialLimit = getEnvInt("MC_MAX_BACKEND_DIALS", 50)
	backendDialSem = make(chan struct{}, backendDialLimit)

	perIPLimit = getEnvInt("MC_MAX_CONNECTIONS_PER_IP", 3)
	perIPConnCount = make(map[string]int)

	log.Printf("[MC] limits: global=%d backend_dials=%d per_ip=%d",
		globalConnLimit, backendDialLimit, perIPLimit)
}

func getEnvInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return defaultVal
}

// tryAcquireGlobalConn возвращает true, если соединение разрешено
func tryAcquireGlobalConn() bool {
	select {
	case globalConnSem <- struct{}{}:
		return true
	default:
		return false
	}
}

func releaseGlobalConn() {
	<-globalConnSem
}

// tryAcquireBackendDial пытается получить слот на dial с таймаутом
func tryAcquireBackendDial(timeout time.Duration) bool {
	select {
	case backendDialSem <- struct{}{}:
		return true
	case <-time.After(timeout):
		return false
	}
}

func releaseBackendDial() {
	<-backendDialSem
}

// tryAcquirePerIP возвращает true, если IP может открыть ещё одно соединение
func tryAcquirePerIP(ip string) bool {
	perIPConnMutex.Lock()
	defer perIPConnMutex.Unlock()
	n := perIPConnCount[ip]
	if n >= perIPLimit {
		return false
	}
	perIPConnCount[ip] = n + 1
	return true
}

func releasePerIP(ip string) {
	perIPConnMutex.Lock()
	defer perIPConnMutex.Unlock()
	n := perIPConnCount[ip]
	if n > 1 {
		perIPConnCount[ip] = n - 1
	} else {
		delete(perIPConnCount, ip)
	}
}
