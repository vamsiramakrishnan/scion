// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package hub

import (
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

// RateLimitConfig holds rate limiting configuration.
type RateLimitConfig struct {
	AuthenticatedRPS     float64 // requests per second for authenticated users
	UnauthenticatedRPS   float64 // requests per second for unauthenticated IPs
	AuthenticatedBurst   int     // burst capacity for authenticated users
	UnauthenticatedBurst int     // burst capacity for unauthenticated IPs
}

// DefaultRateLimitConfig returns sensible defaults.
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		AuthenticatedRPS:     100,
		UnauthenticatedRPS:   20,
		AuthenticatedBurst:   200,
		UnauthenticatedBurst: 50,
	}
}

// HTTPRateLimiter provides per-identity HTTP rate limiting.
type HTTPRateLimiter struct {
	config  RateLimitConfig
	mu      sync.Mutex
	buckets map[string]*rateBucket
	cleanup time.Duration
}

type rateBucket struct {
	tokens    float64
	rate      float64
	burst     int
	lastCheck time.Time
}

// NewHTTPRateLimiter creates a new HTTP rate limiter.
func NewHTTPRateLimiter(config RateLimitConfig) *HTTPRateLimiter {
	rl := &HTTPRateLimiter{
		config:  config,
		buckets: make(map[string]*rateBucket),
		cleanup: 10 * time.Minute,
	}
	go rl.cleanupLoop()
	return rl
}

// Allow checks if the request is allowed for the given key with the specified rate and burst.
func (rl *HTTPRateLimiter) Allow(key string, rate float64, burst int) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, ok := rl.buckets[key]
	if !ok {
		rl.buckets[key] = &rateBucket{
			tokens:    float64(burst) - 1,
			rate:      rate,
			burst:     burst,
			lastCheck: now,
		}
		return true
	}

	elapsed := now.Sub(b.lastCheck).Seconds()
	b.tokens += elapsed * rate
	if b.tokens > float64(burst) {
		b.tokens = float64(burst)
	}
	b.lastCheck = now

	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

func (rl *HTTPRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanup)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		cutoff := time.Now().Add(-30 * time.Minute)
		for id, b := range rl.buckets {
			if b.lastCheck.Before(cutoff) {
				delete(rl.buckets, id)
			}
		}
		rl.mu.Unlock()
	}
}

// Middleware returns an HTTP middleware that enforces rate limits.
// It extracts identity from the request context (set by auth middleware).
// Since rate limiting runs BEFORE auth middleware in the chain, we use IP-based
// limiting here and identity-based limiting is applied post-auth via a second check.
func (rl *HTTPRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip rate limiting for health and readiness checks
		if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
			next.ServeHTTP(w, r)
			return
		}

		// Extract identity: check auth headers for keying
		key, rate, burst := rl.resolveRateKey(r)

		if !rl.Allow(key, rate, burst) {
			w.Header().Set("Retry-After", "1")
			writeError(w, http.StatusTooManyRequests, "rate_limit_exceeded", "Rate limit exceeded. Try again shortly.", nil)
			slog.Warn("Rate limit exceeded", "key", key, "path", r.URL.Path)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// resolveRateKey determines the rate limiting key and rates for a request.
func (rl *HTTPRateLimiter) resolveRateKey(r *http.Request) (key string, rate float64, burst int) {
	// Check for agent token
	if agentToken := r.Header.Get("X-Scion-Agent-Token"); agentToken != "" {
		// Use a hash prefix of the token as key (don't store full token)
		key = "agent:" + agentToken[:min(16, len(agentToken))]
		return key, rl.config.AuthenticatedRPS, rl.config.AuthenticatedBurst
	}

	// Check for bearer token
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		token := auth[7:]
		key = "bearer:" + token[:min(16, len(token))]
		return key, rl.config.AuthenticatedRPS, rl.config.AuthenticatedBurst
	}

	// Check for broker ID
	if brokerID := r.Header.Get("X-Scion-Broker-ID"); brokerID != "" {
		key = "broker:" + brokerID
		return key, rl.config.AuthenticatedRPS, rl.config.AuthenticatedBurst
	}

	// Unauthenticated: key by IP
	ip := r.RemoteAddr
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		ip = strings.Split(fwd, ",")[0]
	}
	key = "ip:" + strings.TrimSpace(ip)
	return key, rl.config.UnauthenticatedRPS, rl.config.UnauthenticatedBurst
}
