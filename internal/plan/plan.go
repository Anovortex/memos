// Package plan resolves per-user usage limits. v1 has a single free tier read
// from the instance profile; a future PRO tier changes only ForUser (a plan
// lookup on the user), never the enforcement sites.
package plan

import (
	"time"

	"github.com/usememos/memos/internal/profile"
	"github.com/usememos/memos/store"
)

// Limits are the effective caps for one user. Zero values mean unlimited.
type Limits struct {
	MaxMemos       int64
	AITokensPerDay int64
}

// ForUser returns the limits that apply to the given user.
func ForUser(p *profile.Profile, _ *store.User) Limits {
	if p == nil {
		return Limits{}
	}
	return Limits{
		MaxMemos:       p.FreeTierMaxMemos,
		AITokensPerDay: p.FreeTierAITokensPerDay,
	}
}

// UsageDate formats a time as the UTC day key used by the user_ai_usage table.
func UsageDate(t time.Time) string {
	return t.UTC().Format("2006-01-02")
}
