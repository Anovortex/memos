package v1

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1pb "github.com/usememos/memos/proto/gen/api/v1"
	"github.com/usememos/memos/store"
)

const instanceStatsCacheTTL = 60 * time.Second

// instanceStatsCache is a single-value, mutex-guarded cache for InstanceStats.
type instanceStatsCache struct {
	mu     sync.Mutex
	value  *v1pb.InstanceStats
	expiry time.Time
}

func (c *instanceStatsCache) get() (*v1pb.InstanceStats, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.value == nil || time.Now().After(c.expiry) {
		return nil, false
	}
	return c.value, true
}

func (c *instanceStatsCache) set(v *v1pb.InstanceStats, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value = v
	c.expiry = time.Now().Add(ttl)
}

// GetInstanceStats returns resource usage statistics. Admin only.
func (s *APIV1Service) GetInstanceStats(ctx context.Context, _ *v1pb.GetInstanceStatsRequest) (*v1pb.InstanceStats, error) {
	user, err := s.fetchCurrentUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get current user: %v", err)
	}
	if user == nil {
		return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
	}
	if user.Role != store.RoleAdmin {
		return nil, status.Errorf(codes.PermissionDenied, "permission denied")
	}

	if cached, ok := s.instanceStatsCache.get(); ok {
		return cached, nil
	}

	stats, err := s.computeInstanceStats(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to compute instance stats: %v", err)
	}
	s.instanceStatsCache.set(stats, instanceStatsCacheTTL)
	return stats, nil
}

// computeInstanceStats runs all stat subqueries in parallel and assembles the result.
// Per-subtask failures degrade to -1 sentinel values; only a total failure (every
// subtask errored) is propagated as an error.
func (s *APIV1Service) computeInstanceStats(ctx context.Context) (*v1pb.InstanceStats, error) {
	stats := &v1pb.InstanceStats{
		Database: &v1pb.InstanceStats_DatabaseStats{
			Driver:    s.Profile.Driver,
			SizeBytes: -1,
		},
		LocalStorageBytes: -1,
		GeneratedTime:     timestamppb.Now(),
	}

	type result struct {
		name string
		err  error
	}
	var (
		mu      sync.Mutex
		results []result
		record  = func(name string, err error) {
			mu.Lock()
			results = append(results, result{name, err})
			mu.Unlock()
		}
	)

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		size, err := s.Store.GetDriver().GetDatabaseSize(gctx)
		if err != nil {
			record("database_size", err)
			return nil
		}
		stats.Database.SizeBytes = size
		return nil
	})

	g.Go(func() error {
		size, err := walkLocalStorage(s.Profile.Data)
		if err != nil {
			record("local_storage", err)
			return nil
		}
		stats.LocalStorageBytes = size
		return nil
	})

	g.Go(func() error {
		usage, err := s.computeUserUsage(gctx)
		if err != nil {
			record("user_usage", err)
			return nil
		}
		stats.UserUsage = usage
		return nil
	})

	_ = g.Wait()

	for _, r := range results {
		slog.Warn("instance stats subtask failed", slog.String("subtask", r.name), slog.String("err", r.err.Error()))
	}

	const totalSubtasks = 3
	if len(results) == totalSubtasks {
		return nil, errors.New("all instance stats subtasks failed")
	}
	return stats, nil
}

func (s *APIV1Service) computeUserUsage(ctx context.Context) ([]*v1pb.InstanceStats_UserUsage, error) {
	users, err := s.Store.ListUsers(ctx, &store.FindUser{})
	if err != nil {
		return nil, errors.Wrap(err, "list users")
	}

	usageByUserID := make(map[int32]*v1pb.InstanceStats_UserUsage, len(users))
	usage := make([]*v1pb.InstanceStats_UserUsage, 0, len(users))
	for _, user := range users {
		item := &v1pb.InstanceStats_UserUsage{Name: BuildUserName(user.Username)}
		usageByUserID[user.ID] = item
		usage = append(usage, item)
	}

	limit := 1000
	for offset := 0; ; offset += limit {
		memos, err := s.Store.ListMemos(ctx, &store.FindMemo{ExcludeComments: true, ExcludeContent: true, Limit: &limit, Offset: &offset})
		if err != nil {
			return nil, errors.Wrap(err, "list memos")
		}
		for _, memo := range memos {
			item := usageByUserID[memo.CreatorID]
			if item == nil {
				continue
			}
			item.MemoCount++
			if item.LastActivityTime == nil || memo.UpdatedTs > item.LastActivityTime.Seconds {
				item.LastActivityTime = timestamppb.New(time.Unix(memo.UpdatedTs, 0))
			}
		}
		if len(memos) < limit {
			break
		}
	}

	for offset := 0; ; offset += limit {
		attachments, err := s.Store.ListAttachments(ctx, &store.FindAttachment{Limit: &limit, Offset: &offset})
		if err != nil {
			return nil, errors.Wrap(err, "list attachments")
		}
		for _, attachment := range attachments {
			item := usageByUserID[attachment.CreatorID]
			if item == nil {
				continue
			}
			item.AttachmentCount++
			item.AttachmentBytes += attachment.Size
		}
		if len(attachments) < limit {
			break
		}
	}

	return usage, nil
}

// walkLocalStorage returns the recursive size of dir in bytes.
// Symlinks are not followed; per-entry errors below the root are ignored
// (the walk continues). An error accessing the root itself is returned.
func walkLocalStorage(dir string) (int64, error) {
	if dir == "" {
		return -1, errors.New("empty data directory")
	}
	var total int64
	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			if path == dir {
				// Root itself is inaccessible — abort the walk.
				return walkErr
			}
			// Ignore per-entry errors (e.g. permission denied on a single file).
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			// Ignore stat errors on individual entries; continue the walk.
			return nil //nolint:nilerr
		}
		total += info.Size()
		return nil
	})
	if err != nil {
		return -1, errors.Wrap(err, "walk failed")
	}
	return total, nil
}
