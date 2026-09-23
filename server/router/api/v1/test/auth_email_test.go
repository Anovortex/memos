package test

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/usememos/memos/internal/email"
	apiv1 "github.com/usememos/memos/proto/gen/api/v1"
	storepb "github.com/usememos/memos/proto/gen/store"
	"github.com/usememos/memos/store"
)

// enableAuthEmail configures SMTP on the instance and captures outgoing mail.
func enableAuthEmail(ctx context.Context, t *testing.T, ts *TestService) *[]*email.Message {
	t.Helper()
	sent := []*email.Message{}
	ts.Service.NotificationEmailSender = func(_ *email.Config, message *email.Message) {
		sent = append(sent, message)
	}
	_, err := ts.Store.UpsertInstanceSetting(ctx, &storepb.InstanceSetting{
		Key: storepb.InstanceSettingKey_NOTIFICATION,
		Value: &storepb.InstanceSetting_NotificationSetting{
			NotificationSetting: &storepb.InstanceNotificationSetting{
				Email: &storepb.InstanceNotificationSetting_EmailSetting{
					Enabled:   true,
					SmtpHost:  "smtp.example.com",
					SmtpPort:  587,
					FromEmail: "bot@example.com",
				},
			},
		},
	})
	require.NoError(t, err)
	return &sent
}

var linkParamsRe = regexp.MustCompile(`user=([^&\s]+)&token=([^&\s]+)`)

// linkParams extracts the user and token query values from the emailed link.
func linkParams(t *testing.T, message *email.Message, path string) (string, string) {
	t.Helper()
	require.Contains(t, message.Body, "http://localhost:8080"+path)
	match := linkParamsRe.FindStringSubmatch(message.Body)
	require.Len(t, match, 3, "link with user and token in body")
	return match[1], match[2]
}

func createVerifiedPasswordUser(ctx context.Context, t *testing.T, ts *TestService, username, password string) *store.User {
	t.Helper()
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	require.NoError(t, err)
	user, err := ts.Store.CreateUser(ctx, &store.User{
		Username:     username,
		Role:         store.RoleUser,
		Email:        username + "@example.com",
		PasswordHash: string(passwordHash),
	})
	require.NoError(t, err)
	require.NoError(t, ts.Store.SetUserEmailVerification(ctx, user.ID, &storepb.EmailVerificationUserSetting{
		VerifiedEmail: user.Email,
		VerifiedAt:    timestamppb.Now(),
	}))
	return user
}

func TestEmailVerificationFlow(t *testing.T) {
	ctx := context.Background()
	ts := NewTestService(t)
	defer ts.Cleanup()
	sent := enableAuthEmail(ctx, t, ts)

	user, err := ts.CreateUnverifiedUser(ctx, "vera")
	require.NoError(t, err)
	userCtx := ts.CreateUserContext(ctx, user.ID)

	me, err := ts.Service.GetCurrentUser(userCtx, &apiv1.GetCurrentUserRequest{})
	require.NoError(t, err)
	require.False(t, me.User.EmailVerified)

	_, err = ts.Service.SendEmailVerification(userCtx, &apiv1.SendEmailVerificationRequest{})
	require.NoError(t, err)
	require.Len(t, *sent, 1)
	require.Equal(t, []string{"vera@example.com"}, (*sent)[0].To)
	userParam, token := linkParams(t, (*sent)[0], "/auth/verify-email")
	require.Equal(t, "vera", userParam)

	_, err = ts.Service.VerifyEmail(ctx, &apiv1.VerifyEmailRequest{Name: "users/vera", Token: "not-the-token"})
	require.Equal(t, codes.InvalidArgument, status.Code(err))

	_, err = ts.Service.VerifyEmail(ctx, &apiv1.VerifyEmailRequest{Name: "users/vera", Token: token})
	require.NoError(t, err)

	me, err = ts.Service.GetCurrentUser(userCtx, &apiv1.GetCurrentUserRequest{})
	require.NoError(t, err)
	require.True(t, me.User.EmailVerified)

	_, err = ts.Service.VerifyEmail(ctx, &apiv1.VerifyEmailRequest{Name: "users/vera", Token: token})
	require.Equal(t, codes.InvalidArgument, status.Code(err), "tokens are single-use")

	_, err = ts.Service.SendEmailVerification(userCtx, &apiv1.SendEmailVerificationRequest{})
	require.NoError(t, err)
	require.Len(t, *sent, 1, "already verified: nothing more is sent")
}

