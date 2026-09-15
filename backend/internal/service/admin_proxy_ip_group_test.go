//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type proxyIPGroupRepoStub struct {
	groups       map[int64]*ProxyIPGroup
	byName       map[string]int64
	nextID       int64
	accountCount map[int64]int64
	removedProxy []int64
}

func newProxyIPGroupRepoStub() *proxyIPGroupRepoStub {
	return &proxyIPGroupRepoStub{
		groups:       map[int64]*ProxyIPGroup{},
		byName:       map[string]int64{},
		nextID:       1,
		accountCount: map[int64]int64{},
	}
}

func (s *proxyIPGroupRepoStub) clone(g *ProxyIPGroup) *ProxyIPGroup {
	if g == nil {
		return nil
	}
	cp := *g
	if g.ProxyIDs != nil {
		cp.ProxyIDs = append([]int64(nil), g.ProxyIDs...)
	}
	return &cp
}

func (s *proxyIPGroupRepoStub) Create(ctx context.Context, group *ProxyIPGroup) error {
	if _, ok := s.byName[group.Name]; ok {
		return ErrProxyIPGroupNameTaken
	}
	group.ID = s.nextID
	s.nextID++
	group.CreatedAt = time.Unix(1, 0).UTC()
	group.UpdatedAt = group.CreatedAt
	s.groups[group.ID] = s.clone(group)
	s.byName[group.Name] = group.ID
	return nil
}

func (s *proxyIPGroupRepoStub) GetByID(ctx context.Context, id int64) (*ProxyIPGroup, error) {
	g, ok := s.groups[id]
	if !ok {
		return nil, ErrProxyIPGroupNotFound
	}
	return s.clone(g), nil
}

func (s *proxyIPGroupRepoStub) GetByName(ctx context.Context, name string) (*ProxyIPGroup, error) {
	id, ok := s.byName[name]
	if !ok {
		return nil, ErrProxyIPGroupNotFound
	}
	return s.GetByID(ctx, id)
}

func (s *proxyIPGroupRepoStub) List(ctx context.Context) ([]ProxyIPGroup, error) {
	out := make([]ProxyIPGroup, 0, len(s.groups))
	for _, g := range s.groups {
		out = append(out, *s.clone(g))
	}
	return out, nil
}

func (s *proxyIPGroupRepoStub) Update(ctx context.Context, group *ProxyIPGroup) error {
	existing, ok := s.groups[group.ID]
	if !ok {
		return ErrProxyIPGroupNotFound
	}
	if existing.Name != group.Name {
		delete(s.byName, existing.Name)
		s.byName[group.Name] = group.ID
	}
	group.UpdatedAt = time.Unix(2, 0).UTC()
	s.groups[group.ID] = s.clone(group)
	return nil
}

func (s *proxyIPGroupRepoStub) Delete(ctx context.Context, id int64) error {
	g, ok := s.groups[id]
	if !ok {
		return ErrProxyIPGroupNotFound
	}
	delete(s.byName, g.Name)
	delete(s.groups, id)
	return nil
}

func (s *proxyIPGroupRepoStub) SetMembers(ctx context.Context, groupID int64, proxyIDs []int64) error {
	g, ok := s.groups[groupID]
	if !ok {
		return ErrProxyIPGroupNotFound
	}
	g.ProxyIDs = append([]int64(nil), proxyIDs...)
	return nil
}

func (s *proxyIPGroupRepoStub) RemoveMembersByProxyID(ctx context.Context, proxyID int64) error {
	s.removedProxy = append(s.removedProxy, proxyID)
	for _, g := range s.groups {
		kept := g.ProxyIDs[:0]
		for _, id := range g.ProxyIDs {
			if id != proxyID {
				kept = append(kept, id)
			}
		}
		g.ProxyIDs = kept
	}
	return nil
}

func (s *proxyIPGroupRepoStub) CountAccountsByGroupID(ctx context.Context, groupID int64) (int64, error) {
	return s.accountCount[groupID], nil
}

func (s *proxyIPGroupRepoStub) ListMemberProxyIDs(ctx context.Context, groupID int64) ([]int64, error) {
	g, err := s.GetByID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	return append([]int64(nil), g.ProxyIDs...), nil
}

type proxyListByIDsStub struct {
	proxyRepoStub
	proxies map[int64]Proxy
}

func (s *proxyListByIDsStub) ListByIDs(ctx context.Context, ids []int64) ([]Proxy, error) {
	out := make([]Proxy, 0, len(ids))
	for _, id := range ids {
		if p, ok := s.proxies[id]; ok {
			out = append(out, p)
		}
	}
	return out, nil
}

