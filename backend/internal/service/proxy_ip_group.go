package service

import (
	"context"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	DefaultProxyIPGroupPerIPConcurrency = 10
	MinProxyIPGroupPerIPConcurrency     = 1
	MaxProxyIPGroupPerIPConcurrency     = 1000
)

var (
	ErrProxyIPGroupNotFound  = infraerrors.NotFound("PROXY_IP_GROUP_NOT_FOUND", "IP group not found")
	ErrProxyIPGroupInUse     = infraerrors.Conflict("PROXY_IP_GROUP_IN_USE", "IP group is still assigned to accounts")
	ErrProxyIPGroupNameTaken = infraerrors.Conflict("PROXY_IP_GROUP_NAME_TAKEN", "IP group name already exists")
	ErrProxyIPGroupNotAllowed = infraerrors.BadRequest("PROXY_IP_GROUP_NOT_ALLOWED", "IP group can only be assigned to OpenAI OAuth accounts")
)

// ProxyIPGroup is an admin-defined set of proxies with one per-IP concurrency cap.
type ProxyIPGroup struct {
	ID               int64
	Name             string
	PerIPConcurrency int
	ProxyIDs         []int64
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type CreateProxyIPGroupInput struct {
	Name             string
	PerIPConcurrency int
	ProxyIDs         []int64
}

type UpdateProxyIPGroupInput struct {
	Name             string
	PerIPConcurrency *int
}

// ProxyIPGroupRepository persists IP groups and membership.
type ProxyIPGroupRepository interface {
	Create(ctx context.Context, group *ProxyIPGroup) error
	GetByID(ctx context.Context, id int64) (*ProxyIPGroup, error)
	GetByName(ctx context.Context, name string) (*ProxyIPGroup, error)
	List(ctx context.Context) ([]ProxyIPGroup, error)
	Update(ctx context.Context, group *ProxyIPGroup) error
	Delete(ctx context.Context, id int64) error
	SetMembers(ctx context.Context, groupID int64, proxyIDs []int64) error
	RemoveMembersByProxyID(ctx context.Context, proxyID int64) error
	CountAccountsByGroupID(ctx context.Context, groupID int64) (int64, error)
	ListMemberProxyIDs(ctx context.Context, groupID int64) ([]int64, error)
}

func normalizeProxyIPGroupConcurrency(value int) (int, error) {
	if value == 0 {
		return DefaultProxyIPGroupPerIPConcurrency, nil
	}
	if value < MinProxyIPGroupPerIPConcurrency || value > MaxProxyIPGroupPerIPConcurrency {
		return 0, infraerrors.BadRequest("PROXY_IP_GROUP_CONCURRENCY_INVALID", "per_ip_concurrency must be between 1 and 1000")
	}
	return value, nil
}

func proxyIsLive(p *Proxy, now time.Time) bool {
	return p != nil && p.IsActive() && !p.IsExpired(now)
}
