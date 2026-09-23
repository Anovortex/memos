package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"
)

// InviteClaims contains claims for invite tokens.
//
// An invite is a signed link with no server-side storage: it names the email
// address an administrator invited, and CreateUser accepts a sign-up for
// exactly that address while it is valid.
type InviteClaims struct {
	Type  string `json:"type"`  // "invite"
	Email string `json:"email"` // Invited email address (normalized)
	jwt.RegisteredClaims
}

// GenerateInviteToken signs an invite for the given normalized email address.
// It returns the token and its expiration time.
func GenerateInviteToken(email string, secret []byte) (string, time.Time, error) {
	if email == "" {
		return "", time.Time{}, errors.New("email is required")
	}
	expiresAt := time.Now().Add(InviteTokenDuration)

	claims := &InviteClaims{
		Type:  "invite",
		Email: email,
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
	return claims, nil
}
