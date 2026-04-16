package state

import (
	"sync"
	"time"
)

type inMemoryCounter struct {
	timestamps []int64
}

type RateLimiter struct {
	mu    sync.Mutex
	store map[string]*inMemoryCounter
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{store: make(map[string]*inMemoryCounter)}
}

func (l *RateLimiter) Allow(key string, maxRequests int, window time.Duration) bool {
	if maxRequests <= 0 {
		return true
	}
	now := time.Now().Unix()
	cutoff := now - int64(window.Seconds())

	l.mu.Lock()
	defer l.mu.Unlock()

	counter, ok := l.store[key]
	if !ok {
		l.store[key] = &inMemoryCounter{timestamps: []int64{now}}
		return true
	}

	valid := make([]int64, 0, len(counter.timestamps)+1)
	for _, ts := range counter.timestamps {
		if ts > cutoff {
			valid = append(valid, ts)
		}
	}

	if len(valid) >= maxRequests {
		counter.timestamps = valid
		return false
	}

	valid = append(valid, now)
	counter.timestamps = valid
	return true
}
