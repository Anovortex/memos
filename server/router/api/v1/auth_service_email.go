package v1

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/usememos/memos/internal/email"
	"github.com/usememos/memos/internal/util"
	v1pb "github.com/usememos/memos/proto/gen/api/v1"
	storepb "github.com/usememos/memos/proto/gen/store"
	"github.com/usememos/memos/server/notification"
	"github.com/usememos/memos/store"
)

const (
	emailVerificationTTL = 24 * time.Hour
	passwordResetTTL     = time.Hour
	authTokenLength      = 32
	// invalidLinkMessage is deliberately the same for a bad user, a bad token,
	// and an expired token so links cannot be probed.
	invalidLinkMessage = "this link is invalid or has expired"
)

// authMailer is what sending an auth email needs: a valid SMTP config, the
// reply-to, and the public base URL for links.
type authMailer struct {
	config  *email.Config
	replyTo string
	baseURL string
}

// newAuthMailer resolves the mailer or returns FailedPrecondition naming what
// is missing, which the UI shows to the user.
func (s *APIV1Service) newAuthMailer(ctx context.Context) (*authMailer, error) {
	setting, err := s.Store.GetInstanceNotificationSetting(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get notification setting")
	}
	emailSetting := setting.GetEmail()
	if emailSetting == nil || !emailSetting.Enabled {
		return nil, status.Errorf(codes.FailedPrecondition, "email is not configured on this instance")
	}
	config := notification.EmailConfigFromInstanceSetting(emailSetting)
	if err := config.Validate(); err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "email is not configured on this instance")
	}
	baseURL := strings.TrimRight(strings.TrimSpace(s.Profile.InstanceURL), "/")
	if baseURL == "" {
		return nil, status.Errorf(codes.FailedPrecondition, "instance URL is not configured")
	}
	return &authMailer{config: config, replyTo: emailSetting.ReplyTo, baseURL: baseURL}, nil
}

func (s *APIV1Service) sendAuthEmail(mailer *authMailer, message *email.Message) {
	message.ReplyTo = mailer.replyTo
	sender := s.NotificationEmailSender
	if sender == nil {
		sender = email.SendAsync
	}
	sender(mailer.config, message)
}

func (m *authMailer) link(path string, user *store.User, token string) string {
	return fmt.Sprintf("%s%s?user=%s&token=%s", m.baseURL, path, url.QueryEscape(user.Username), url.QueryEscape(token))
}

// newAuthToken returns a random token for a link and its SHA-256 hex, which is
// the only form that is stored.
func newAuthToken() (string, string, error) {
	token, err := util.RandomString(authTokenLength)
	if err != nil {
		return "", "", err
	}
	return token, hashAuthToken(token), nil
}

func hashAuthToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func tokenMatches(storedHash, token string) bool {
	if storedHash == "" || token == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(storedHash), []byte(hashAuthToken(token))) == 1
}

func expired(expiresAt *timestamppb.Timestamp) bool {
	return expiresAt == nil || !time.Now().Before(expiresAt.AsTime())
}

// userWithEmailVerified converts a user for the caller's own responses and
// fills email_verified. A store failure logs and reads as unverified.
func (s *APIV1Service) userWithEmailVerified(ctx context.Context, user *store.User) *v1pb.User {
	converted := convertUserFromStore(user, user)
	verified, err := s.Store.IsEmailVerified(ctx, user)
	if err != nil {
		slog.Warn("failed to check email verification", slog.Int("userID", int(user.ID)), slog.Any("err", err))
		return converted
	}
	converted.EmailVerified = verified
	return converted
}

// sendEmailVerification issues a fresh token for the user's current email and
// sends the link. Errors are FailedPrecondition (nothing to send or not
// configured) or Internal.
func (s *APIV1Service) sendEmailVerification(ctx context.Context, user *store.User) error {
	address := normalizeEmail(user.Email)
	if address == "" {
		return status.Errorf(codes.FailedPrecondition, "add an email address to your account first")
	}
	mailer, err := s.newAuthMailer(ctx)
	if err != nil {
		return err
	}
	token, hash, err := newAuthToken()
	if err != nil {
		return status.Errorf(codes.Internal, "failed to generate token")
	}
	verification, err := s.Store.GetUserEmailVerification(ctx, user.ID)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to get email verification")
	}
	if verification == nil {
		verification = &storepb.EmailVerificationUserSetting{}
	}
	verification.PendingEmail = address
	verification.PendingTokenHash = hash
	verification.PendingExpiresAt = timestamppb.New(time.Now().Add(emailVerificationTTL))
	if err := s.Store.SetUserEmailVerification(ctx, user.ID, verification); err != nil {
		return status.Errorf(codes.Internal, "failed to save email verification")
	}

	body := []string{
		fmt.Sprintf("Hi %s,", displayName(user)),
		"",
		"Confirm this email address for your account by opening the link below.",
		"It expires in 24 hours.",
		"",
		mailer.link("/auth/verify-email", user, token),
		"",
		"If you did not create this account, you can ignore this email.",
	}
	s.sendAuthEmail(mailer, &email.Message{
		To:      []string{address},
		Subject: "[Memos] Verify your email address",
		Body:    strings.Join(body, "\n"),
	})
	return nil
}

