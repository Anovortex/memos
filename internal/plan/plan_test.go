package plan

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"

	storepb "github.com/usememos/memos/proto/gen/store"
)

func TestEffective(t *testing.T) {
	now := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	teams := func(expire *timestamppb.Timestamp) *storepb.PackageUserSetting {
		return &storepb.PackageUserSetting{Plan: storepb.PackageUserSetting_TEAMS, ExpireTime: expire}
	}

	tests := []struct {
		name string
		pkg  *storepb.PackageUserSetting
		want Tier
	}{
		{"no package", nil, TierFree},
		{"free package", &storepb.PackageUserSetting{Plan: storepb.PackageUserSetting_FREE}, TierFree},
		{"unspecified plan", &storepb.PackageUserSetting{}, TierFree},
		{"teams without expiry", teams(nil), TierTeams},
		{"teams before expiry", teams(timestamppb.New(now.Add(time.Hour))), TierTeams},
		{"teams at expiry", teams(timestamppb.New(now)), TierFree},
		{"teams after expiry", teams(timestamppb.New(now.Add(-time.Hour))), TierFree},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Effective(tt.pkg, now))
		})
	}
}
