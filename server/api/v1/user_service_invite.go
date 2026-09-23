package v1

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/usememos/memos/internal/email"
	"github.com/usememos/memos/internal/plan"
	"github.com/usememos/memos/internal/ratelimit"
	v1pb "github.com/usememos/memos/proto/gen/api/v1"
	"github.com/usememos/memos/server/auth"
	"github.com/usememos/memos/store"
)

// inviteSignupPath is the frontend route that reads the invite query param.
const inviteSignupPath = "/auth/signup"

// CreateUserInvite issues a signed invite link for one email address. The
// operator may always invite; a member may invite while on an active Teams
// package. The link grants the caller's own role: operators invite operators,
// members invite members. The link is always returned so the caller can share
// it; when SMTP is configured it is also emailed to the invitee.
func (s *APIV1Service) CreateUserInvite(ctx context.Context, request *v1pb.CreateUserInviteRequest) (*v1pb.UserInvite, error) {
	currentUser, err := s.fetchCurrentUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
	}
	if currentUser == nil {
		return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
	}
	if err := s.throttleAndCharge(ratelimit.ScopeInviteUser, userKey(currentUser.ID), 1); err != nil {
		return nil, err
	}
	if currentUser.Role != store.RoleAdmin {
		pkg, err := s.Store.GetUserPackage(ctx, currentUser.ID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get user package: %v", err)
		}
		if plan.Effective(pkg, time.Now()) != plan.TierTeams {
			return nil, status.Errorf(codes.PermissionDenied, "invites are part of the Teams package")
		}
	}

	address, err := store.NormalizeEmail(request.GetEmail())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}
	if address == "" {
		return nil, status.Errorf(codes.InvalidArgument, "email is required")
	}
	// The unique index guards the real write at sign-up; this lookup only
	// spares the inviter a link that can never be redeemed.
	holder, err := s.Store.GetUser(ctx, &store.FindUser{Email: &address})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check email: %v", err)
	}
	if holder != nil {
		return nil, status.Error(codes.AlreadyExists, emailTakenMessage)
	}

	// The link needs a public base URL even when no email goes out.
	baseURL := s.instanceBaseURL()
	if baseURL == "" {
		return nil, status.Errorf(codes.FailedPrecondition, "instance URL is not configured")
	}

	token, expiresAt, err := auth.GenerateInviteToken(address, string(currentUser.Role), []byte(s.Secret))
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
	invitation := "has invited you to join Noviledger Notes."
	if currentUser.Role == store.RoleAdmin {
		invitation = "has invited you to help run Noviledger Notes as an operator."
	}
	body := []string{
		"Hi,",
		"",
		fmt.Sprintf("%s %s", displayName(currentUser), invitation),
		"Open the link below to create your account with this email address. It expires in 7 days.",
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
