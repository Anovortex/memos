// Package plan resolves per-user tiers and usage limits. Limits are still a
// single free tier read from the instance profile; the tier itself comes from
// the operator-set package and lapses on its own at the expiry, no cron.
package plan

import (
	"time"

	"github.com/usememos/memos/internal/profile"
	storepb "github.com/usememos/memos/proto/gen/store"
	"github.com/usememos/memos/store"
)

// Tier is a user's effective package.
type Tier string

const (
	TierFree  Tier = "FREE"
	TierTeams Tier = "TEAMS"
)

// Effective resolves the tier a package grants at the given time. A missing
// package, or a Teams package whose expiry has passed, is Free.
func Effective(pkg *storepb.PackageUserSetting, now time.Time) Tier {
	if pkg == nil || pkg.Plan != storepb.PackageUserSetting_TEAMS {
		return TierFree
	}
	if pkg.ExpireTime != nil && !now.Before(pkg.ExpireTime.AsTime()) {
		return TierFree
	}
	return TierTeams
}

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
