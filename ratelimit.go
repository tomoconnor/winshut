// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"net/http"
	"sync"
	"time"
)

type powerRateLimiter struct {
	mu     sync.Mutex
	tokens float64
	last   time.Time
	rate   float64 // tokens per second
	burst  float64
}

func newPowerRateLimiter(rate float64, burst int) *powerRateLimiter {
	return &powerRateLimiter{
		tokens: float64(burst),
		last:   time.Now(),
		rate:   rate,
		burst:  float64(burst),
	}
}

func (l *powerRateLimiter) allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	l.tokens += now.Sub(l.last).Seconds() * l.rate
	l.last = now
	if l.tokens > l.burst {
		l.tokens = l.burst
	}
	if l.tokens < 1 {
		return false
	}
	l.tokens--
	return true
}

func (l *powerRateLimiter) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.allow() {
			writeJSON(w, http.StatusTooManyRequests, response{Status: "error", Message: "rate limit exceeded"})
			return
		}
		next.ServeHTTP(w, r)
	})
}
