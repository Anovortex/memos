package test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/usememos/memos/internal/ratelimit"
	apiv1 "github.com/usememos/memos/proto/gen/api/v1"
	apiv1server "github.com/usememos/memos/server/router/api/v1"
)

// fromIP simulates a request whose socket peer is the given public address.
func fromIP(ctx context.Context, ip string) context.Context {
	return metadata.NewIncomingContext(ctx, metadata.Pairs("x-memos-peer", ip+":50000"))
}

func onePerHour() *ratelimit.Limiter {
	return ratelimit.New(rate.Every(time.Hour), 1)
}

func passwordSignIn(ctx context.Context, ts *TestService, username, password string) error {
	_, err := ts.Service.SignIn(apiv1server.WithHeaderCarrier(ctx), &apiv1.SignInRequest{
		Credentials: &apiv1.SignInRequest_PasswordCredentials_{
			PasswordCredentials: &apiv1.SignInRequest_PasswordCredentials{Username: username, Password: password},
		},
	})
	return err
}

func TestRateLimits(t *testing.T) {
	ctx := context.Background()

	t.Run("sign-in is limited per client address", func(t *testing.T) {
		ts := NewTestService(t)
		defer ts.Cleanup()
		createLegacyPasswordUser(ctx, t, ts, "alice", "password123")
		ts.Service.RateLimits = apiv1server.RateLimits{apiv1server.FlowSignInIP: onePerHour()}

		err := passwordSignIn(fromIP(ctx, "203.0.113.1"), ts, "alice", "wrong-password")
		require.Equal(t, codes.InvalidArgument, status.Code(err), "first attempt is judged on credentials")

		err = passwordSignIn(fromIP(ctx, "203.0.113.1"), ts, "alice", "password123")
		require.Equal(t, codes.ResourceExhausted, status.Code(err), "second attempt from the same address is throttled even with the right password")

		err = passwordSignIn(fromIP(ctx, "203.0.113.2"), ts, "alice", "password123")
		require.NoError(t, err, "another address is unaffected")
	})

	t.Run("sign-in is limited per username across addresses", func(t *testing.T) {
		ts := NewTestService(t)
		defer ts.Cleanup()
		createLegacyPasswordUser(ctx, t, ts, "bob", "password123")
		ts.Service.RateLimits = apiv1server.RateLimits{apiv1server.FlowSignInUser: onePerHour()}

		err := passwordSignIn(fromIP(ctx, "203.0.113.1"), ts, "bob", "wrong-password")
		require.Equal(t, codes.InvalidArgument, status.Code(err))

		err = passwordSignIn(fromIP(ctx, "203.0.113.2"), ts, "BOB", "password123")
		require.Equal(t, codes.ResourceExhausted, status.Code(err), "username buckets are case-insensitive")
	})

	t.Run("sign-up is limited per client address", func(t *testing.T) {
		ts := NewTestService(t)
		defer ts.Cleanup()
		_, err := ts.CreateHostUser(ctx, "admin")
		require.NoError(t, err)
		ts.Service.RateLimits = apiv1server.RateLimits{apiv1server.FlowSignUpIP: onePerHour()}

		_, err = signUp(fromIP(ctx, "203.0.113.1"), ts, "first", "first@example.com", "password123")
		require.NoError(t, err)

		_, err = signUp(fromIP(ctx, "203.0.113.1"), ts, "second", "second@example.com", "password123")
		require.Equal(t, codes.ResourceExhausted, status.Code(err))

		_, err = signUp(fromIP(ctx, "203.0.113.2"), ts, "third", "third@example.com", "password123")
		require.NoError(t, err)
	})

	t.Run("admin-created users are not counted against sign-up limits", func(t *testing.T) {
		ts := NewTestService(t)
		defer ts.Cleanup()
		admin, err := ts.CreateHostUser(ctx, "admin")
		require.NoError(t, err)
		adminCtx := ts.CreateUserContext(fromIP(ctx, "203.0.113.1"), admin.ID)
		ts.Service.RateLimits = apiv1server.RateLimits{apiv1server.FlowSignUpIP: onePerHour()}

		for _, name := range []string{"svc-a", "svc-b"} {
			_, err = signUp(adminCtx, ts, name, "", "password123")
			require.NoError(t, err, name)
		}
	})

	t.Run("semantic search is limited per user", func(t *testing.T) {
		ts := NewTestService(t)
		defer ts.Cleanup()
		user, err := ts.CreateRegularUser(ctx, "searcher")
		require.NoError(t, err)
		userCtx := ts.CreateUserContext(ctx, user.ID)
		configureSearch(ctx, t, ts, &fixedQueryEmbedder{queryVector: []float32{1, 0}, tokens: 1})
		ts.Service.RateLimits = apiv1server.RateLimits{apiv1server.FlowSearchUser: onePerHour()}

		_, err = ts.Service.SearchMemos(userCtx, &apiv1.SearchMemosRequest{Query: "anything"})
		require.NoError(t, err)

		_, err = ts.Service.SearchMemos(userCtx, &apiv1.SearchMemosRequest{Query: "anything"})
		require.Equal(t, codes.ResourceExhausted, status.Code(err))
	})

	t.Run("nil RateLimits allows everything", func(t *testing.T) {
		ts := NewTestService(t)
		defer ts.Cleanup()
		createLegacyPasswordUser(ctx, t, ts, "carol", "password123")
		require.Nil(t, ts.Service.RateLimits)

		for range 5 {
			require.NoError(t, passwordSignIn(fromIP(ctx, "203.0.113.1"), ts, "carol", "password123"))
		}
	})
}
