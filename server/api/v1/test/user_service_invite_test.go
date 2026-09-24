package test

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	apiv1 "github.com/usememos/memos/proto/gen/api/v1"
	storepb "github.com/usememos/memos/proto/gen/store"
	"github.com/usememos/memos/server/auth"
)

// closeRegistration switches self sign-up off on the instance.
func closeRegistration(ctx context.Context, t *testing.T, ts *TestService) {
	t.Helper()
	_, err := ts.Store.UpsertInstanceSetting(ctx, &storepb.InstanceSetting{
		Key: storepb.InstanceSettingKey_GENERAL,
		Value: &storepb.InstanceSetting_GeneralSetting{
			GeneralSetting: &storepb.InstanceGeneralSetting{DisallowUserRegistration: true},
		},
	})
	require.NoError(t, err)
}

// grantTeams puts a user on the Teams package, lapsing at expire (nil = never).
func grantTeams(ctx context.Context, t *testing.T, ts *TestService, userID int32, expire *timestamppb.Timestamp) {
	t.Helper()
	_, err := ts.Store.UpsertUserSetting(ctx, &storepb.UserSetting{
		UserId: userID,
		Key:    storepb.UserSetting_PACKAGE,
		Value: &storepb.UserSetting_Package{
			Package: &storepb.PackageUserSetting{Plan: storepb.PackageUserSetting_TEAMS, ExpireTime: expire},
		},
	})
	require.NoError(t, err)
}

func inviteTokenFor(t *testing.T, ts *TestService, email, role string) string {
	t.Helper()
	token, _, err := auth.GenerateInviteToken(email, role, []byte(ts.Secret))
	require.NoError(t, err)
	return token
}

func signUpWithInvite(ctx context.Context, ts *TestService, username, email, password, inviteToken string) (*apiv1.User, error) {
	return ts.Service.CreateUser(ctx, &apiv1.CreateUserRequest{
		User: &apiv1.User{
			Username: username,
			Email:    email,
			Password: password,
		},
		InviteToken: inviteToken,
	})
}

