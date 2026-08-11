//go:build unit

// 密钥级兜底（fallback_key_id）校验逻辑的单元测试。
// 覆盖 validateFallbackKey 在各场景下的行为：自引用、跨用户、不存在、合法。

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

// fallbackRepoStub 仅实现 validateFallbackKey 所需的 GetByID，
// 通过 keys map 支持按 ID 返回不同的密钥。
type fallbackRepoStub struct {
	keys map[int64]*APIKey
}

func (s *fallbackRepoStub) GetByID(ctx context.Context, id int64) (*APIKey, error) {
	if k, ok := s.keys[id]; ok {
		clone := *k
		return &clone, nil
	}
	return nil, ErrAPIKeyNotFound
}

func (s *fallbackRepoStub) GetByUserIDAndName(ctx context.Context, userID int64, name string) (*APIKey, error) {
	panic("unexpected")
}

// 其余接口方法在本测试中不应被调用。
func (s *fallbackRepoStub) Create(ctx context.Context, key *APIKey) error { panic("unexpected") }
func (s *fallbackRepoStub) GetKeyAndOwnerID(ctx context.Context, id int64) (string, int64, error) {
	panic("unexpected")
}
func (s *fallbackRepoStub) GetByKey(ctx context.Context, key string) (*APIKey, error) {
	panic("unexpected")
}
func (s *fallbackRepoStub) GetByKeyForAuth(ctx context.Context, key string) (*APIKey, error) {
	panic("unexpected")
}
func (s *fallbackRepoStub) Update(ctx context.Context, key *APIKey) error { panic("unexpected") }
func (s *fallbackRepoStub) Delete(ctx context.Context, id int64) error    { panic("unexpected") }
func (s *fallbackRepoStub) DeleteWithAudit(context.Context, int64) error  { panic("unexpected") }
func (s *fallbackRepoStub) ListByUserID(ctx context.Context, userID int64, params pagination.PaginationParams, filters APIKeyListFilters) ([]APIKey, *pagination.PaginationResult, error) {
	panic("unexpected")
}
func (s *fallbackRepoStub) VerifyOwnership(ctx context.Context, userID int64, apiKeyIDs []int64) ([]int64, error) {
	panic("unexpected")
}
func (s *fallbackRepoStub) CountByUserID(ctx context.Context, userID int64) (int64, error) {
	panic("unexpected")
}
func (s *fallbackRepoStub) ExistsByKey(ctx context.Context, key string) (bool, error) {
	panic("unexpected")
}
func (s *fallbackRepoStub) ListByGroupID(ctx context.Context, groupID int64, params pagination.PaginationParams) ([]APIKey, *pagination.PaginationResult, error) {
	panic("unexpected")
}
func (s *fallbackRepoStub) SearchAPIKeys(ctx context.Context, userID int64, keyword string, limit int) ([]APIKey, error) {
	panic("unexpected")
}
func (s *fallbackRepoStub) ClearGroupIDByGroupID(ctx context.Context, groupID int64) (int64, error) {
	panic("unexpected")
}
func (s *fallbackRepoStub) UpdateGroupIDByUserAndGroup(ctx context.Context, userID, oldGroupID, newGroupID int64) (int64, error) {
	panic("unexpected")
}
func (s *fallbackRepoStub) CountByGroupID(ctx context.Context, groupID int64) (int64, error) {
	panic("unexpected")
}
func (s *fallbackRepoStub) ListKeysByUserID(ctx context.Context, userID int64) ([]string, error) {
	panic("unexpected")
}
func (s *fallbackRepoStub) ListKeysByGroupID(ctx context.Context, groupID int64) ([]string, error) {
	panic("unexpected")
}
func (s *fallbackRepoStub) IncrementQuotaUsed(ctx context.Context, id int64, amount float64) (float64, error) {
	panic("unexpected")
}
func (s *fallbackRepoStub) UpdateLastUsed(ctx context.Context, id int64, usedAt time.Time) error {
	panic("unexpected")
}
func (s *fallbackRepoStub) IncrementRateLimitUsage(ctx context.Context, id int64, cost float64) error {
	panic("unexpected")
}
func (s *fallbackRepoStub) ResetRateLimitWindows(ctx context.Context, id int64) error {
	panic("unexpected")
}
func (s *fallbackRepoStub) GetRateLimitData(ctx context.Context, id int64) (*APIKeyRateLimitData, error) {
	panic("unexpected")
}

func TestValidateFallbackKey(t *testing.T) {
	repo := &fallbackRepoStub{keys: map[int64]*APIKey{
		1: {ID: 1, UserID: 100, Key: "self"},
		2: {ID: 2, UserID: 100, Key: "same-owner"},
		3: {ID: 3, UserID: 200, Key: "other-owner"},
	}}
	svc := &APIKeyService{apiKeyRepo: repo}
	ctx := context.Background()

	t.Run("self-reference rejected on update", func(t *testing.T) {
		err := svc.validateFallbackKey(ctx, 100, 1, 1) // selfID=1, fallback=1
		require.ErrorIs(t, err, ErrFallbackKeyInvalid)
	})

	t.Run("cross-user rejected", func(t *testing.T) {
		err := svc.validateFallbackKey(ctx, 100, 3, 1) // key 3 belongs to user 200
		require.ErrorIs(t, err, ErrFallbackKeyInvalid)
	})

	t.Run("non-existent rejected", func(t *testing.T) {
		err := svc.validateFallbackKey(ctx, 100, 999, 1)
		require.ErrorIs(t, err, ErrFallbackKeyInvalid)
	})

	t.Run("valid same-owner different key accepted", func(t *testing.T) {
		err := svc.validateFallbackKey(ctx, 100, 2, 1) // self=1, fallback=2, same user
		require.NoError(t, err)
	})

	t.Run("valid on create (no self to exclude)", func(t *testing.T) {
		err := svc.validateFallbackKey(ctx, 100, 2, 0) // selfID=0 (create)
		require.NoError(t, err)
	})
}

func TestNormalizePositiveID(t *testing.T) {
	zero := int64(0)
	neg := int64(-5)
	pos := int64(7)

	require.Nil(t, normalizePositiveID(nil))
	require.Nil(t, normalizePositiveID(&zero))
	require.Nil(t, normalizePositiveID(&neg))
	require.Equal(t, &pos, normalizePositiveID(&pos))
}