func TestEmailVerificationRejectsExpiredAndStaleTokens(t *testing.T) {
	ctx := context.Background()
	ts := NewTestService(t)
	defer ts.Cleanup()
	sent := enableAuthEmail(ctx, t, ts)
	user, err := ts.CreateUnverifiedUser(ctx, "stale")
	require.NoError(t, err)
	userCtx := ts.CreateUserContext(ctx, user.ID)

	_, err = ts.Service.SendEmailVerification(userCtx, &apiv1.SendEmailVerificationRequest{})
	require.NoError(t, err)
	_, token := linkParams(t, (*sent)[0], "/auth/verify-email")

	verification, err := ts.Store.GetUserEmailVerification(ctx, user.ID)
	require.NoError(t, err)
	verification.PendingExpiresAt = timestamppb.New(time.Now().Add(-time.Minute))
	require.NoError(t, ts.Store.SetUserEmailVerification(ctx, user.ID, verification))
	_, err = ts.Service.VerifyEmail(ctx, &apiv1.VerifyEmailRequest{Name: "users/stale", Token: token})
	require.Equal(t, codes.InvalidArgument, status.Code(err), "expired token")

	_, err = ts.Service.SendEmailVerification(userCtx, &apiv1.SendEmailVerificationRequest{})
	require.NoError(t, err)
	_, token = linkParams(t, (*sent)[1], "/auth/verify-email")
	newEmail := "moved@example.com"
	_, err = ts.Store.UpdateUser(ctx, &store.UpdateUser{ID: user.ID, Email: &newEmail})
	require.NoError(t, err)
	_, err = ts.Service.VerifyEmail(ctx, &apiv1.VerifyEmailRequest{Name: "users/stale", Token: token})
	require.Equal(t, codes.InvalidArgument, status.Code(err), "token for a previous address")
}

