package repository

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

func newSiteMessageEntRepo(t *testing.T) (service.SiteMessageRepository, *dbent.Client) {
	t.Helper()

	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", t.Name()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })

	return NewSiteMessageRepository(client), client
}

func mustCreateSiteMessageRepoUser(t *testing.T, ctx context.Context, client *dbent.Client, email string) *dbent.User {
	t.Helper()

	user, err := client.User.Create().
		SetEmail(email).
		SetPasswordHash("hash").
		SetRole(service.RoleUser).
		SetStatus(service.StatusActive).
		Save(ctx)
	require.NoError(t, err)
	return user
}

func TestSiteMessageRepositoryCreateListAndVisibility(t *testing.T) {
	repo, client := newSiteMessageEntRepo(t)
	ctx := context.Background()
	alice := mustCreateSiteMessageRepoUser(t, ctx, client, "alice-site-message@example.com")
	bob := mustCreateSiteMessageRepoUser(t, ctx, client, "bob-site-message@example.com")
	outsider := mustCreateSiteMessageRepoUser(t, ctx, client, "outsider-site-message@example.com")
	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)

	old := &service.SiteMessage{
		SenderID:    alice.ID,
		RecipientID: bob.ID,
		Subject:     "old",
		Content:     "body",
		CreatedAt:   now.AddDate(0, 0, -31),
		UpdatedAt:   now.AddDate(0, 0, -31),
	}
	require.NoError(t, repo.Create(ctx, old))

	newer := &service.SiteMessage{
		SenderID:    alice.ID,
		RecipientID: bob.ID,
		Subject:     "newer",
		Content:     "body",
		CreatedAt:   now.Add(-time.Hour),
		UpdatedAt:   now.Add(-time.Hour),
	}
	require.NoError(t, repo.Create(ctx, newer))
	require.NotZero(t, newer.ID)

	parentID := newer.ID
	reply := &service.SiteMessage{
		SenderID:    bob.ID,
		RecipientID: alice.ID,
		ParentID:    &parentID,
		Subject:     "Re: newer",
		Content:     "reply",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	require.NoError(t, repo.Create(ctx, reply))

	cutoff := now.AddDate(0, 0, -30)
	inbox, result, err := repo.ListInbox(ctx, bob.ID, pagination.PaginationParams{Page: 1, PageSize: 10}, cutoff)
	require.NoError(t, err)
	require.Equal(t, int64(1), result.Total)
	require.Len(t, inbox, 1)
	require.Equal(t, "newer", inbox[0].Subject)
	require.NotNil(t, inbox[0].Sender)
	require.Equal(t, alice.Email, inbox[0].Sender.Email)

	sent, result, err := repo.ListSent(ctx, alice.ID, pagination.PaginationParams{Page: 1, PageSize: 10}, cutoff)
	require.NoError(t, err)
	require.Equal(t, int64(1), result.Total)
	require.Len(t, sent, 1)
	require.Equal(t, newer.ID, sent[0].ID)

	got, err := repo.GetVisibleByID(ctx, newer.ID, bob.ID, cutoff)
	require.NoError(t, err)
	require.Equal(t, newer.ID, got.ID)
	require.NotNil(t, got.Recipient)
	require.Equal(t, bob.Email, got.Recipient.Email)

	_, err = repo.GetVisibleByID(ctx, newer.ID, outsider.ID, cutoff)
	require.ErrorIs(t, err, service.ErrSiteMessageNotFound)

	_, err = repo.GetVisibleByID(ctx, old.ID, bob.ID, cutoff)
	require.ErrorIs(t, err, service.ErrSiteMessageNotFound)
}

func TestSiteMessageRepositoryReadCountsAndCleanup(t *testing.T) {
	repo, client := newSiteMessageEntRepo(t)
	ctx := context.Background()
	alice := mustCreateSiteMessageRepoUser(t, ctx, client, "alice-unread@example.com")
	bob := mustCreateSiteMessageRepoUser(t, ctx, client, "bob-unread@example.com")
	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	cutoff := now.AddDate(0, 0, -30)

	unread := &service.SiteMessage{
		SenderID:    alice.ID,
		RecipientID: bob.ID,
		Subject:     "unread",
		Content:     "body",
		CreatedAt:   now.Add(-time.Hour),
		UpdatedAt:   now.Add(-time.Hour),
	}
	require.NoError(t, repo.Create(ctx, unread))

	readAt := now.Add(-2 * time.Hour)
	read := &service.SiteMessage{
		SenderID:    alice.ID,
		RecipientID: bob.ID,
		Subject:     "read",
		Content:     "body",
		ReadAt:      &readAt,
		CreatedAt:   now.Add(-2 * time.Hour),
		UpdatedAt:   now.Add(-2 * time.Hour),
	}
	require.NoError(t, repo.Create(ctx, read))

	expired := &service.SiteMessage{
		SenderID:    alice.ID,
		RecipientID: bob.ID,
		Subject:     "expired",
		Content:     "body",
		CreatedAt:   now.AddDate(0, 0, -31),
		UpdatedAt:   now.AddDate(0, 0, -31),
	}
	require.NoError(t, repo.Create(ctx, expired))

	count, err := repo.CountUnread(ctx, bob.ID, cutoff)
	require.NoError(t, err)
	require.Equal(t, int64(1), count)

	markedAt := now.Add(time.Minute)
	require.NoError(t, repo.MarkRead(ctx, unread.ID, bob.ID, markedAt))
	require.NoError(t, repo.MarkRead(ctx, unread.ID, bob.ID, markedAt.Add(time.Hour)))

	count, err = repo.CountUnread(ctx, bob.ID, cutoff)
	require.NoError(t, err)
	require.Equal(t, int64(0), count)

	sentSince, err := repo.CountSentSince(ctx, alice.ID, now.AddDate(0, 0, -1))
	require.NoError(t, err)
	require.Equal(t, int64(2), sentSince)

	deleted, err := repo.DeleteOlderThan(ctx, cutoff)
	require.NoError(t, err)
	require.Equal(t, int64(1), deleted)

	_, err = repo.GetVisibleByID(ctx, expired.ID, bob.ID, time.Time{})
	require.ErrorIs(t, err, service.ErrSiteMessageNotFound)
}
