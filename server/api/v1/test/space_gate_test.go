package test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	apiv1 "github.com/usememos/memos/proto/gen/api/v1"
)

// Spaces stay closed until the Teams package gates them: creating or joining
// one is refused server-side, so the hidden UI is not the only guard.
func TestSpacesClosedUntilGated(t *testing.T) {
	ctx := context.Background()
	ts := NewTestService(t)
	defer ts.Cleanup()
	ts.Profile.SpacesEnabled = false

	member, err := ts.CreateRegularUser(ctx, "member")
	require.NoError(t, err)
	memberCtx := ts.CreateUserContext(ctx, member.ID)

	_, err = ts.Service.CreateSpace(memberCtx, &apiv1.CreateSpaceRequest{})
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
	_, err = ts.Service.CreateSpaceInvitation(memberCtx, &apiv1.CreateSpaceInvitationRequest{})
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
	_, err = ts.Service.AcceptSpaceInvitation(memberCtx, &apiv1.AcceptSpaceInvitationRequest{})
	require.Equal(t, codes.FailedPrecondition, status.Code(err))

	// Anonymous callers are still told to sign in first.
	_, err = ts.Service.CreateSpace(ctx, &apiv1.CreateSpaceRequest{})
	require.Equal(t, codes.Unauthenticated, status.Code(err))
}
