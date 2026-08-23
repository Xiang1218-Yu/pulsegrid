package ratelimit

import (
	"sync"
	"time"
)

type Decision struct {
	Allowed    bool          `json:"allowed"`
	Remaining  int           `json:"remaining"`
	RetryAfter time.Duration `json:"retry_after"`
}

type Limiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	buckets map[string]bucket
}

type bucket struct {
	count int
	start time.Time
}

func New(limit int, window time.Duration) *Limiter {
	if limit < 1 {
		limit = 60
	}
	if window <= 0 {
		window = time.Minute
	}
	return &Limiter{limit: limit, window: window, buckets: map[string]bucket{}}
}

func (l *Limiter) Allow(key string, now time.Time) Decision {
	l.mu.Lock()
	defer l.mu.Unlock()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	value, ok := l.buckets[key]
	if !ok || now.Sub(value.start) >= l.window {
		l.buckets[key] = bucket{count: 1, start: now}
		return Decision{Allowed: true, Remaining: l.limit - 1}
	}
	if value.count >= l.limit {
		return Decision{Allowed: false, Remaining: 0, RetryAfter: l.window - now.Sub(value.start)}
	}
	value.count++
	l.buckets[key] = value
	return Decision{Allowed: true, Remaining: l.limit - value.count}
}

func (l *Limiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.buckets, key)
}

func (l *Limiter) ClearExpired(now time.Time) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	removed := 0
	for key, value := range l.buckets {
		if now.Sub(value.start) >= l.window {
			delete(l.buckets, key)
			removed++
		}
	}
	return removed
}

func (l *Limiter) Snapshot() map[string]Decision {
	l.mu.Lock()
	defer l.mu.Unlock()
	result := map[string]Decision{}
	now := time.Now().UTC()
	for key, value := range l.buckets {
		remaining := l.limit - value.count
		if remaining < 0 {
			remaining = 0
		}
		result[key] = Decision{Allowed: value.count < l.limit, Remaining: remaining, RetryAfter: maxDuration(l.window - now.Sub(value.start))}
	}
	return result
}

func maxDuration(value time.Duration) time.Duration {
	if value < 0 {
		return 0
	}
	return value
}
