package service

import (
	"context"
	"errors"
)

// InvalidateAuthCacheByKey 清除指定 API Key 的认证缓存
func (s *APIKeyService) InvalidateAuthCacheByKey(ctx context.Context, key string) {
	if key == "" {
		return
	}
	cacheKey := s.authCacheKey(key)
	s.deleteAuthCache(ctx, cacheKey)
}

// InvalidateAuthCacheByUserID 清除用户相关的 API Key 认证缓存
func (s *APIKeyService) InvalidateAuthCacheByUserID(ctx context.Context, userID int64) {
	if userID <= 0 {
		return
	}
	keys, err := s.apiKeyRepo.ListKeysByUserID(ctx, userID)
	if err != nil {
		return
	}
	s.deleteAuthCacheByKeys(ctx, keys)
}

// InvalidateAuthCacheByUserIDStrict is the durable-wallet variant: unlike the
// legacy best-effort method it returns repository/Redis errors so an outbox
// worker can retry without losing a committed balance change notification.
func (s *APIKeyService) InvalidateAuthCacheByUserIDStrict(ctx context.Context, userID int64) error {
	if s == nil || userID <= 0 {
		return nil
	}
	keys, err := s.apiKeyRepo.ListKeysByUserID(ctx, userID)
	if err != nil {
		return err
	}
	var result error
	for _, key := range keys {
		if key == "" {
			continue
		}
		cacheKey := s.authCacheKey(key)
		if s.authCacheL1 != nil {
			s.authCacheL1.Del(cacheKey)
		}
		if s.authNegativeCacheL1 != nil {
			s.authNegativeCacheL1.Del(cacheKey)
		}
		if s.cache == nil {
			continue
		}
		if err := s.cache.DeleteAuthCache(ctx, cacheKey); err != nil {
			result = errors.Join(result, err)
			continue
		}
		if err := s.cache.PublishAuthCacheInvalidation(ctx, cacheKey); err != nil {
			result = errors.Join(result, err)
		}
	}
	return result
}

// InvalidateAuthCacheByGroupID 清除分组相关的 API Key 认证缓存
func (s *APIKeyService) InvalidateAuthCacheByGroupID(ctx context.Context, groupID int64) {
	if groupID <= 0 {
		return
	}
	keys, err := s.apiKeyRepo.ListKeysByGroupID(ctx, groupID)
	if err != nil {
		return
	}
	s.deleteAuthCacheByKeys(ctx, keys)
}

func (s *APIKeyService) deleteAuthCacheByKeys(ctx context.Context, keys []string) {
	if len(keys) == 0 {
		return
	}
	for _, key := range keys {
		if key == "" {
			continue
		}
		s.deleteAuthCache(ctx, s.authCacheKey(key))
	}
}