func TestEmailVerificationNeedsConfiguredEmail(t *testing.T) {
	ctx := context.Background()
	ts := NewTestService(t)
	defer ts.Cleanup()
	user, err := ts.CreateUnverifiedUser(ctx, "noconfig")
	require.NoError(t, err)

	_, err = ts.Service.SendEmailVerification(ts.CreateUserContext(ctx, user.ID), &apiv1.SendEmailVerificationRequest{})
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestSignUpSendsVerificationEmail(t *testing.T) {
	ctx := context.Background()
	ts := NewTestService(t)
	defer ts.Cleanup()
	sent := enableAuthEmail(ctx, t, ts)
	admin, err := ts.CreateHostUser(ctx, "admin")
	require.NoError(t, err)

	_, err = signUp(ctx, ts, "newbie", "Newbie@Example.com", "password123")
	require.NoError(t, err)
	require.Len(t, *sent, 1)
	require.Equal(t, []string{"newbie@example.com"}, (*sent)[0].To)
	linkParams(t, (*sent)[0], "/auth/verify-email")

	_, err = signUp(ts.CreateUserContext(ctx, admin.ID), ts, "svc", "svc@example.com", "password123")
	require.NoError(t, err)
	require.Len(t, *sent, 1, "admin-created accounts get no verification email")
}

func TestPasswordResetFlow(t *testing.T) {
	ctx := context.Background()
	ts := NewTestService(t)
	defer ts.Cleanup()
	sent := enableAuthEmail(ctx, t, ts)
	user := createVerifiedPasswordUser(ctx, t, ts, "rex", "oldpassword1")
	require.NoError(t, ts.Store.AddUserRefreshToken(ctx, user.ID, &storepb.RefreshTokensUserSetting_RefreshToken{TokenId: "old-session"}))

	_, err := ts.Service.RequestPasswordReset(ctx, &apiv1.RequestPasswordResetRequest{Email: " REX@Example.com "})
	require.NoError(t, err)
	require.Len(t, *sent, 1)
	require.Equal(t, []string{"rex@example.com"}, (*sent)[0].To)
	userParam, token := linkParams(t, (*sent)[0], "/auth/reset-password")
	require.Equal(t, "rex", userParam)

	_, err = ts.Service.ResetPassword(ctx, &apiv1.ResetPasswordRequest{Name: "users/rex", Token: "wrong", NewPassword: "newpassword1"})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	_, err = ts.Service.ResetPassword(ctx, &apiv1.ResetPasswordRequest{Name: "users/rex", Token: token, NewPassword: "short"})
	require.Equal(t, codes.InvalidArgument, status.Code(err))

	_, err = ts.Service.ResetPassword(ctx, &apiv1.ResetPasswordRequest{Name: "users/rex", Token: token, NewPassword: "newpassword1"})
	require.NoError(t, err)

	tokens, err := ts.Store.GetUserRefreshTokens(ctx, user.ID)
	require.NoError(t, err)
	require.Empty(t, tokens, "existing sessions are revoked")

	require.NoError(t, passwordSignIn(ctx, ts, "rex", "newpassword1"))
	require.Equal(t, codes.InvalidArgument, status.Code(passwordSignIn(ctx, ts, "rex", "oldpassword1")))

	_, err = ts.Service.ResetPassword(ctx, &apiv1.ResetPasswordRequest{Name: "users/rex", Token: token, NewPassword: "another-one1"})
	require.Equal(t, codes.InvalidArgument, status.Code(err), "reset tokens are single-use")
}

func TestPasswordResetIsSilentForUnknownOrUnverified(t *testing.T) {
	ctx := context.Background()
	ts := NewTestService(t)
	defer ts.Cleanup()
	sent := enableAuthEmail(ctx, t, ts)
	_, err := ts.CreateUnverifiedUser(ctx, "unverified")
	require.NoError(t, err)

	_, err = ts.Service.RequestPasswordReset(ctx, &apiv1.RequestPasswordResetRequest{Email: "nobody@example.com"})
	require.NoError(t, err)
	_, err = ts.Service.RequestPasswordReset(ctx, &apiv1.RequestPasswordResetRequest{Email: "unverified@example.com"})
	require.NoError(t, err)
	require.Empty(t, *sent)

	_, err = ts.Service.RequestPasswordReset(ctx, &apiv1.RequestPasswordResetRequest{Email: ""})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestPasswordResetNeedsConfiguredEmail(t *testing.T) {
	ctx := context.Background()
	ts := NewTestService(t)
	defer ts.Cleanup()

	_, err := ts.Service.RequestPasswordReset(ctx, &apiv1.RequestPasswordResetRequest{Email: "anyone@example.com"})
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestPasswordResetWorksForAdminWithoutVerification(t *testing.T) {
	ctx := context.Background()
	ts := NewTestService(t)
	defer ts.Cleanup()
	sent := enableAuthEmail(ctx, t, ts)
	_, err := ts.CreateHostUser(ctx, "owner")
	require.NoError(t, err)

	_, err = ts.Service.RequestPasswordReset(ctx, &apiv1.RequestPasswordResetRequest{Email: "owner@example.com"})
	require.NoError(t, err)
	require.Len(t, *sent, 1, "the instance owner can always recover the account")
}

func TestSearchMemosRequiresVerifiedEmail(t *testing.T) {
	ctx := context.Background()
	ts := NewTestService(t)
	defer ts.Cleanup()
	configureSearch(ctx, t, ts, &fixedQueryEmbedder{queryVector: []float32{1, 0}, tokens: 1})

	user, err := ts.CreateUnverifiedUser(ctx, "unverified-searcher")
	require.NoError(t, err)
	userCtx := ts.CreateUserContext(ctx, user.ID)
	_, err = ts.Service.SearchMemos(userCtx, &apiv1.SearchMemosRequest{Query: "anything"})
	require.Equal(t, codes.PermissionDenied, status.Code(err))

	require.NoError(t, ts.Store.SetUserEmailVerification(ctx, user.ID, &storepb.EmailVerificationUserSetting{
		VerifiedEmail: user.Email,
		VerifiedAt:    timestamppb.Now(),
	}))
	_, err = ts.Service.SearchMemos(userCtx, &apiv1.SearchMemosRequest{Query: "anything"})
	require.NoError(t, err)

	admin, err := ts.CreateHostUser(ctx, "owner")
	require.NoError(t, err)
	_, err = ts.Service.SearchMemos(ts.CreateUserContext(ctx, admin.ID), &apiv1.SearchMemosRequest{Query: "anything"})
	require.NoError(t, err, "admins are exempt")
}
