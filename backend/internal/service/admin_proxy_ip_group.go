package service

import (
	"context"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

func (s *adminServiceImpl) ListProxyIPGroups(ctx context.Context) ([]ProxyIPGroup, error) {
	if s.proxyIPGroupRepo == nil {
		return []ProxyIPGroup{}, nil
	}
	return s.proxyIPGroupRepo.List(ctx)
}

func (s *adminServiceImpl) GetProxyIPGroup(ctx context.Context, id int64) (*ProxyIPGroup, error) {
	if s.proxyIPGroupRepo == nil {
		return nil, ErrProxyIPGroupNotFound
	}
	return s.proxyIPGroupRepo.GetByID(ctx, id)
}

func (s *adminServiceImpl) CreateProxyIPGroup(ctx context.Context, input *CreateProxyIPGroupInput) (*ProxyIPGroup, error) {
	if s.proxyIPGroupRepo == nil {
		return nil, ErrProxyIPGroupNotFound
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, infraerrors.BadRequest("PROXY_IP_GROUP_NAME_REQUIRED", "IP group name is required")
	}
	concurrency, err := normalizeProxyIPGroupConcurrency(input.PerIPConcurrency)
	if err != nil {
		return nil, err
	}
	if existing, err := s.proxyIPGroupRepo.GetByName(ctx, name); err != nil && err != ErrProxyIPGroupNotFound {
		return nil, err
	} else if existing != nil {
		return nil, ErrProxyIPGroupNameTaken
	}
	if err := s.validateProxyIPGroupMembers(ctx, input.ProxyIDs); err != nil {
		return nil, err
	}
	group := &ProxyIPGroup{
		Name:             name,
		PerIPConcurrency: concurrency,
		ProxyIDs:         uniqueInt64s(input.ProxyIDs),
	}
	if err := s.proxyIPGroupRepo.Create(ctx, group); err != nil {
		return nil, err
	}
	if len(group.ProxyIDs) > 0 {
		if err := s.proxyIPGroupRepo.SetMembers(ctx, group.ID, group.ProxyIDs); err != nil {
			return nil, err
		}
	}
	return s.proxyIPGroupRepo.GetByID(ctx, group.ID)
}

func (s *adminServiceImpl) UpdateProxyIPGroup(ctx context.Context, id int64, input *UpdateProxyIPGroupInput) (*ProxyIPGroup, error) {
	if s.proxyIPGroupRepo == nil {
		return nil, ErrProxyIPGroupNotFound
	}
	group, err := s.proxyIPGroupRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if name := strings.TrimSpace(input.Name); name != "" && name != group.Name {
		if existing, err := s.proxyIPGroupRepo.GetByName(ctx, name); err != nil && err != ErrProxyIPGroupNotFound {
			return nil, err
		} else if existing != nil && existing.ID != id {
			return nil, ErrProxyIPGroupNameTaken
		}
		group.Name = name
	}
	if input.PerIPConcurrency != nil {
		concurrency, err := normalizeProxyIPGroupConcurrency(*input.PerIPConcurrency)
		if err != nil {
			return nil, err
		}
		group.PerIPConcurrency = concurrency
	}
	if err := s.proxyIPGroupRepo.Update(ctx, group); err != nil {
		return nil, err
	}
	return s.proxyIPGroupRepo.GetByID(ctx, id)
}

func (s *adminServiceImpl) DeleteProxyIPGroup(ctx context.Context, id int64) error {
	if s.proxyIPGroupRepo == nil {
		return ErrProxyIPGroupNotFound
	}
	if _, err := s.proxyIPGroupRepo.GetByID(ctx, id); err != nil {
		return err
	}
	count, err := s.proxyIPGroupRepo.CountAccountsByGroupID(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrProxyIPGroupInUse
	}
	return s.proxyIPGroupRepo.Delete(ctx, id)
}

func (s *adminServiceImpl) SetProxyIPGroupMembers(ctx context.Context, id int64, proxyIDs []int64) (*ProxyIPGroup, error) {
	if s.proxyIPGroupRepo == nil {
		return nil, ErrProxyIPGroupNotFound
	}
	if _, err := s.proxyIPGroupRepo.GetByID(ctx, id); err != nil {
		return nil, err
	}
	ids := uniqueInt64s(proxyIDs)
	if err := s.validateProxyIPGroupMembers(ctx, ids); err != nil {
		return nil, err
	}
	if err := s.proxyIPGroupRepo.SetMembers(ctx, id, ids); err != nil {
		return nil, err
	}
	return s.proxyIPGroupRepo.GetByID(ctx, id)
}

func (s *adminServiceImpl) validateProxyIPGroupMembers(ctx context.Context, proxyIDs []int64) error {
	ids := uniqueInt64s(proxyIDs)
	if len(ids) == 0 {
		return nil
	}
	proxies, err := s.proxyRepo.ListByIDs(ctx, ids)
	if err != nil {
		return err
	}
	found := make(map[int64]struct{}, len(proxies))
	for i := range proxies {
		found[proxies[i].ID] = struct{}{}
	}
	for _, id := range ids {
		if _, ok := found[id]; !ok {
			return ErrProxyNotFound
		}
	}
	return nil
}

func uniqueInt64s(ids []int64) []int64 {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[int64]struct{}, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
