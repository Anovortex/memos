package test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/usememos/memos/store"
)

func TestUserAIUsageEmptyIsZero(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	ts := NewTestingStore(ctx, t)
	defer ts.Close()
	user, err := createTestingHostUser(ctx, ts)
	require.NoError(t, err)

	tokens, err := ts.GetUserAIUsage(ctx, user.ID, "2026-08-28")
	require.NoError(t, err)
	require.Equal(t, int64(0), tokens)
}

func TestUserAIUsageIncrementAccumulates(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	ts := NewTestingStore(ctx, t)
	defer ts.Close()
	user, err := createTestingHostUser(ctx, ts)
	require.NoError(t, err)

	require.NoError(t, ts.IncrementUserAIUsage(ctx, user.ID, "2026-08-28", 100))
	require.NoError(t, ts.IncrementUserAIUsage(ctx, user.ID, "2026-08-28", 50))

	tokens, err := ts.GetUserAIUsage(ctx, user.ID, "2026-08-28")
	require.NoError(t, err)
	require.Equal(t, int64(150), tokens)
}

func TestUserAIUsageIsolatedByDateAndUser(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	ts := NewTestingStore(ctx, t)
	defer ts.Close()
	userA, err := createTestingHostUser(ctx, ts)
	require.NoError(t, err)
	userB, err := createTestingUserWithRole(ctx, ts, "usage-user-b", store.RoleUser)
	require.NoError(t, err)

	require.NoError(t, ts.IncrementUserAIUsage(ctx, userA.ID, "2026-08-28", 100))
	require.NoError(t, ts.IncrementUserAIUsage(ctx, userA.ID, "2026-08-29", 7))
	require.NoError(t, ts.IncrementUserAIUsage(ctx, userB.ID, "2026-08-28", 3))

	tokens, err := ts.GetUserAIUsage(ctx, userA.ID, "2026-08-28")
	require.NoError(t, err)
	require.Equal(t, int64(100), tokens)

	tokens, err = ts.GetUserAIUsage(ctx, userA.ID, "2026-08-29")
	require.NoError(t, err)
	require.Equal(t, int64(7), tokens)

	tokens, err = ts.GetUserAIUsage(ctx, userB.ID, "2026-08-28")
	require.NoError(t, err)
	require.Equal(t, int64(3), tokens)
}
