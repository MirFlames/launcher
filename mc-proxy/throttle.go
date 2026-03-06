package main

import (
	"log"
	"sync"
	"time"
)

var logThrottle = struct {
	sync.Mutex
	lastBackendDialErr time.Time
	count              int
}{}

const logThrottleInterval = 5 * time.Second

func logBackendDialError(err error) {
	logThrottle.Lock()
	defer logThrottle.Unlock()

	now := time.Now()
	if now.Sub(logThrottle.lastBackendDialErr) < logThrottleInterval {
		logThrottle.count++
		return
	}
	if logThrottle.count > 0 {
		log.Printf("[MC] BACKEND_DIAL_ERROR: %v (и ещё %d за последние %.0fs)", err, logThrottle.count, logThrottleInterval.Seconds())
	} else {
		log.Printf("[MC] BACKEND_DIAL_ERROR: %v", err)
	}
	logThrottle.lastBackendDialErr = now
	logThrottle.count = 0
}
