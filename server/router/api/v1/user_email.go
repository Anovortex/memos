package v1

import (
	"context"
	"net/mail"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/usememos/memos/store"
)

// normalizeEmail trims and lowercases an address so lookups and uniqueness are
// case-insensitive. The empty string stays empty.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// validateEmailAddress rejects a non-empty address that does not parse as a
// bare address. Display-name forms like "Admin <admin@example.com>" parse but
// are refused so the stored value is always the canonical address.
func validateEmailAddress(email string) error {
	if email == "" {
		return nil
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return status.Errorf(codes.InvalidArgument, "invalid email address")
	}
	return nil
}

// checkEmailAvailable returns AlreadyExists when another user (excluding
// excludeUserID, if set) already has the normalized address.
// ponytail: API-level uniqueness; add a DB unique index when we take upstream's
// unique-email migration.
func (s *APIV1Service) checkEmailAvailable(ctx context.Context, email string, excludeUserID *int32) error {
	if email == "" {
		return nil
	}
	users, err := s.Store.ListUsers(ctx, &store.FindUser{Email: &email})
	if err != nil {
		return status.Errorf(codes.Internal, "failed to check email: %v", err)
	}
	for _, user := range users {
		if excludeUserID != nil && user.ID == *excludeUserID {
			continue
		}
		return status.Errorf(codes.AlreadyExists, "email is already in use")
	}
	return nil
}

// checkUsernameAvailable returns AlreadyExists when the username is taken.
func (s *APIV1Service) checkUsernameAvailable(ctx context.Context, username string) error {
	existing, err := s.Store.GetUser(ctx, &store.FindUser{Username: &username})
	if err != nil {
		return status.Errorf(codes.Internal, "failed to check username: %v", err)
	}
	if existing != nil {
		return status.Errorf(codes.AlreadyExists, "username is already taken")
	}
	return nil
}
