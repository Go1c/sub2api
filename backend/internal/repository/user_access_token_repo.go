package repository

import (
	"context"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/useraccesstoken"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type userAccessTokenRepository struct {
	client *dbent.Client
}

// NewUserAccessTokenRepository creates a UserAccessToken repository.
func NewUserAccessTokenRepository(client *dbent.Client) service.UserAccessTokenRepository {
	return &userAccessTokenRepository{client: client}
}

func (r *userAccessTokenRepository) Create(ctx context.Context, token *service.UserAccessToken) error {
	client := clientFromContext(ctx, r.client)
	created, err := client.UserAccessToken.Create().
		SetUserID(token.UserID).
		SetName(token.Name).
		SetTokenHash(token.TokenHash).
		SetTokenPrefix(token.TokenPrefix).
		SetExpiresAt(token.ExpiresAt).
		Save(ctx)
	if err != nil {
		return translatePersistenceError(err, nil, nil)
	}
	token.ID = created.ID
	token.CreatedAt = created.CreatedAt
	token.UpdatedAt = created.UpdatedAt
	token.LastUsedAt = created.LastUsedAt
	token.RevokedAt = created.RevokedAt
	return nil
}

func (r *userAccessTokenRepository) GetByID(ctx context.Context, id int64) (*service.UserAccessToken, error) {
	m, err := r.client.UserAccessToken.Query().
		Where(useraccesstoken.IDEQ(id)).
		Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrUserAccessTokenNotFound, nil)
	}
	return userAccessTokenEntityToService(m), nil
}

func (r *userAccessTokenRepository) GetByTokenHash(ctx context.Context, hash string) (*service.UserAccessToken, error) {
	m, err := r.client.UserAccessToken.Query().
		Where(useraccesstoken.TokenHashEQ(hash)).
		Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrUserAccessTokenNotFound, nil)
	}
	return userAccessTokenEntityToService(m), nil
}

func (r *userAccessTokenRepository) ListByUserID(ctx context.Context, userID int64) ([]service.UserAccessToken, error) {
	rows, err := r.client.UserAccessToken.Query().
		Where(useraccesstoken.UserIDEQ(userID)).
		Order(dbent.Desc(useraccesstoken.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]service.UserAccessToken, 0, len(rows))
	for _, m := range rows {
		out = append(out, *userAccessTokenEntityToService(m))
	}
	return out, nil
}

func (r *userAccessTokenRepository) CountActiveByUserID(ctx context.Context, userID int64, now time.Time) (int, error) {
	n, err := r.client.UserAccessToken.Query().
		Where(
			useraccesstoken.UserIDEQ(userID),
			useraccesstoken.RevokedAtIsNil(),
			useraccesstoken.ExpiresAtGT(now),
		).
		Count(ctx)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func (r *userAccessTokenRepository) RevokeByIDForUser(ctx context.Context, userID, id int64, revokedAt time.Time) error {
	n, err := r.client.UserAccessToken.Update().
		Where(
			useraccesstoken.IDEQ(id),
			useraccesstoken.UserIDEQ(userID),
			useraccesstoken.RevokedAtIsNil(),
		).
		SetRevokedAt(revokedAt).
		Save(ctx)
	if err != nil {
		return err
	}
	if n == 0 {
		// Either missing, wrong owner, or already revoked — treat as not found for ownership isolation.
		// If already revoked by same user, still 404-ish is fine; client can refresh list.
		// Prefer: check existence for clearer already-revoked path.
		exists, err := r.client.UserAccessToken.Query().
			Where(useraccesstoken.IDEQ(id), useraccesstoken.UserIDEQ(userID)).
			Exist(ctx)
		if err != nil {
			return err
		}
		if !exists {
			return service.ErrUserAccessTokenNotFound
		}
		// already revoked → idempotent success
		return nil
	}
	return nil
}

func (r *userAccessTokenRepository) TouchLastUsedAt(ctx context.Context, id int64, usedAt time.Time) error {
	return r.client.UserAccessToken.UpdateOneID(id).
		SetLastUsedAt(usedAt).
		Exec(ctx)
}

func userAccessTokenEntityToService(m *dbent.UserAccessToken) *service.UserAccessToken {
	if m == nil {
		return nil
	}
	return &service.UserAccessToken{
		ID:          m.ID,
		UserID:      m.UserID,
		Name:        m.Name,
		TokenHash:   m.TokenHash,
		TokenPrefix: m.TokenPrefix,
		ExpiresAt:   m.ExpiresAt,
		LastUsedAt:  m.LastUsedAt,
		RevokedAt:   m.RevokedAt,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}
