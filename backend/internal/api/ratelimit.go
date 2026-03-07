package api

import (
	"errors"
	"log"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ServerConfig struct {
	RateLimit RateLimitConfig
}

type RateLimitConfig struct {
	Enabled        bool
	Window         time.Duration
	MutationLimit  int
	ExpensiveLimit int
	PipelineLimit  int
}

type rateLimitPolicy struct {
	name  string
	limit int
}

type slidingWindowRateLimiter struct {
	mu        sync.Mutex
	window    time.Duration
	buckets   map[string][]time.Time
	lastSweep time.Time
}

func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		RateLimit: RateLimitConfig{
			Enabled:        true,
			Window:         time.Minute,
			MutationLimit:  24,
			ExpensiveLimit: 10,
			PipelineLimit:  6,
		},
	}
}

func newSlidingWindowRateLimiter(window time.Duration) *slidingWindowRateLimiter {
	if window <= 0 {
		window = time.Minute
	}
	return &slidingWindowRateLimiter{
		window:  window,
		buckets: make(map[string][]time.Time),
	}
}

func (l *slidingWindowRateLimiter) Allow(key string, limit int, now time.Time) (bool, int, time.Duration) {
	if limit <= 0 {
		return true, 0, 0
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	l.sweep(now)
	bucket := l.pruneBucketLocked(key, now)
	if len(bucket) >= limit {
		retryAfter := bucket[0].Add(l.window).Sub(now)
		if retryAfter < time.Second {
			retryAfter = time.Second
		}
		return false, 0, retryAfter
	}
	bucket = append(bucket, now)
	l.buckets[key] = bucket
	remaining := limit - len(bucket)
	if remaining < 0 {
		remaining = 0
	}
	return true, remaining, 0
}

func (l *slidingWindowRateLimiter) sweep(now time.Time) {
	if !l.lastSweep.IsZero() && now.Sub(l.lastSweep) < l.window {
		return
	}
	cutoff := now.Add(-l.window)
	for key, bucket := range l.buckets {
		kept := make([]time.Time, 0, len(bucket))
		for _, ts := range bucket {
			if ts.After(cutoff) {
				kept = append(kept, ts)
			}
		}
		if len(kept) == 0 {
			delete(l.buckets, key)
			continue
		}
		l.buckets[key] = kept
	}
	l.lastSweep = now
}

func (l *slidingWindowRateLimiter) pruneBucketLocked(key string, now time.Time) []time.Time {
	bucket := l.buckets[key]
	if len(bucket) == 0 {
		return nil
	}
	cutoff := now.Add(-l.window)
	kept := make([]time.Time, 0, len(bucket))
	for _, ts := range bucket {
		if ts.After(cutoff) {
			kept = append(kept, ts)
		}
	}
	if len(kept) == 0 {
		delete(l.buckets, key)
		return nil
	}
	l.buckets[key] = kept
	return kept
}

func (s *Server) mutationPolicy(name string) rateLimitPolicy {
	return rateLimitPolicy{name: "mutation:" + name, limit: s.config.RateLimit.MutationLimit}
}

func (s *Server) expensivePolicy(name string) rateLimitPolicy {
	return rateLimitPolicy{name: "expensive:" + name, limit: s.config.RateLimit.ExpensiveLimit}
}

func (s *Server) pipelinePolicy(name string) rateLimitPolicy {
	return rateLimitPolicy{name: "pipeline:" + name, limit: s.config.RateLimit.PipelineLimit}
}

func (s *Server) rateLimit(policy rateLimitPolicy) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !s.config.RateLimit.Enabled || s.rateLimiter == nil || policy.limit <= 0 {
				next.ServeHTTP(w, r)
				return
			}

			clientKey := policy.name + "|" + clientIP(r)
			allowed, remaining, retryAfter := s.rateLimiter.Allow(clientKey, policy.limit, time.Now().UTC())
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(policy.limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
			if allowed {
				next.ServeHTTP(w, r)
				return
			}

			retrySeconds := int(math.Ceil(retryAfter.Seconds()))
			if retrySeconds < 1 {
				retrySeconds = 1
			}
			w.Header().Set("Retry-After", strconv.Itoa(retrySeconds))
			w.Header().Set("X-RateLimit-Remaining", "0")
			log.Printf("api rate limit triggered policy=%s ip=%s method=%s path=%s", policy.name, clientIP(r), r.Method, r.URL.Path)
			respondError(w, http.StatusTooManyRequests, errors.New("rate limit exceeded, please retry later"))
		})
	}
}

func clientIP(r *http.Request) string {
	for _, header := range []string{"X-Forwarded-For", "X-Real-IP"} {
		value := strings.TrimSpace(r.Header.Get(header))
		if value == "" {
			continue
		}
		if header == "X-Forwarded-For" {
			parts := strings.Split(value, ",")
			if len(parts) > 0 {
				value = strings.TrimSpace(parts[0])
			}
		}
		if value != "" {
			return value
		}
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}
	if strings.TrimSpace(r.RemoteAddr) != "" {
		return strings.TrimSpace(r.RemoteAddr)
	}
	return "unknown"
}
