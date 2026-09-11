package repository

import (
	"context"
	"database/sql"
	"fmt"
	"sort"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/account"
	"github.com/Wei-Shaw/sub2api/ent/proxyipgroup"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type proxyIPGroupRepository struct {
	client *dbent.Client
	sql    sqlExecutor
}

func NewProxyIPGroupRepository(client *dbent.Client, sqlDB *sql.DB) service.ProxyIPGroupRepository {
	return &proxyIPGroupRepository{client: client, sql: sqlDB}
}

func (r *proxyIPGroupRepository) Create(ctx context.Context, group *service.ProxyIPGroup) error {
	created, err := r.client.ProxyIPGroup.Create().
		SetName(group.Name).
		SetPerIPConcurrency(group.PerIPConcurrency).
		Save(ctx)
	if err != nil {
		return err
	}
	group.ID = created.ID
	group.CreatedAt = created.CreatedAt
	group.UpdatedAt = created.UpdatedAt
	return nil
}

func (r *proxyIPGroupRepository) GetByID(ctx context.Context, id int64) (*service.ProxyIPGroup, error) {
	m, err := r.client.ProxyIPGroup.Get(ctx, id)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, service.ErrProxyIPGroupNotFound
		}
		return nil, err
	}
	return r.toService(ctx, m)
}

func (r *proxyIPGroupRepository) GetByName(ctx context.Context, name string) (*service.ProxyIPGroup, error) {
	m, err := r.client.ProxyIPGroup.Query().
		Where(proxyipgroup.NameEQ(name)).
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, service.ErrProxyIPGroupNotFound
		}
		return nil, err
	}
	return r.toService(ctx, m)
}

func (r *proxyIPGroupRepository) List(ctx context.Context) ([]service.ProxyIPGroup, error) {
	rows, err := r.client.ProxyIPGroup.Query().
		Order(proxyipgroup.ByID()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]service.ProxyIPGroup, 0, len(rows))
	for _, row := range rows {
		mapped, err := r.toService(ctx, row)
		if err != nil {
			return nil, err
		}
		out = append(out, *mapped)
	}
	return out, nil
}

func (r *proxyIPGroupRepository) Update(ctx context.Context, group *service.ProxyIPGroup) error {
	updated, err := r.client.ProxyIPGroup.UpdateOneID(group.ID).
		SetName(group.Name).
		SetPerIPConcurrency(group.PerIPConcurrency).
		Save(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return service.ErrProxyIPGroupNotFound
		}
		return err
	}
	group.UpdatedAt = updated.UpdatedAt
	return nil
}

func (r *proxyIPGroupRepository) Delete(ctx context.Context, id int64) error {
	err := r.client.ProxyIPGroup.DeleteOneID(id).Exec(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return service.ErrProxyIPGroupNotFound
		}
		return err
	}
	return nil
}

func (r *proxyIPGroupRepository) SetMembers(ctx context.Context, groupID int64, proxyIDs []int64) error {
	if _, err := r.sql.ExecContext(ctx, `DELETE FROM proxy_ip_group_members WHERE group_id = $1`, groupID); err != nil {
		return err
	}
	for _, proxyID := range proxyIDs {
		if _, err := r.sql.ExecContext(ctx, `
			INSERT INTO proxy_ip_group_members (group_id, proxy_id)
			VALUES ($1, $2)
			ON CONFLICT (group_id, proxy_id) DO NOTHING
		`, groupID, proxyID); err != nil {
			return err
		}
	}
	return nil
}

func (r *proxyIPGroupRepository) RemoveMembersByProxyID(ctx context.Context, proxyID int64) error {
	_, err := r.sql.ExecContext(ctx, `DELETE FROM proxy_ip_group_members WHERE proxy_id = $1`, proxyID)
	return err
}

func (r *proxyIPGroupRepository) CountAccountsByGroupID(ctx context.Context, groupID int64) (int64, error) {
	count, err := r.client.Account.Query().
		Where(account.ProxyIPGroupIDEQ(groupID)).
		Count(ctx)
	if err != nil {
		return 0, err
	}
	return int64(count), nil
}

func (r *proxyIPGroupRepository) ListMemberProxyIDs(ctx context.Context, groupID int64) ([]int64, error) {
	rows, err := r.sql.QueryContext(ctx, `
		SELECT proxy_id FROM proxy_ip_group_members
		WHERE group_id = $1
		ORDER BY proxy_id ASC
	`, groupID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}

func (r *proxyIPGroupRepository) toService(ctx context.Context, m *dbent.ProxyIPGroup) (*service.ProxyIPGroup, error) {
	if m == nil {
		return nil, fmt.Errorf("nil proxy ip group")
	}
	ids, err := r.ListMemberProxyIDs(ctx, m.ID)
	if err != nil {
		return nil, err
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return &service.ProxyIPGroup{
		ID:               m.ID,
		Name:             m.Name,
		PerIPConcurrency: m.PerIPConcurrency,
		ProxyIDs:         ids,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
	}, nil
}
