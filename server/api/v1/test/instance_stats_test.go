package test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	v1pb "github.com/usememos/memos/proto/gen/api/v1"
	"github.com/usememos/memos/store"
)

func TestGetInstanceStats_HappyPath(t *testing.T) {
	ctx := context.Background()
	ts := NewTestService(t)
	defer ts.Cleanup()

	admin, err := ts.CreateHostUser(ctx, "admin1")
	require.NoError(t, err)
	adminCtx := ts.CreateUserContext(ctx, admin.ID)

	resp, err := ts.Service.GetInstanceStats(adminCtx, &v1pb.GetInstanceStatsRequest{})
	require.NoError(t, err)
	require.NotNil(t, resp)

	require.NotNil(t, resp.Database)
	require.Equal(t, "sqlite", resp.Database.Driver)
	require.Greater(t, resp.Database.SizeBytes, int64(0))

	require.GreaterOrEqual(t, resp.LocalStorageBytes, int64(0))
}

func TestGetInstanceStats_UserUsage(t *testing.T) {
	ctx := context.Background()
	ts := NewTestService(t)
	defer ts.Cleanup()

	admin, err := ts.CreateHostUser(ctx, "admin1")
	require.NoError(t, err)
	user, err := ts.CreateRegularUser(ctx, "alice")
	require.NoError(t, err)
	_, err = ts.Service.CreateAttachment(ts.CreateUserContext(ctx, user.ID), &v1pb.CreateAttachmentRequest{
		Attachment: &v1pb.Attachment{Filename: "report.txt", Content: []byte("12345")},
	})
	require.NoError(t, err)
	_, err = ts.Store.CreateMemo(ctx, &store.Memo{UID: "alice-note", CreatorID: user.ID, Content: "hello", Visibility: store.Private})
	require.NoError(t, err)

	resp, err := ts.Service.GetInstanceStats(ts.CreateUserContext(ctx, admin.ID), &v1pb.GetInstanceStatsRequest{})
	require.NoError(t, err)

	var alice *v1pb.InstanceStats_UserUsage
	for _, usage := range resp.UserUsage {
		if usage.Name == "users/alice" {
			alice = usage
		}
	}
	require.NotNil(t, alice)
	require.Equal(t, int32(1), alice.MemoCount)
	require.Equal(t, int32(1), alice.AttachmentCount)
	require.Equal(t, int64(5), alice.AttachmentBytes)
	require.NotNil(t, alice.LastActivityTime)

	var adminUsage *v1pb.InstanceStats_UserUsage
	for _, usage := range resp.UserUsage {
		if usage.Name == "users/admin1" {
			adminUsage = usage
		}
	}
	require.NotNil(t, adminUsage)
	require.Zero(t, adminUsage.MemoCount)
	require.Zero(t, adminUsage.AttachmentBytes)
}

func TestGetInstanceStats_UserUsageIncludesMoreThanDefaultAttachmentLimit(t *testing.T) {
	ctx := context.Background()
	ts := NewTestService(t)
	defer ts.Cleanup()

	admin, err := ts.CreateHostUser(ctx, "admin1")
	require.NoError(t, err)
	user, err := ts.CreateRegularUser(ctx, "alice")
	require.NoError(t, err)

	const attachmentCount = 101
	for i := range attachmentCount {
		_, err := ts.Store.CreateAttachment(ctx, &store.Attachment{
			UID:       fmt.Sprintf("attachment-%03d", i),
			CreatorID: user.ID,
			Filename:  fmt.Sprintf("attachment-%03d.txt", i),
			Type:      "text/plain",
			Size:      2,
		})
		require.NoError(t, err)
	}

	resp, err := ts.Service.GetInstanceStats(ts.CreateUserContext(ctx, admin.ID), &v1pb.GetInstanceStatsRequest{})
	require.NoError(t, err)

	for _, usage := range resp.UserUsage {
		if usage.Name == "users/alice" {
			require.Equal(t, int32(attachmentCount), usage.AttachmentCount)
			require.Equal(t, int64(attachmentCount*2), usage.AttachmentBytes)
			return
		}
	}
	require.Fail(t, "alice usage not found")
}

func TestGetInstanceStats_NonAdminDenied(t *testing.T) {
	ctx := context.Background()
	ts := NewTestService(t)
	defer ts.Cleanup()

	// Need an admin to exist (otherwise instance is uninitialized).
	admin, err := ts.CreateHostUser(ctx, "admin1")
	require.NoError(t, err)
	_ = admin

	regular, err := ts.CreateRegularUser(ctx, "alice")
	require.NoError(t, err)
	regularCtx := ts.CreateUserContext(ctx, regular.ID)

	_, err = ts.Service.GetInstanceStats(regularCtx, &v1pb.GetInstanceStatsRequest{})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.PermissionDenied, st.Code())
}

func TestGetInstanceStats_SecondaryAdminDenied(t *testing.T) {
	ctx := context.Background()
	ts := NewTestService(t)
	defer ts.Cleanup()

	_, err := ts.CreateHostUser(ctx, "owner")
	require.NoError(t, err)
	secondaryAdmin, err := ts.CreateHostUser(ctx, "secondary-admin")
	require.NoError(t, err)

	_, err = ts.Service.GetInstanceStats(ts.CreateUserContext(ctx, secondaryAdmin.ID), &v1pb.GetInstanceStatsRequest{})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.PermissionDenied, st.Code())
}

func TestGetInstanceProfile_AdminRemainsOriginalOwner(t *testing.T) {
	ctx := context.Background()
	ts := NewTestService(t)
	defer ts.Cleanup()

	_, err := ts.CreateHostUser(ctx, "owner")
	require.NoError(t, err)
	_, err = ts.CreateHostUser(ctx, "secondary-admin")
	require.NoError(t, err)

	profile, err := ts.Service.GetInstanceProfile(ctx, &v1pb.GetInstanceProfileRequest{})
	require.NoError(t, err)
	require.NotNil(t, profile.Admin)
	require.Equal(t, "users/owner", profile.Admin.Name)
}

func TestGetInstanceStats_Cache(t *testing.T) {
	ctx := context.Background()
	ts := NewTestService(t)
	defer ts.Cleanup()

	admin, err := ts.CreateHostUser(ctx, "admin1")
	require.NoError(t, err)
	adminCtx := ts.CreateUserContext(ctx, admin.ID)

	first, err := ts.Service.GetInstanceStats(adminCtx, &v1pb.GetInstanceStatsRequest{})
	require.NoError(t, err)

	second, err := ts.Service.GetInstanceStats(adminCtx, &v1pb.GetInstanceStatsRequest{})
	require.NoError(t, err)

	// Cache hit: same pointer (the cache returns the stored *InstanceStats directly).
	require.Same(t, first, second)
}
