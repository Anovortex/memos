package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// signInvite signs otherwise valid invite claims with the given role verbatim.
func signInvite(t *testing.T, secret []byte, email, role string) string {
	t.Helper()
	claims := &InviteClaims{
		Type:  "invite",
		Email: email,
		Role:  role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    Issuer,
			Audience:  jwt.ClaimStrings{InviteTokenAudienceName},
			Subject:   email,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["kid"] = KeyID
	tokenString, err := token.SignedString(secret)
	require.NoError(t, err)
	return tokenString
}

func TestInviteToken(t *testing.T) {
	secret := []byte("test-secret")

	t.Run("round trip", func(t *testing.T) {
		token, expiresAt, err := GenerateInviteToken("invited@example.com", InviteRoleUser, secret)
		require.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.True(t, expiresAt.After(time.Now().Add(InviteTokenDuration-time.Minute)))
		assert.True(t, expiresAt.Before(time.Now().Add(InviteTokenDuration+time.Minute)))

		claims, err := ParseInviteToken(token, secret)
		require.NoError(t, err)
		assert.Equal(t, "invite", claims.Type)
		assert.Equal(t, "invited@example.com", claims.Email)
		assert.Equal(t, "invited@example.com", claims.Subject)
		assert.Equal(t, InviteRoleUser, claims.Role)
		assert.Equal(t, jwt.ClaimStrings{InviteTokenAudienceName}, claims.Audience)
	})

	t.Run("carries the operator role", func(t *testing.T) {
		token, _, err := GenerateInviteToken("op@example.com", InviteRoleAdmin, secret)
		require.NoError(t, err)

		claims, err := ParseInviteToken(token, secret)
		require.NoError(t, err)
		assert.Equal(t, InviteRoleAdmin, claims.Role)
	})

	t.Run("rejects an unknown role at issue time", func(t *testing.T) {
		_, _, err := GenerateInviteToken("x@example.com", "OWNER", secret)
		assert.Error(t, err)
	})

	t.Run("treats a link without a role as a member invite", func(t *testing.T) {
		claims, err := ParseInviteToken(signInvite(t, secret, "old@example.com", ""), secret)
		require.NoError(t, err)
		assert.Equal(t, InviteRoleUser, claims.Role)
	})

	t.Run("rejects a signed link with an unknown role", func(t *testing.T) {
		_, err := ParseInviteToken(signInvite(t, secret, "x@example.com", "OWNER"), secret)
		assert.Error(t, err)
	})

	t.Run("requires an email", func(t *testing.T) {
		_, _, err := GenerateInviteToken("", InviteRoleUser, secret)
		assert.Error(t, err)
	})

	t.Run("fails when expired", func(t *testing.T) {
		claims := &InviteClaims{
			Type:  "invite",
			Email: "late@example.com",
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:    Issuer,
				Audience:  jwt.ClaimStrings{InviteTokenAudienceName},
				Subject:   "late@example.com",
				IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
			},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		token.Header["kid"] = KeyID
		tokenString, err := token.SignedString(secret)
		require.NoError(t, err)

		_, err = ParseInviteToken(tokenString, secret)
		require.Error(t, err)
		assert.ErrorIs(t, err, jwt.ErrTokenExpired)
	})

	t.Run("fails with a tampered signature", func(t *testing.T) {
		token, _, err := GenerateInviteToken("invited@example.com", InviteRoleUser, secret)
		require.NoError(t, err)
		parts := strings.Split(token, ".")
		require.Len(t, parts, 3)
		parts[2] = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"

		_, err = ParseInviteToken(strings.Join(parts, "."), secret)
		assert.Error(t, err)
	})

	t.Run("fails with a tampered payload", func(t *testing.T) {
		token, _, err := GenerateInviteToken("invited@example.com", InviteRoleUser, secret)
		require.NoError(t, err)
		other, _, err := GenerateInviteToken("other@example.com", InviteRoleUser, secret)
		require.NoError(t, err)
		parts := strings.Split(token, ".")
		otherParts := strings.Split(other, ".")
		parts[1] = otherParts[1]

		_, err = ParseInviteToken(strings.Join(parts, "."), secret)
		assert.Error(t, err)
	})

	t.Run("rejects an access token", func(t *testing.T) {
		accessToken, _, err := GenerateAccessTokenV2(1, "testuser", "USER", "ACTIVE", secret)
		require.NoError(t, err)

		_, err = ParseInviteToken(accessToken, secret)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid audience")
	})

	t.Run("rejects a refresh token", func(t *testing.T) {
		refreshToken, _, err := GenerateRefreshToken(1, "token-id", secret)
		require.NoError(t, err)

		_, err = ParseInviteToken(refreshToken, secret)
		assert.Error(t, err)
	})

	t.Run("fails with wrong secret", func(t *testing.T) {
		token, _, err := GenerateInviteToken("invited@example.com", InviteRoleUser, secret)
		require.NoError(t, err)

		_, err = ParseInviteToken(token, []byte("wrong-secret"))
		assert.Error(t, err)
	})

	t.Run("fails with garbage", func(t *testing.T) {
		_, err := ParseInviteToken("not-a-token", secret)
		assert.Error(t, err)
	})
}
