package test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	apiv1 "github.com/usememos/memos/proto/gen/api/v1"
	storepb "github.com/usememos/memos/proto/gen/store"
	apiv1server "github.com/usememos/memos/server/api/v1"
	"github.com/usememos/memos/store"
)

func signUp(ctx context.Context, ts *TestService, username, email, password string) (*apiv1.User, error) {
	return ts.Service.CreateUser(ctx, &apiv1.CreateUserRequest{
		User: &apiv1.User{
			Username: username,
			Email:    email,
			Password: password,
		},
	})
}

func TestSignupCorrectness(t *testing.T) {
	ctx := context.Background()

	t.Run("duplicate username is AlreadyExists with a friendly message", func(t *testing.T) {
		ts := NewTestService(t)
		defer ts.Cleanup()
		_, err := ts.CreateHostUser(ctx, "admin")
		require.NoError(t, err)

		_, err = signUp(ctx, ts, "alice", "alice@example.com", "password123")
		require.NoError(t, err)

		_, err = signUp(ctx, ts, "alice", "other@example.com", "password123")
		require.Error(t, err)
		require.Equal(t, codes.AlreadyExists, status.Code(err))
		require.Contains(t, err.Error(), "already in use")
		require.NotContains(t, err.Error(), "UNIQUE constraint")
	})

	t.Run("duplicate email is AlreadyExists regardless of case and stored lowercase", func(t *testing.T) {
		ts := NewTestService(t)
		defer ts.Cleanup()
		_, err := ts.CreateHostUser(ctx, "admin")
		require.NoError(t, err)

		created, err := signUp(ctx, ts, "alice", " Alice@Example.com ", "password123")
		require.NoError(t, err)
		require.Equal(t, "alice@example.com", created.Email)

		_, err = signUp(ctx, ts, "bob", "ALICE@example.com", "password123")
		require.Error(t, err)
		require.Equal(t, codes.AlreadyExists, status.Code(err))
		require.Contains(t, err.Error(), "already in use")
	})

	t.Run("self sign-up requires a valid email", func(t *testing.T) {
		ts := NewTestService(t)
		defer ts.Cleanup()
		_, err := ts.CreateHostUser(ctx, "admin")
		require.NoError(t, err)

		_, err = signUp(ctx, ts, "noemail", "", "password123")
		require.Error(t, err)
		require.Equal(t, codes.InvalidArgument, status.Code(err))
		require.Contains(t, err.Error(), "email is required")

		_, err = signUp(ctx, ts, "bademail", "not-an-email", "password123")
		require.Error(t, err)
		require.Equal(t, codes.InvalidArgument, status.Code(err))
		require.Contains(t, err.Error(), "not a valid address")

		_, err = signUp(ctx, ts, "displayname", "Admin <admin@example.com>", "password123")
		require.Error(t, err)
		require.Equal(t, codes.InvalidArgument, status.Code(err), "display-name forms are not canonical addresses")
	})

	t.Run("closed registration does not reveal taken names", func(t *testing.T) {
		ts := NewTestService(t)
		defer ts.Cleanup()
		_, err := ts.CreateHostUser(ctx, "admin")
		require.NoError(t, err)
		_, err = signUp(ctx, ts, "alice", "alice@example.com", "password123")
		require.NoError(t, err)

		_, err = ts.Store.UpsertInstanceSetting(ctx, &storepb.InstanceSetting{
			Key: storepb.InstanceSettingKey_GENERAL,
			Value: &storepb.InstanceSetting_GeneralSetting{
				GeneralSetting: &storepb.InstanceGeneralSetting{DisallowUserRegistration: true},
			},
		})
		require.NoError(t, err)

		_, err = signUp(ctx, ts, "alice", "alice@example.com", "password123")
		require.Equal(t, codes.PermissionDenied, status.Code(err), "the gate must answer before any uniqueness lookup")
	})

	t.Run("first-run admin also requires an email", func(t *testing.T) {
		ts := NewTestService(t)
		defer ts.Cleanup()

		_, err := signUp(ctx, ts, "owner", "", "password123")
		require.Error(t, err)
		require.Equal(t, codes.InvalidArgument, status.Code(err))

		owner, err := signUp(ctx, ts, "owner", "Owner@Example.com", "password123")
		require.NoError(t, err)
		require.Equal(t, apiv1.User_ADMIN, owner.Role)
		require.Equal(t, "owner@example.com", owner.Email)
	})

	t.Run("password must be at least 8 characters", func(t *testing.T) {
		ts := NewTestService(t)
		defer ts.Cleanup()

		_, err := signUp(ctx, ts, "shorty", "shorty@example.com", "short1")
		require.Error(t, err)
		require.Equal(t, codes.InvalidArgument, status.Code(err))
		require.Contains(t, err.Error(), "at least 8")
	})

	t.Run("admin-created users may omit the email", func(t *testing.T) {
		ts := NewTestService(t)
		defer ts.Cleanup()
		admin, err := ts.CreateHostUser(ctx, "admin")
		require.NoError(t, err)
		adminCtx := ts.CreateUserContext(ctx, admin.ID)

		_, err = signUp(adminCtx, ts, "service-account", "", "password123")
		require.NoError(t, err)
	})

	t.Run("invalid username explains the rule", func(t *testing.T) {
		ts := NewTestService(t)
		defer ts.Cleanup()

		_, err := signUp(ctx, ts, "bad_name", "bad@example.com", "password123")
		require.Error(t, err)
		require.Equal(t, codes.InvalidArgument, status.Code(err))
		require.Contains(t, err.Error(), "invalid username")
	})

	t.Run("UpdateUser email is validated, normalized, and unique", func(t *testing.T) {
		ts := NewTestService(t)
		defer ts.Cleanup()
		alice, err := ts.CreateRegularUser(ctx, "alice")
		require.NoError(t, err)
		bob, err := ts.CreateRegularUser(ctx, "bob")
		require.NoError(t, err)
		bobCtx := ts.CreateUserContext(ctx, bob.ID)

		updateEmail := func(email string) (*apiv1.User, error) {
			return ts.Service.UpdateUser(bobCtx, &apiv1.UpdateUserRequest{
				User: &apiv1.User{
					Name:  apiv1server.BuildUserName(bob.Username),
					Email: email,
				},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"email"}},
			})
		}

		_, err = updateEmail(" ALICE@example.com ")
		require.Error(t, err)
		require.Equal(t, codes.AlreadyExists, status.Code(err), "alice's email is %s", alice.Email)

		_, err = updateEmail("nope")
		require.Error(t, err)
		require.Equal(t, codes.InvalidArgument, status.Code(err))

		updated, err := updateEmail(" Bob.New@Example.com ")
		require.NoError(t, err)
		require.Equal(t, "bob.new@example.com", updated.Email)

		// Re-saving the same address (different case) is not a conflict with self.
		_, err = updateEmail("BOB.NEW@example.com")
		require.NoError(t, err)

		stored, err := ts.Store.GetUser(ctx, &store.FindUser{ID: &bob.ID})
		require.NoError(t, err)
		require.Equal(t, "bob.new@example.com", stored.Email)
	})
}
