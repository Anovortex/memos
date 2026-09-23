package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"
)

// Roles an invite can grant. They mirror the store's user roles; the role
// follows the inviter (operators invite operators, members invite members)
// and is signed into the link so it cannot be changed on the way.
const (
	InviteRoleAdmin = "ADMIN"
	InviteRoleUser  = "USER"
)

// InviteClaims contains claims for invite tokens.
//
// An invite is a signed link with no server-side storage: it names the email
// address that was invited and the role the account gets, and CreateUser
// accepts a sign-up for exactly that address while the link is valid.
type InviteClaims struct {
	Type  string `json:"type"`           // "invite"
	Email string `json:"email"`          // Invited email address (normalized)
	Role  string `json:"role,omitempty"` // InviteRoleAdmin or InviteRoleUser; empty on links issued before roles existed
	jwt.RegisteredClaims
}

func validInviteRole(role string) bool {
	return role == InviteRoleAdmin || role == InviteRoleUser
}

// GenerateInviteToken signs an invite for the given normalized email address
// and role. It returns the token and its expiration time.
func GenerateInviteToken(email, role string, secret []byte) (string, time.Time, error) {
	if email == "" {
		return "", time.Time{}, errors.New("email is required")
	}
	if !validInviteRole(role) {
		return "", time.Time{}, errors.Errorf("invalid invite role %q", role)
	}
	expiresAt := time.Now().Add(InviteTokenDuration)

	claims := &InviteClaims{
		Type:  "invite",
		Email: email,
		Role:  role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    Issuer,
			Audience:  jwt.ClaimStrings{InviteTokenAudienceName},
			Subject:   email,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["kid"] = KeyID

	tokenString, err := token.SignedString(secret)
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expiresAt, nil
}

// ParseInviteToken parses and validates an invite token. Access and refresh
// tokens are rejected by audience and type, so a session token can never be
// presented as an invite.
func ParseInviteToken(tokenString string, secret []byte) (*InviteClaims, error) {
	claims := &InviteClaims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, verifyJWTKeyFunc(secret),
		jwt.WithIssuer(Issuer),
		jwt.WithAudience(InviteTokenAudienceName),
	)
	if err != nil {
		return nil, err
	}
	if claims.Type != "invite" {
		return nil, errors.New("invalid token type: expected invite token")
	}
	if claims.Email == "" {
		return nil, errors.New("invalid invite token: missing email")
	}
	// Links issued before roles were signed in were only ever member invites.
	if claims.Role == "" {
		claims.Role = InviteRoleUser
	}
	if !validInviteRole(claims.Role) {
		return nil, errors.Errorf("invalid invite token: unknown role %q", claims.Role)
	}
	return claims, nil
}