func TestInvitedSignUp(t *testing.T) {
	ctx := context.Background()

	t.Run("closed registration rejects a sign-up without an invite", func(t *testing.T) {
		ts := NewTestService(t)
		defer ts.Cleanup()
		_, err := ts.CreateHostUser(ctx, "admin")
		require.NoError(t, err)
		closeRegistration(ctx, t, ts)

		_, err = signUp(ctx, ts, "alice", "alice@example.com", "password123")
		require.Equal(t, codes.PermissionDenied, status.Code(err))
	})

	t.Run("closed registration accepts a valid invite and creates a USER", func(t *testing.T) {
		ts := NewTestService(t)
		defer ts.Cleanup()
		_, err := ts.CreateHostUser(ctx, "admin")
		require.NoError(t, err)
		closeRegistration(ctx, t, ts)

		token := inviteTokenFor(t, ts, "alice@example.com", auth.InviteRoleUser)
		user, err := signUpWithInvite(ctx, ts, "alice", "Alice@Example.com", "password123", token)
		require.NoError(t, err)
		require.Equal(t, apiv1.User_USER, user.Role)
		require.Equal(t, "alice@example.com", user.Email)
	})

	t.Run("invite for a different email is rejected", func(t *testing.T) {
		ts := NewTestService(t)
		defer ts.Cleanup()
		_, err := ts.CreateHostUser(ctx, "admin")
		require.NoError(t, err)
		closeRegistration(ctx, t, ts)

		token := inviteTokenFor(t, ts, "alice@example.com", auth.InviteRoleUser)
		_, err = signUpWithInvite(ctx, ts, "bob", "bob@example.com", "password123", token)
		require.Equal(t, codes.PermissionDenied, status.Code(err))
		require.Contains(t, err.Error(), "different email")
	})

	t.Run("tampered invite is rejected", func(t *testing.T) {
		ts := NewTestService(t)
		defer ts.Cleanup()
		_, err := ts.CreateHostUser(ctx, "admin")
		require.NoError(t, err)
		closeRegistration(ctx, t, ts)

		parts := strings.Split(inviteTokenFor(t, ts, "alice@example.com", auth.InviteRoleUser), ".")
		require.Len(t, parts, 3)
		parts[2] = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
		_, err = signUpWithInvite(ctx, ts, "alice", "alice@example.com", "password123", strings.Join(parts, "."))
		require.Equal(t, codes.PermissionDenied, status.Code(err))
		require.Contains(t, err.Error(), "invalid or has expired")
	})

	t.Run("expired invite is rejected", func(t *testing.T) {
		ts := NewTestService(t)
		defer ts.Cleanup()
		_, err := ts.CreateHostUser(ctx, "admin")
		require.NoError(t, err)
		closeRegistration(ctx, t, ts)

		claims := &auth.InviteClaims{
			Type:  "invite",
			Email: "alice@example.com",
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:    auth.Issuer,
				Audience:  jwt.ClaimStrings{auth.InviteTokenAudienceName},
				Subject:   "alice@example.com",
				IssuedAt:  jwt.NewNumericDate(time.Now().Add(-8 * 24 * time.Hour)),
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
			},
		}
		expired := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		expired.Header["kid"] = auth.KeyID
		token, err := expired.SignedString([]byte(ts.Secret))
		require.NoError(t, err)

		_, err = signUpWithInvite(ctx, ts, "alice", "alice@example.com", "password123", token)
		require.Equal(t, codes.PermissionDenied, status.Code(err))
		require.Contains(t, err.Error(), "invalid or has expired")
	})

	t.Run("an operator invite creates an ADMIN on a closed instance", func(t *testing.T) {
		ts := NewTestService(t)
		defer ts.Cleanup()
		_, err := ts.CreateHostUser(ctx, "admin")
		require.NoError(t, err)
		closeRegistration(ctx, t, ts)

		token := inviteTokenFor(t, ts, "op@example.com", auth.InviteRoleAdmin)
		user, err := signUpWithInvite(ctx, ts, "op", "op@example.com", "password123", token)
		require.NoError(t, err)
		require.Equal(t, apiv1.User_ADMIN, user.Role)
	})

	t.Run("invite does not bypass password sign-up being off, even for an operator", func(t *testing.T) {
		ts := NewTestService(t)
		defer ts.Cleanup()
		_, err := ts.CreateHostUser(ctx, "admin")
		require.NoError(t, err)
		_, err = ts.Store.UpsertInstanceSetting(ctx, &storepb.InstanceSetting{
			Key: storepb.InstanceSettingKey_GENERAL,
			Value: &storepb.InstanceSetting_GeneralSetting{
				GeneralSetting: &storepb.InstanceGeneralSetting{DisallowUserRegistration: true, DisallowPasswordAuth: true},
			},
		})
		require.NoError(t, err)

		token := inviteTokenFor(t, ts, "alice@example.com", auth.InviteRoleAdmin)
		_, err = signUpWithInvite(ctx, ts, "alice", "alice@example.com", "password123", token)
		require.Equal(t, codes.PermissionDenied, status.Code(err))
		require.Contains(t, err.Error(), "password signup is not allowed")
	})

	t.Run("invite also works when registration is open", func(t *testing.T) {
		ts := NewTestService(t)
		defer ts.Cleanup()
		_, err := ts.CreateHostUser(ctx, "admin")
		require.NoError(t, err)

		token := inviteTokenFor(t, ts, "alice@example.com", auth.InviteRoleUser)
		user, err := signUpWithInvite(ctx, ts, "alice", "alice@example.com", "password123", token)
		require.NoError(t, err)
		require.Equal(t, apiv1.User_USER, user.Role)
	})

	t.Run("validate_only checks the invite too", func(t *testing.T) {
		ts := NewTestService(t)
		defer ts.Cleanup()
		_, err := ts.CreateHostUser(ctx, "admin")
		require.NoError(t, err)
		closeRegistration(ctx, t, ts)

		_, err = ts.Service.CreateUser(ctx, &apiv1.CreateUserRequest{
			User:         &apiv1.User{Username: "alice", Email: "alice@example.com", Password: "password123"},
			InviteToken:  "not-a-token",
			ValidateOnly: true,
		})
		require.Equal(t, codes.PermissionDenied, status.Code(err))

		preview, err := ts.Service.CreateUser(ctx, &apiv1.CreateUserRequest{
			User:         &apiv1.User{Username: "alice", Email: "alice@example.com", Password: "password123"},
			InviteToken:  inviteTokenFor(t, ts, "alice@example.com", auth.InviteRoleUser),
			ValidateOnly: true,
		})
		require.NoError(t, err)
		require.Equal(t, apiv1.User_USER, preview.Role)
	})
}