// sendEmailVerificationBestEffort sends a verification email after sign-up or
// an email change without failing the request; a missing configuration is
// expected on many instances and only logged at debug level.
func (s *APIV1Service) sendEmailVerificationBestEffort(ctx context.Context, user *store.User) {
	err := s.sendEmailVerification(ctx, user)
	if err == nil {
		return
	}
	if status.Code(err) == codes.FailedPrecondition {
		slog.Debug("skipping verification email", slog.Int("userID", int(user.ID)), slog.String("reason", status.Convert(err).Message()))
		return
	}
	slog.Warn("failed to send verification email", slog.Int("userID", int(user.ID)), slog.Any("err", err))
}

// markEmailVerifiedBestEffort records the user's current email as verified
// without a challenge (an identity provider already proved it).
func (s *APIV1Service) markEmailVerifiedBestEffort(ctx context.Context, user *store.User) {
	address := normalizeEmail(user.Email)
	if address == "" {
		return
	}
	err := s.Store.SetUserEmailVerification(ctx, user.ID, &storepb.EmailVerificationUserSetting{
		VerifiedEmail: address,
		VerifiedAt:    timestamppb.Now(),
	})
	if err != nil {
		slog.Warn("failed to mark email verified", slog.Int("userID", int(user.ID)), slog.Any("err", err))
	}
}

func displayName(user *store.User) string {
	if strings.TrimSpace(user.Nickname) != "" {
		return user.Nickname
	}
	return user.Username
}

// SendEmailVerification emails the authenticated user a verification link.
func (s *APIV1Service) SendEmailVerification(ctx context.Context, _ *v1pb.SendEmailVerificationRequest) (*emptypb.Empty, error) {
	user, err := s.fetchCurrentUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user")
	}
	if user == nil {
		return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
	}
	if err := s.RateLimits.check(FlowVerifyUser, userRateKey(user.ID)); err != nil {
		return nil, err
	}
	verified, err := s.Store.IsEmailVerified(ctx, user)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check email verification")
	}
	if verified {
		return &emptypb.Empty{}, nil
	}
	if err := s.sendEmailVerification(ctx, user); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// VerifyEmail consumes a verification token from an emailed link.
