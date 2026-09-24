package test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	v1pb "github.com/usememos/memos/proto/gen/api/v1"
)

// The billing setting (price and how to pay) is read by signed-in members and
// written only by the operator.
func TestInstanceBillingSetting(t *testing.T) {
	ctx := context.Background()
	ts := NewTestService(t)
	defer ts.Cleanup()

	admin, err := ts.CreateHostUser(ctx, "admin")
	require.NoError(t, err)
	member, err := ts.CreateRegularUser(ctx, "member")
	require.NoError(t, err)
	adminCtx := ts.CreateUserContext(ctx, admin.ID)
	memberCtx := ts.CreateUserContext(ctx, member.ID)
	name := "instance/settings/BILLING"
	update := func(setting *v1pb.InstanceSetting_BillingSetting) *v1pb.UpdateInstanceSettingRequest {
		return &v1pb.UpdateInstanceSettingRequest{
			Setting: &v1pb.InstanceSetting{Name: name, Value: &v1pb.InstanceSetting_BillingSetting_{BillingSetting: setting}},
		}
	}

	t.Run("anonymous visitors cannot read it", func(t *testing.T) {
		_, err := ts.Service.GetInstanceSetting(ctx, &v1pb.GetInstanceSettingRequest{Name: name})
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("a member reads defaults before the operator sets anything", func(t *testing.T) {
		setting, err := ts.Service.GetInstanceSetting(memberCtx, &v1pb.GetInstanceSettingRequest{Name: name})
		require.NoError(t, err)
		require.Equal(t, int32(30), setting.GetBillingSetting().PeriodDays)
		require.Empty(t, setting.GetBillingSetting().TeamsPrice)
	})

	t.Run("a member cannot change it", func(t *testing.T) {
		_, err := ts.Service.UpdateInstanceSetting(memberCtx, update(&v1pb.InstanceSetting_BillingSetting{TeamsPrice: "1"}))
		require.Equal(t, codes.PermissionDenied, status.Code(err))
	})

	t.Run("the operator sets price, instructions and period, and members see them", func(t *testing.T) {
		_, err := ts.Service.UpdateInstanceSetting(adminCtx, update(&v1pb.InstanceSetting_BillingSetting{
			TeamsPrice:          "$5 / month",
			PaymentInstructions: "PayPal pay@example.com, reference: your username",
			PeriodDays:          31,
		}))
		require.NoError(t, err)

		setting, err := ts.Service.GetInstanceSetting(memberCtx, &v1pb.GetInstanceSettingRequest{Name: name})
		require.NoError(t, err)
		require.Equal(t, "$5 / month", setting.GetBillingSetting().TeamsPrice)
		require.Contains(t, setting.GetBillingSetting().PaymentInstructions, "PayPal")
		require.Equal(t, int32(31), setting.GetBillingSetting().PeriodDays)
	})

	t.Run("out-of-range values are InvalidArgument", func(t *testing.T) {
		_, err := ts.Service.UpdateInstanceSetting(adminCtx, update(&v1pb.InstanceSetting_BillingSetting{PeriodDays: 5000}))
		require.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = ts.Service.UpdateInstanceSetting(adminCtx, update(&v1pb.InstanceSetting_BillingSetting{TeamsPrice: strings.Repeat("x", 201)}))
		require.Equal(t, codes.InvalidArgument, status.Code(err))
	})
}
