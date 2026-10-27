// Package ratelimit slows down clients that send too many requests, using
// one token bucket per key (an IP address, an email, a user id).
package ratelimit

import (
	"math"
	"sync"
	"time"
)

// Limiter allows Burst requests at once per key, refilled at one token
// every Every. It is safe for concurrent use.
type Limiter struct {
	Burst int
	Every time.Duration

	mu      sync.Mutex
	buckets map[string]*bucket
	swept   time.Time
	now     func() time.Time // replaced in tests
}

type bucket struct {
	tokens float64
	last   time.Time
}

// New returns a limiter allowing burst requests per key, refilled at one
// every every.
func New(burst int, every time.Duration) *Limiter {
	return &Limiter{Burst: burst, Every: every, buckets: map[string]*bucket{}, now: time.Now}
}

// Allow takes a token for key. When none is left it returns false and how
// long until the next one.
func (l *Limiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.sweep(now)
	b := l.buckets[key]
	if b == nil {
		b = &bucket{tokens: float64(l.Burst), last: now}
		l.buckets[key] = b
	}
	b.tokens = math.Min(float64(l.Burst), b.tokens+float64(now.Sub(b.last))/float64(l.Every))
	b.last = now
	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}
	return false, time.Duration((1 - b.tokens) * float64(l.Every))
}

// sweep drops buckets that have refilled completely, so memory stays
// bounded by the number of recently active keys. It runs at most once a
// minute.
func (l *Limiter) sweep(now time.Time) {
	if now.Sub(l.swept) < time.Minute {
		return
	}
	l.swept = now
	full := time.Duration(l.Burst) * l.Every
	for k, b := range l.buckets {
		if now.Sub(b.last) > full {
			delete(l.buckets, k)
		}
	}
}