func (s *APIV1Service) VerifyEmail(ctx context.Context, request *v1pb.VerifyEmailRequest) (*emptypb.Empty, error) {
	if err := s.RateLimits.check(FlowTokenIP, clientIP(ctx)); err != nil {
		return nil, err
	}
	user, err := ResolveUserByName(ctx, s.Store, request.GetName())
	if err != nil || user == nil {
		return nil, status.Errorf(codes.InvalidArgument, invalidLinkMessage)
	}
	verification, err := s.Store.GetUserEmailVerification(ctx, user.ID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get email verification")
	}
	if verification == nil ||
		!tokenMatches(verification.PendingTokenHash, request.GetToken()) ||
		expired(verification.PendingExpiresAt) ||
		verification.PendingEmail != normalizeEmail(user.Email) {
		return nil, status.Errorf(codes.InvalidArgument, invalidLinkMessage)
	}
	if err := s.Store.SetUserEmailVerification(ctx, user.ID, &storepb.EmailVerificationUserSetting{
		VerifiedEmail: verification.PendingEmail,
		VerifiedAt:    timestamppb.Now(),
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to save email verification")
	}
	return &emptypb.Empty{}, nil
}

// RequestPasswordReset emails a reset link when exactly one account owns the
// verified address. It always returns OK so accounts cannot be enumerated.
func (s *APIV1Service) RequestPasswordReset(ctx context.Context, request *v1pb.RequestPasswordResetRequest) (*emptypb.Empty, error) {
	address := normalizeEmail(request.GetEmail())
	if address == "" {
		return nil, status.Errorf(codes.InvalidArgument, "email is required")
	}
	if err := s.RateLimits.check(FlowResetIP, clientIP(ctx)); err != nil {
		return nil, err
	}
	if err := s.RateLimits.check(FlowResetEmail, address); err != nil {
		return nil, err
	}
	// Configuration is checked before the lookup so the answer never depends
	// on whether the account exists.
	mailer, err := s.newAuthMailer(ctx)
	if err != nil {
		return nil, err
	}

	users, err := s.Store.ListUsers(ctx, &store.FindUser{Email: &address})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find user")
	}
	if len(users) != 1 || users[0].RowStatus == store.Archived {
		return &emptypb.Empty{}, nil
	}
	user := users[0]
	verified, err := s.Store.IsEmailVerified(ctx, user)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check email verification")
	}
	if !verified {
		return &emptypb.Empty{}, nil
	}
	passwordAuthOff, err := s.passwordAuthDisabledFor(ctx, user)
	if err != nil {
		return nil, err
	}
	if passwordAuthOff {
		// A password would be unusable; stay silent like the other no-send paths.
		return &emptypb.Empty{}, nil
	}

	token, hash, err := newAuthToken()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate token")
	}
	if err := s.Store.SetUserPasswordReset(ctx, user.ID, &storepb.PasswordResetUserSetting{
		TokenHash: hash,
		ExpiresAt: timestamppb.New(time.Now().Add(passwordResetTTL)),
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to save password reset")
	}

	body := []string{
		fmt.Sprintf("Hi %s,", displayName(user)),
		"",
		"Someone asked to reset the password for this account. Open the link",
		"below to choose a new password. It expires in 1 hour.",
		"",
		mailer.link("/auth/reset-password", user, token),
		"",
		"If you did not ask for this, you can ignore this email; your password",
		"stays the same.",
	}
	s.sendAuthEmail(mailer, &email.Message{
		To:      []string{address},
		Subject: "[Memos] Reset your password",
		Body:    strings.Join(body, "\n"),
	})
	return &emptypb.Empty{}, nil
}

// ResetPassword sets a new password from a reset link and signs out every
// existing session of that user.
func (s *APIV1Service) ResetPassword(ctx context.Context, request *v1pb.ResetPasswordRequest) (*emptypb.Empty, error) {
	if err := s.RateLimits.check(FlowTokenIP, clientIP(ctx)); err != nil {
		return nil, err
	}
	if err := validatePassword(request.GetNewPassword()); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}
	user, err := ResolveUserByName(ctx, s.Store, request.GetName())
	if err != nil || user == nil {
		return nil, status.Errorf(codes.InvalidArgument, invalidLinkMessage)
	}
	reset, err := s.Store.GetUserPasswordReset(ctx, user.ID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get password reset")
	}
	if reset == nil || !tokenMatches(reset.TokenHash, request.GetToken()) || expired(reset.ExpiresAt) {
		return nil, status.Errorf(codes.InvalidArgument, invalidLinkMessage)
	}
	passwordAuthOff, err := s.passwordAuthDisabledFor(ctx, user)
	if err != nil {
		return nil, err
	}
	if passwordAuthOff {
		return nil, status.Errorf(codes.FailedPrecondition, "password sign-in is disabled on this instance")
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(request.GetNewPassword()), bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate password hash")
	}
	passwordHashStr := string(passwordHash)
	updatedTs := time.Now().Unix()
	if _, err := s.Store.UpdateUser(ctx, &store.UpdateUser{
		ID:           user.ID,
		UpdatedTs:    &updatedTs,
		PasswordHash: &passwordHashStr,
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update password")
	}
	if err := s.Store.ClearUserPasswordReset(ctx, user.ID); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to clear password reset")
	}
	// Every existing session is revoked; the new password is the only way in.
	if err := s.Store.DeleteUserSettings(ctx, &store.DeleteUserSetting{
		UserID: &user.ID,
		Key:    storepb.UserSetting_REFRESH_TOKENS,
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to revoke sessions")
	}
	return &emptypb.Empty{}, nil
}

// passwordAuthDisabledFor mirrors SignIn: password auth may be switched off for
// regular users while administrators keep it.
func (s *APIV1Service) passwordAuthDisabledFor(ctx context.Context, user *store.User) (bool, error) {
	generalSetting, err := s.Store.GetInstanceGeneralSetting(ctx)
	if err != nil {
		return false, status.Errorf(codes.Internal, "failed to get instance general setting")
	}
	return generalSetting.DisallowPasswordAuth && user.Role != store.RoleAdmin, nil
}
