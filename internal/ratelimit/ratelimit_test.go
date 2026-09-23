package ratelimit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func TestAllowRespectsBurstPerKey(t *testing.T) {
	l := New(rate.Every(time.Hour), 2)

	require.True(t, l.Allow("a"))
	require.True(t, l.Allow("a"))
	require.False(t, l.Allow("a"), "third call within the window is denied")

	require.True(t, l.Allow("b"), "keys are independent")
}

func TestAllowRefillsOverTime(t *testing.T) {
	l := New(PerMinute(60), 1) // one token per second
	now := time.Unix(1_700_000_000, 0)
	l.now = func() time.Time { return now }

	require.True(t, l.Allow("a"))
	require.False(t, l.Allow("a"))
	now = now.Add(time.Second)
	require.True(t, l.Allow("a"))
}

func TestIdleEntriesAreEvictedOnlyAfterRefill(t *testing.T) {
	l := New(PerHour(3), 3) // refills fully in an hour
	require.Equal(t, time.Hour, l.ttl, "ttl covers the refill window")
	now := time.Unix(1_700_000_000, 0)
	l.now = func() time.Time { return now }

	for range 3 {
		require.True(t, l.Allow("a"))
	}
	require.False(t, l.Allow("a"))

	// Well past the old 10-minute idle window, but the bucket is still nearly
	// empty, so the entry must survive and keep denying.
	now = now.Add(15 * time.Minute)
	require.False(t, l.Allow("a"), "an idle gap shorter than the refill window grants no fresh burst")

	now = now.Add(time.Hour + time.Minute)
	require.True(t, l.Allow("b"))
	_, stillThere := l.entries["a"]
	require.False(t, stillThere, "entry idle for longer than ttl is swept")
	require.Len(t, l.entries, 1)
}

func TestFastLimitersKeepMinimumTTL(t *testing.T) {
	require.Equal(t, minIdleTTL, New(PerMinute(60), 1).ttl)
}

func TestEntryCapEvictsToBoundMemory(t *testing.T) {
	l := New(PerMinute(60), 1)
	l.maxEntries = 2
	for _, key := range []string{"a", "b", "c", "d"} {
		l.Allow(key)
	}
	require.LessOrEqual(t, len(l.entries), 2)
}

func TestNilLimiterAllowsEverything(t *testing.T) {
	var l *Limiter
	require.True(t, l.Allow("anything"))
}

func TestRateHelpers(t *testing.T) {
	require.Equal(t, rate.Every(time.Second), PerMinute(60))
	require.Equal(t, rate.Every(time.Minute), PerHour(60))
}
