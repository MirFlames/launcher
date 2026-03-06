package main

import (
	"sync"
	"time"
)

const (
	rateLimitWindow = 10 * time.Second
	rateLimitMax    = 20
)

var rateLimiter = struct {
	sync.Mutex
	attempts map[string][]time.Time
}{attempts: make(map[string][]time.Time)}

func rateLimitAllow(ip string) bool {
	rateLimiter.Lock()
	defer rateLimiter.Unlock()

	now := time.Now()
	cutoff := now.Add(-rateLimitWindow)

	attempts := rateLimiter.attempts[ip]
	// Удалить старые
	var kept []time.Time
	for _, t := range attempts {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= rateLimitMax {
		return false
	}
	kept = append(kept, now)
	rateLimiter.attempts[ip] = kept
	return true
}
