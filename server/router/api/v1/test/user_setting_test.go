package test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	colorpb "google.golang.org/genproto/googleapis/type/color"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	apiv1 "github.com/usememos/memos/proto/gen/api/v1"
	storepb "github.com/usememos/memos/proto/gen/store"
	apiv1server "github.com/usememos/memos/server/router/api/v1"
)

func TestListUserSettingsOmitsInternalStoreSettings(t *testing.T) {
	ctx := context.Background()
	ts := NewTestService(t)
	defer ts.Cleanup()

	user, err := ts.CreateRegularUser(ctx, "locale-user")
	require.NoError(t, err)

	_, err = ts.Store.UpsertUserSetting(ctx, &storepb.UserSetting{
		UserId: user.ID,
		Key:    storepb.UserSetting_REFRESH_TOKENS,
		Value: &storepb.UserSetting_RefreshTokens{
			RefreshTokens: &storepb.RefreshTokensUserSetting{},
		},
	})
	require.NoError(t, err)

	_, err = ts.Store.UpsertUserSetting(ctx, &storepb.UserSetting{
		UserId: user.ID,
		Key:    storepb.UserSetting_SHORTCUTS,
		Value: &storepb.UserSetting_Shortcuts{
			Shortcuts: &storepb.ShortcutsUserSetting{},
		},
	})
	require.NoError(t, err)

	_, err = ts.Store.UpsertUserSetting(ctx, &storepb.UserSetting{
		UserId: user.ID,
		Key:    storepb.UserSetting_GENERAL,
		Value: &storepb.UserSetting_General{
			General: &storepb.GeneralUserSetting{
				Locale: "ja",
			},
		},
	})
	require.NoError(t, err)

	resp, err := ts.Service.ListUserSettings(ts.CreateUserContext(ctx, user.ID), &apiv1.ListUserSettingsRequest{
		Parent: apiv1server.BuildUserName(user.Username),
	})
	require.NoError(t, err)
	require.Len(t, resp.Settings, 1)
	require.Equal(t, "users/locale-user/settings/GENERAL", resp.Settings[0].Name)
	require.Equal(t, "ja", resp.Settings[0].GetGeneralSetting().Locale)
}

func TestUserTagSettings(t *testing.T) {
	ctx := context.Background()
	ts := NewTestService(t)
	defer ts.Cleanup()

	user, err := ts.CreateRegularUser(ctx, "tag-user")
	require.NoError(t, err)
	userCtx := ts.CreateUserContext(ctx, user.ID)

	tagsSetting := &apiv1.UserSetting{
		Name: "users/tag-user/settings/TAGS",
		Value: &apiv1.UserSetting_TagsSetting_{
			TagsSetting: &apiv1.UserSetting_TagsSetting{
				Tags: map[string]*apiv1.UserSetting_TagMetadata{
					"bug": {
						BackgroundColor: &colorpb.Color{
							Red:   0.9,
							Green: 0.1,
							Blue:  0.1,
						},
					},
					"private/.*": {
						BlurContent: true,
					},
				},
			},
		},
	}

	updated, err := ts.Service.UpdateUserSetting(userCtx, &apiv1.UpdateUserSettingRequest{
		Setting:    tagsSetting,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"tags"}},
	})
	require.NoError(t, err)
	require.Equal(t, "users/tag-user/settings/TAGS", updated.Name)
	require.Contains(t, updated.GetTagsSetting().GetTags(), "bug")
	require.True(t, updated.GetTagsSetting().GetTags()["private/.*"].BlurContent)

	got, err := ts.Service.GetUserSetting(userCtx, &apiv1.GetUserSettingRequest{Name: "users/tag-user/settings/TAGS"})
	require.NoError(t, err)
	require.Contains(t, got.GetTagsSetting().GetTags(), "bug")

	resp, err := ts.Service.ListUserSettings(userCtx, &apiv1.ListUserSettingsRequest{
		Parent: apiv1server.BuildUserName(user.Username),
	})
	require.NoError(t, err)
	var foundTags bool
	for _, setting := range resp.Settings {
		if setting.GetTagsSetting() != nil {
			foundTags = true
			require.Equal(t, "users/tag-user/settings/TAGS", setting.Name)
		}
	}
	require.True(t, foundTags)
}

