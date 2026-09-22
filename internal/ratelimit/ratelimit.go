// Package ratelimit is a small in-process, per-key token-bucket limiter used
// to bound sign-in attempts, sign-ups, and AI calls.
//
// ponytail: single node, memory only; swap the map for a shared store if we
// ever run more than one replica.
package ratelimit

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const (
	// idleTTL is how long an unused key keeps its bucket before it is dropped.
	idleTTL = 10 * time.Minute
	// sweepEvery bounds how often Allow scans for idle keys.
	sweepEvery = time.Minute
)

// Limiter holds one token bucket per key. A nil *Limiter allows everything,
// which is how tests and disabled deployments opt out.
type Limiter struct {
	mu        sync.Mutex
	limit     rate.Limit
	burst     int
	entries   map[string]*entry
	lastSweep time.Time
	// now is a test seam.
	now func() time.Time
}

type entry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// New creates a limiter refilling at limit with the given burst per key.
func New(limit rate.Limit, burst int) *Limiter {
	return &Limiter{
		limit:   limit,
		burst:   burst,
		entries: map[string]*entry{},
		now:     time.Now,
	}
}

// PerMinute returns a rate of n events per minute.
func PerMinute(n int) rate.Limit {
	return rate.Every(time.Minute / time.Duration(n))
}

// PerHour returns a rate of n events per hour.
func PerHour(n int) rate.Limit {
	return rate.Every(time.Hour / time.Duration(n))
}

// Allow reports whether one event for key fits within its budget right now.
func (l *Limiter) Allow(key string) bool {
	if l == nil {
		return true
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.sweepLocked(now)

	e, ok := l.entries[key]
	if !ok {
		e = &entry{limiter: rate.NewLimiter(l.limit, l.burst)}
		l.entries[key] = e
	}
	e.lastSeen = now
	return e.limiter.AllowN(now, 1)
}

// sweepLocked drops keys idle for longer than idleTTL, at most once per sweepEvery.
func (l *Limiter) sweepLocked(now time.Time) {
	if now.Sub(l.lastSweep) < sweepEvery {
		return
	}
	l.lastSweep = now
	for key, e := range l.entries {
		if now.Sub(e.lastSeen) > idleTTL {
			delete(l.entries, key)
		}
	}
}