func TestAdminService_CreateProxyIPGroup_DefaultConcurrency(t *testing.T) {
	groups := newProxyIPGroupRepoStub()
	proxies := &proxyListByIDsStub{proxies: map[int64]Proxy{4: {ID: 4, Name: "fr"}}}
	svc := &adminServiceImpl{proxyIPGroupRepo: groups, proxyRepo: proxies}

	created, err := svc.CreateProxyIPGroup(context.Background(), &CreateProxyIPGroupInput{
		Name:     "  france  ",
		ProxyIDs: []int64{4, 4},
	})
	require.NoError(t, err)
	require.Equal(t, "france", created.Name)
	require.Equal(t, 10, created.PerIPConcurrency)
	require.Equal(t, []int64{4}, created.ProxyIDs)
}

func TestAdminService_CreateProxyIPGroup_RejectsUnknownProxy(t *testing.T) {
	svc := &adminServiceImpl{
		proxyIPGroupRepo: newProxyIPGroupRepoStub(),
		proxyRepo:        &proxyListByIDsStub{proxies: map[int64]Proxy{}},
	}
	_, err := svc.CreateProxyIPGroup(context.Background(), &CreateProxyIPGroupInput{
		Name:     "empty",
		ProxyIDs: []int64{99},
	})
	require.ErrorIs(t, err, ErrProxyNotFound)
}

func TestAdminService_CreateProxyIPGroup_RejectsDuplicateName(t *testing.T) {
	groups := newProxyIPGroupRepoStub()
	svc := &adminServiceImpl{proxyIPGroupRepo: groups, proxyRepo: &proxyListByIDsStub{}}
	_, err := svc.CreateProxyIPGroup(context.Background(), &CreateProxyIPGroupInput{Name: "france"})
	require.NoError(t, err)
	_, err = svc.CreateProxyIPGroup(context.Background(), &CreateProxyIPGroupInput{Name: "france"})
	require.ErrorIs(t, err, ErrProxyIPGroupNameTaken)
}

func TestAdminService_DeleteProxyIPGroup_InUse(t *testing.T) {
	groups := newProxyIPGroupRepoStub()
	svc := &adminServiceImpl{proxyIPGroupRepo: groups, proxyRepo: &proxyListByIDsStub{}}
	created, err := svc.CreateProxyIPGroup(context.Background(), &CreateProxyIPGroupInput{Name: "france"})
	require.NoError(t, err)
	groups.accountCount[created.ID] = 1

	err = svc.DeleteProxyIPGroup(context.Background(), created.ID)
	require.ErrorIs(t, err, ErrProxyIPGroupInUse)
}

func TestAdminService_DeleteProxyIPGroup_Success(t *testing.T) {
	groups := newProxyIPGroupRepoStub()
	svc := &adminServiceImpl{proxyIPGroupRepo: groups, proxyRepo: &proxyListByIDsStub{}}
	created, err := svc.CreateProxyIPGroup(context.Background(), &CreateProxyIPGroupInput{Name: "france"})
	require.NoError(t, err)

	require.NoError(t, svc.DeleteProxyIPGroup(context.Background(), created.ID))
	_, err = svc.GetProxyIPGroup(context.Background(), created.ID)
	require.ErrorIs(t, err, ErrProxyIPGroupNotFound)
}

func TestAdminService_DeleteProxy_RemovesIPGroupMembers(t *testing.T) {
	groups := newProxyIPGroupRepoStub()
	require.NoError(t, groups.Create(context.Background(), &ProxyIPGroup{Name: "g", ProxyIDs: []int64{7}}))
	svc := &adminServiceImpl{
		proxyRepo:        &proxyRepoStub{},
		proxyIPGroupRepo: groups,
	}
	require.NoError(t, svc.DeleteProxy(context.Background(), 7))
	require.Equal(t, []int64{7}, groups.removedProxy)
}

func TestAdminService_UpdateProxyIPGroup_ConcurrencyBounds(t *testing.T) {
	groups := newProxyIPGroupRepoStub()
	svc := &adminServiceImpl{proxyIPGroupRepo: groups, proxyRepo: &proxyListByIDsStub{}}
	created, err := svc.CreateProxyIPGroup(context.Background(), &CreateProxyIPGroupInput{Name: "france"})
	require.NoError(t, err)

	tooHigh := 1001
	_, err = svc.UpdateProxyIPGroup(context.Background(), created.ID, &UpdateProxyIPGroupInput{PerIPConcurrency: &tooHigh})
	require.Error(t, err)

	ok := 20
	updated, err := svc.UpdateProxyIPGroup(context.Background(), created.ID, &UpdateProxyIPGroupInput{PerIPConcurrency: &ok})
	require.NoError(t, err)
	require.Equal(t, 20, updated.PerIPConcurrency)
}
