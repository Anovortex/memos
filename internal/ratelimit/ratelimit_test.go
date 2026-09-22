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

func TestIdleEntriesAreEvicted(t *testing.T) {
	// A zero rate never refills, so only eviction can make the key allow again.
	l := New(rate.Limit(0), 1)
	now := time.Unix(1_700_000_000, 0)
	l.now = func() time.Time { return now }

	require.True(t, l.Allow("a"))
	require.False(t, l.Allow("a"))

	now = now.Add(idleTTL + time.Second)
	require.True(t, l.Allow("a"), "fresh bucket after the idle entry was swept")
	require.Len(t, l.entries, 1)
}

func TestNilLimiterAllowsEverything(t *testing.T) {
	var l *Limiter
	require.True(t, l.Allow("anything"))
}

func TestRateHelpers(t *testing.T) {
	require.Equal(t, rate.Every(time.Second), PerMinute(60))
	require.Equal(t, rate.Every(time.Minute), PerHour(60))
}