func TestCreateUserInvite(t *testing.T) {
	ctx := context.Background()

	t.Run("anonymous caller is Unauthenticated", func(t *testing.T) {
		ts := NewTestService(t)
		defer ts.Cleanup()

		_, err := ts.Service.CreateUserInvite(ctx, &apiv1.CreateUserInviteRequest{Email: "alice@example.com"})
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("member on Free is PermissionDenied", func(t *testing.T) {
		ts := NewTestService(t)
		defer ts.Cleanup()
		member, err := ts.CreateRegularUser(ctx, "member")
		require.NoError(t, err)

		_, err = ts.Service.CreateUserInvite(ts.CreateUserContext(ctx, member.ID), &apiv1.CreateUserInviteRequest{Email: "friend@example.com"})
		require.Equal(t, codes.PermissionDenied, status.Code(err))
		require.Contains(t, err.Error(), "Teams")
	})

	t.Run("member whose Teams package lapsed is PermissionDenied", func(t *testing.T) {
		ts := NewTestService(t)
		defer ts.Cleanup()
		member, err := ts.CreateRegularUser(ctx, "member")
		require.NoError(t, err)
		grantTeams(ctx, t, ts, member.ID, timestamppb.New(time.Now().Add(-time.Minute)))

		_, err = ts.Service.CreateUserInvite(ts.CreateUserContext(ctx, member.ID), &apiv1.CreateUserInviteRequest{Email: "friend@example.com"})
		require.Equal(t, codes.PermissionDenied, status.Code(err))
	})

	t.Run("member on Teams gets a member invite", func(t *testing.T) {
		ts := NewTestService(t)
		defer ts.Cleanup()
		member, err := ts.CreateRegularUser(ctx, "member")
		require.NoError(t, err)
		grantTeams(ctx, t, ts, member.ID, timestamppb.New(time.Now().Add(24*time.Hour)))

		invite, err := ts.Service.CreateUserInvite(ts.CreateUserContext(ctx, member.ID), &apiv1.CreateUserInviteRequest{Email: "friend@example.com"})
		require.NoError(t, err)
		claims, err := auth.ParseInviteToken(invite.Token, []byte(ts.Secret))
		require.NoError(t, err)
		require.Equal(t, auth.InviteRoleUser, claims.Role)

		closeRegistration(ctx, t, ts)
		user, err := signUpWithInvite(ctx, ts, "friend", "friend@example.com", "password123", invite.Token)
		require.NoError(t, err)
		require.Equal(t, apiv1.User_USER, user.Role)
	})

	t.Run("operator gets an operator invite whose token parses back to the email", func(t *testing.T) {
		ts := NewTestService(t)
		defer ts.Cleanup()
		admin, err := ts.CreateHostUser(ctx, "admin")
		require.NoError(t, err)
		adminCtx := ts.CreateUserContext(ctx, admin.ID)

		invite, err := ts.Service.CreateUserInvite(adminCtx, &apiv1.CreateUserInviteRequest{Email: " Alice@Example.com "})
		require.NoError(t, err)
		require.Equal(t, "alice@example.com", invite.Email)
		require.False(t, invite.EmailSent, "no SMTP configured")
		require.True(t, strings.HasPrefix(invite.Link, "http://localhost:8080/auth/signup?invite="), invite.Link)
		require.WithinDuration(t, time.Now().Add(auth.InviteTokenDuration), invite.ExpireTime.AsTime(), time.Minute)

		parsed, err := url.Parse(invite.Link)
		require.NoError(t, err)
		require.Equal(t, invite.Token, parsed.Query().Get("invite"))
		claims, err := auth.ParseInviteToken(invite.Token, []byte(ts.Secret))
		require.NoError(t, err)
		require.Equal(t, "alice@example.com", claims.Email)
		require.Equal(t, auth.InviteRoleAdmin, claims.Role)

		// The link works end to end on a closed instance and creates an operator.
		closeRegistration(ctx, t, ts)
		user, err := signUpWithInvite(ctx, ts, "alice", "alice@example.com", "password123", parsed.Query().Get("invite"))
		require.NoError(t, err)
		require.Equal(t, apiv1.User_ADMIN, user.Role)
	})

	t.Run("invalid or missing email is InvalidArgument", func(t *testing.T) {
		ts := NewTestService(t)
		defer ts.Cleanup()
		admin, err := ts.CreateHostUser(ctx, "admin")
		require.NoError(t, err)
		adminCtx := ts.CreateUserContext(ctx, admin.ID)

		_, err = ts.Service.CreateUserInvite(adminCtx, &apiv1.CreateUserInviteRequest{Email: ""})
		require.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = ts.Service.CreateUserInvite(adminCtx, &apiv1.CreateUserInviteRequest{Email: "not-an-email"})
		require.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("taken email is AlreadyExists", func(t *testing.T) {
		ts := NewTestService(t)
		defer ts.Cleanup()
		admin, err := ts.CreateHostUser(ctx, "admin")
		require.NoError(t, err)
		_, err = ts.CreateRegularUser(ctx, "taken")
		require.NoError(t, err)

		_, err = ts.Service.CreateUserInvite(ts.CreateUserContext(ctx, admin.ID), &apiv1.CreateUserInviteRequest{Email: "taken@example.com"})
		require.Equal(t, codes.AlreadyExists, status.Code(err))
	})

	t.Run("missing instance URL is FailedPrecondition", func(t *testing.T) {
		ts := NewTestService(t)
		defer ts.Cleanup()
		admin, err := ts.CreateHostUser(ctx, "admin")
		require.NoError(t, err)
		ts.Profile.InstanceURL = ""

		_, err = ts.Service.CreateUserInvite(ts.CreateUserContext(ctx, admin.ID), &apiv1.CreateUserInviteRequest{Email: "alice@example.com"})
		require.Equal(t, codes.FailedPrecondition, status.Code(err))
	})

	t.Run("with SMTP the invitee gets one email containing the link", func(t *testing.T) {
		ts := NewTestService(t)
		defer ts.Cleanup()
		sent := enableAuthEmail(ctx, t, ts)
		admin, err := ts.CreateHostUser(ctx, "admin")
		require.NoError(t, err)

		invite, err := ts.Service.CreateUserInvite(ts.CreateUserContext(ctx, admin.ID), &apiv1.CreateUserInviteRequest{Email: "alice@example.com"})
		require.NoError(t, err)
		require.True(t, invite.EmailSent)
		require.Len(t, *sent, 1)
		require.Equal(t, []string{"alice@example.com"}, (*sent)[0].To)
		require.Contains(t, (*sent)[0].Subject, "invited")
		require.Contains(t, (*sent)[0].Body, invite.Link)
		require.Contains(t, (*sent)[0].Body, "admin")
		require.Contains(t, (*sent)[0].Body, "as an operator")
	})
}