func TestUserTagSettingsRejectInvalidInput(t *testing.T) {
	ctx := context.Background()
	ts := NewTestService(t)
	defer ts.Cleanup()

	user, err := ts.CreateRegularUser(ctx, "invalid-tag-user")
	require.NoError(t, err)
	userCtx := ts.CreateUserContext(ctx, user.ID)

	_, err = ts.Service.UpdateUserSetting(userCtx, &apiv1.UpdateUserSettingRequest{
		Setting: &apiv1.UserSetting{
			Name: "users/invalid-tag-user/settings/TAGS",
			Value: &apiv1.UserSetting_TagsSetting_{
				TagsSetting: &apiv1.UserSetting_TagsSetting{
					Tags: map[string]*apiv1.UserSetting_TagMetadata{
						"(": {},
					},
				},
			},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"tags"}},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid")

	_, err = ts.Service.UpdateUserSetting(userCtx, &apiv1.UpdateUserSettingRequest{
		Setting: &apiv1.UserSetting{
			Name: "users/invalid-tag-user/settings/TAGS",
			Value: &apiv1.UserSetting_TagsSetting_{
				TagsSetting: &apiv1.UserSetting_TagsSetting{
					Tags: map[string]*apiv1.UserSetting_TagMetadata{
						"bug": {
							BackgroundColor: &colorpb.Color{Red: 1.2},
						},
					},
				},
			},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"tags"}},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid")

	_, err = ts.Service.UpdateUserSetting(userCtx, &apiv1.UpdateUserSettingRequest{
		Setting: &apiv1.UserSetting{
			Name: "users/invalid-tag-user/settings/TAGS",
			Value: &apiv1.UserSetting_TagsSetting_{
				TagsSetting: &apiv1.UserSetting_TagsSetting{
					Tags: map[string]*apiv1.UserSetting_TagMetadata{
						"bug": {},
					},
				},
			},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"locale"}},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported update mask path")
}

func TestUserTagSettingsPermissionDeniedForOtherUser(t *testing.T) {
	ctx := context.Background()
	ts := NewTestService(t)
	defer ts.Cleanup()

	_, err := ts.CreateRegularUser(ctx, "tag-owner")
	require.NoError(t, err)
	other, err := ts.CreateRegularUser(ctx, "tag-other")
	require.NoError(t, err)
	otherCtx := ts.CreateUserContext(ctx, other.ID)

	_, err = ts.Service.GetUserSetting(otherCtx, &apiv1.GetUserSettingRequest{Name: "users/tag-owner/settings/TAGS"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "permission denied")

	_, err = ts.Service.UpdateUserSetting(otherCtx, &apiv1.UpdateUserSettingRequest{
		Setting: &apiv1.UserSetting{
			Name: "users/tag-owner/settings/TAGS",
			Value: &apiv1.UserSetting_TagsSetting_{
				TagsSetting: &apiv1.UserSetting_TagsSetting{
					Tags: map[string]*apiv1.UserSetting_TagMetadata{"bug": {}},
				},
			},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"tags"}},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "permission denied")
}

func TestUserPackageSetting(t *testing.T) {
	ctx := context.Background()
	ts := NewTestService(t)
	defer ts.Cleanup()

	admin, err := ts.CreateHostUser(ctx, "admin")
	require.NoError(t, err)
	member, err := ts.CreateRegularUser(ctx, "member")
	require.NoError(t, err)
	other, err := ts.CreateRegularUser(ctx, "other")
	require.NoError(t, err)
	adminCtx := ts.CreateUserContext(ctx, admin.ID)
	memberCtx := ts.CreateUserContext(ctx, member.ID)
	otherCtx := ts.CreateUserContext(ctx, other.ID)
	name := "users/member/settings/PACKAGE"
	expire := timestamppb.New(time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC))
	packageUpdate := func(plan apiv1.UserSetting_PackageSetting_Plan, paths ...string) *apiv1.UpdateUserSettingRequest {
		return &apiv1.UpdateUserSettingRequest{
			Setting: &apiv1.UserSetting{
				Name: name,
				Value: &apiv1.UserSetting_PackageSetting_{
					PackageSetting: &apiv1.UserSetting_PackageSetting{Plan: plan, ExpireTime: expire},
				},
			},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: paths},
		}
	}

	t.Run("a member with no package reads Free", func(t *testing.T) {
		setting, err := ts.Service.GetUserSetting(memberCtx, &apiv1.GetUserSettingRequest{Name: name})
		require.NoError(t, err)
		require.Equal(t, apiv1.UserSetting_PackageSetting_FREE, setting.GetPackageSetting().Plan)
		require.Nil(t, setting.GetPackageSetting().ExpireTime)
	})

	t.Run("a member cannot set their own package", func(t *testing.T) {
		_, err := ts.Service.UpdateUserSetting(memberCtx, packageUpdate(apiv1.UserSetting_PackageSetting_TEAMS, "plan"))
		require.Equal(t, codes.PermissionDenied, status.Code(err))
		require.Contains(t, err.Error(), "operator")
	})

	t.Run("the operator sets Teams with an expiry", func(t *testing.T) {
		setting, err := ts.Service.UpdateUserSetting(adminCtx, packageUpdate(apiv1.UserSetting_PackageSetting_TEAMS, "plan", "expire_time"))
		require.NoError(t, err)
		require.Equal(t, apiv1.UserSetting_PackageSetting_TEAMS, setting.GetPackageSetting().Plan)
		require.True(t, expire.AsTime().Equal(setting.GetPackageSetting().ExpireTime.AsTime()))

		mine, err := ts.Service.GetUserSetting(memberCtx, &apiv1.GetUserSettingRequest{Name: name})
		require.NoError(t, err)
		require.Equal(t, apiv1.UserSetting_PackageSetting_TEAMS, mine.GetPackageSetting().Plan)

		listed, err := ts.Service.ListUserSettings(memberCtx, &apiv1.ListUserSettingsRequest{Parent: "users/member"})
		require.NoError(t, err)
		var found bool
		for _, s := range listed.Settings {
			if s.GetPackageSetting() != nil {
				found = true
				require.Equal(t, apiv1.UserSetting_PackageSetting_TEAMS, s.GetPackageSetting().Plan)
			}
		}
		require.True(t, found, "package listed with the member's own settings")
	})

	t.Run("the operator reads a member's package, another member cannot", func(t *testing.T) {
		setting, err := ts.Service.GetUserSetting(adminCtx, &apiv1.GetUserSettingRequest{Name: name})
		require.NoError(t, err)
		require.Equal(t, apiv1.UserSetting_PackageSetting_TEAMS, setting.GetPackageSetting().Plan)

		_, err = ts.Service.GetUserSetting(otherCtx, &apiv1.GetUserSettingRequest{Name: name})
		require.Equal(t, codes.PermissionDenied, status.Code(err))
		_, err = ts.Service.GetUserSetting(otherCtx, &apiv1.GetUserSettingRequest{Name: "users/member/settings/GENERAL"})
		require.Equal(t, codes.PermissionDenied, status.Code(err), "operator exception is for the package only")
	})

	t.Run("a mask of plan only keeps the expiry", func(t *testing.T) {
		setting, err := ts.Service.UpdateUserSetting(adminCtx, packageUpdate(apiv1.UserSetting_PackageSetting_FREE, "plan"))
		require.NoError(t, err)
		require.Equal(t, apiv1.UserSetting_PackageSetting_FREE, setting.GetPackageSetting().Plan)
		require.NotNil(t, setting.GetPackageSetting().ExpireTime)
	})

	t.Run("an unknown plan or mask path is InvalidArgument", func(t *testing.T) {
		_, err := ts.Service.UpdateUserSetting(adminCtx, packageUpdate(apiv1.UserSetting_PackageSetting_PLAN_UNSPECIFIED, "plan"))
		require.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = ts.Service.UpdateUserSetting(adminCtx, packageUpdate(apiv1.UserSetting_PackageSetting_TEAMS, "tags"))
		require.Equal(t, codes.InvalidArgument, status.Code(err))
	})
}
