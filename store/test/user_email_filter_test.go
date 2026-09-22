package test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/usememos/memos/store"
)

// FindUser.Email must match case-insensitively: legacy rows may hold mixed-case
// addresses while the API normalizes new ones to lowercase.
func TestFindUserByEmailIsCaseInsensitive(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	ts := NewTestingStore(ctx, t)
	defer ts.Close()

	created, err := ts.CreateUser(ctx, &store.User{
		Username:     "mixed",
		Role:         store.RoleUser,
		Email:        "Mixed.Case@Example.com",
		PasswordHash: "x",
	})
	require.NoError(t, err)

	for _, probe := range []string{"mixed.case@example.com", "MIXED.CASE@EXAMPLE.COM", "Mixed.Case@Example.com"} {
		email := probe
		found, err := ts.ListUsers(ctx, &store.FindUser{Email: &email})
		require.NoError(t, err, probe)
		require.Len(t, found, 1, probe)
		require.Equal(t, created.ID, found[0].ID, probe)
	}

	other := "nobody@example.com"
	found, err := ts.ListUsers(ctx, &store.FindUser{Email: &other})
	require.NoError(t, err)
	require.Empty(t, found)
}
