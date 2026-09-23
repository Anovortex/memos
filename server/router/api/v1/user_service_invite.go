package v1

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/usememos/memos/internal/email"
	v1pb "github.com/usememos/memos/proto/gen/api/v1"
	"github.com/usememos/memos/server/auth"
	"github.com/usememos/memos/store"
)

// inviteSignupPath is the frontend route that reads the invite query param.
const inviteSignupPath = "/auth/signup"

// CreateUserInvite issues a signed invite link for one email address.
// Administrators only. The link is always returned so the operator can share
// it; when SMTP is configured it is also emailed to the invitee.
func (s *APIV1Service) CreateUserInvite(ctx context.Context, request *v1pb.CreateUserInviteRequest) (*v1pb.UserInvite, error) {
	currentUser, err := s.fetchCurrentUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
	}
	if currentUser == nil {
		return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
	}
	if currentUser.Role != store.RoleAdmin {
		return nil, status.Errorf(codes.PermissionDenied, "permission denied")
	}

	address := normalizeEmail(request.GetEmail())
	if address == "" {
		return nil, status.Errorf(codes.InvalidArgument, "email is required")
	}
	if err := validateEmailAddress(address); err != nil {
		return nil, err
	}
	if err := s.checkEmailAvailable(ctx, address, nil); err != nil {
		return nil, err
	}

	// The link needs a public base URL even when no email goes out.
	baseURL := s.instanceBaseURL()
	if baseURL == "" {
		return nil, status.Errorf(codes.FailedPrecondition, "instance URL is not configured")
	}

	token, expiresAt, err := auth.GenerateInviteToken(address, []byte(s.Secret))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate invite token")
	}
	invite := &v1pb.UserInvite{
		Email:      address,
		Token:      token,
		Link:       fmt.Sprintf("%s%s?invite=%s", baseURL, inviteSignupPath, url.QueryEscape(token)),
		ExpireTime: timestamppb.New(expiresAt),
	}

	// Email is optional: without SMTP the operator copies the link.
	config, replyTo, err := s.authEmailConfig(ctx)
	if err != nil {
		if status.Code(err) == codes.FailedPrecondition {
			return invite, nil
		}
		return nil, err
	}
	body := []string{
		"Hi,",
		"",
		fmt.Sprintf("%s runs Noviledger Notes and has invited you to create an account", displayName(currentUser)),
		"with this email address. Open the link below to sign up. It expires in 7 days.",
		"",
		invite.Link,
		"",
		"If you were not expecting this, you can ignore this email.",
	}
	s.sendAuthEmail(&authMailer{config: config, replyTo: replyTo, baseURL: baseURL}, &email.Message{
		To:      []string{address},
		Subject: "[Memos] You're invited to Noviledger Notes",
		Body:    strings.Join(body, "\n"),
	})
	invite.EmailSent = true
	return invite, nil
}
