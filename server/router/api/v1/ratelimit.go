package v1

import (
	"strconv"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/usememos/memos/internal/ratelimit"
)

// RateLimitFlow names one rate-limited flow.
type RateLimitFlow string

// Rate-limited flows. The key each one is bucketed by is in the comment.
const (
	FlowSignInIP   RateLimitFlow = "signin-ip"   // client address
	FlowSignInUser RateLimitFlow = "signin-user" // lowercased username
	FlowSignUpIP   RateLimitFlow = "signup-ip"   // client address
	FlowResetIP    RateLimitFlow = "reset-ip"    // client address
	FlowResetEmail RateLimitFlow = "reset-email" // normalized email
	FlowVerifyUser RateLimitFlow = "verify-user" // user ID
	FlowSearchUser RateLimitFlow = "search-user" // user ID
	FlowTokenIP    RateLimitFlow = "token-ip"    // client address; VerifyEmail and ResetPassword
	FlowInviteUser RateLimitFlow = "invite-user" // user ID
)

// RateLimits maps each flow to its limiter. A nil map, or a missing flow,
// allows everything; the test helper relies on that to opt out.
type RateLimits map[RateLimitFlow]*ratelimit.Limiter

// DefaultRateLimits returns the production budgets.
func DefaultRateLimits() RateLimits {
	return RateLimits{
		FlowSignInIP:   ratelimit.New(ratelimit.PerMinute(10), 20),
		FlowSignInUser: ratelimit.New(ratelimit.PerMinute(5), 10),
		FlowSignUpIP:   ratelimit.New(ratelimit.PerHour(3), 3),
		FlowResetIP:    ratelimit.New(ratelimit.PerHour(5), 5),
		FlowResetEmail: ratelimit.New(ratelimit.PerHour(3), 3),
		FlowVerifyUser: ratelimit.New(ratelimit.PerHour(3), 3),
		FlowSearchUser: ratelimit.New(ratelimit.PerMinute(30), 30),
		FlowInviteUser: ratelimit.New(ratelimit.PerHour(20), 20),
		// Tokens are ~190 bits, so this is a backstop, not the defense.
		FlowTokenIP: ratelimit.New(ratelimit.PerHour(30), 30),
	}
}

// Allow reports whether one more event for key fits the flow's budget.
func (r RateLimits) Allow(flow RateLimitFlow, key string) bool {
	return r[flow].Allow(key)
}

// check returns ResourceExhausted when the flow's budget for key is spent.
func (r RateLimits) check(flow RateLimitFlow, key string) error {
	if r.Allow(flow, key) {
		return nil
	}
	return status.Errorf(codes.ResourceExhausted, "too many attempts, please try again later")
}

// userRateKey buckets a flow by user ID.
func userRateKey(userID int32) string {
	return strconv.FormatInt(int64(userID), 10)
}
