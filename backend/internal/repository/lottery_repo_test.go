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

func newLotteryEntRepo(t *testing.T) (service.LotteryRepository, *dbent.Client) {
	t.Helper()

	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", t.Name()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })

	return NewLotteryRepository(client), client
}

func mustCreateLotteryRepoUser(t *testing.T, ctx context.Context, client *dbent.Client, email string, role string) *dbent.User {
	t.Helper()
	user, err := client.User.Create().
		SetEmail(email).
		SetPasswordHash("hash").
		SetRole(role).
		SetStatus(service.StatusActive).
		Save(ctx)
	require.NoError(t, err)
	return user
}

func TestLotteryRepositoryCreateCampaignArchivesActive(t *testing.T) {
	repo, client := newLotteryEntRepo(t)
	ctx := context.Background()
	admin := mustCreateLotteryRepoUser(t, ctx, client, "lottery-admin@example.com", service.RoleAdmin)
	now := time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC)

	oldCampaign := &service.LotteryCampaign{
		Name:            "old",
		Subtitle:        "old",
		Status:          service.LotteryStatusActive,
		PrizeCount:      1,
		MaxParticipants: 2,
		CreatedBy:       admin.ID,
		CreatedAt:       now.Add(-time.Hour),
		UpdatedAt:       now.Add(-time.Hour),
	}
	require.NoError(t, repo.CreateCampaign(ctx, oldCampaign, []service.LotteryCode{{Code: "OLD", CreatedAt: now, UpdatedAt: now}}))
	require.NoError(t, repo.FinishActiveCampaigns(ctx, now))

	newCampaign := &service.LotteryCampaign{
		Name:            "new",
		Subtitle:        "new",
		Status:          service.LotteryStatusActive,
		PrizeCount:      2,
		MaxParticipants: 3,
		CreatedBy:       admin.ID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	require.NoError(t, repo.CreateCampaign(ctx, newCampaign, []service.LotteryCode{
		{Code: "A", CreatedAt: now, UpdatedAt: now},
		{Code: "B", CreatedAt: now, UpdatedAt: now},
	}))

	gotOld, err := repo.GetCampaign(ctx, oldCampaign.ID)
	require.NoError(t, err)
	require.Equal(t, service.LotteryStatusFinished, gotOld.Status)
	require.NotNil(t, gotOld.FinishedAt)

	active, err := repo.GetActiveCampaign(ctx)
	require.NoError(t, err)
	require.Equal(t, newCampaign.ID, active.ID)

	list, result, err := repo.ListCampaigns(ctx, pagination.PaginationParams{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, int64(2), result.Total)
	require.Len(t, list, 2)
	require.Equal(t, newCampaign.ID, list[0].ID)
}

func TestLotteryRepositoryDrawPersistence(t *testing.T) {
	repo, client := newLotteryEntRepo(t)
	ctx := context.Background()
	admin := mustCreateLotteryRepoUser(t, ctx, client, "lottery-admin-draw@example.com", service.RoleAdmin)
	user := mustCreateLotteryRepoUser(t, ctx, client, "lottery-user-draw@example.com", service.RoleUser)
	now := time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC)

	campaign := &service.LotteryCampaign{
		Name:            "draw",
		Subtitle:        "draw",
		Status:          service.LotteryStatusActive,
		PrizeCount:      1,
		MaxParticipants: 1,
		CreatedBy:       admin.ID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	require.NoError(t, repo.CreateCampaign(ctx, campaign, []service.LotteryCode{{Code: "WIN", CreatedAt: now, UpdatedAt: now}}))

	require.NoError(t, repo.WithTx(ctx, func(txCtx context.Context) error {
		code, err := repo.PickUnassignedCode(txCtx, campaign.ID)
		require.NoError(t, err)
		draw := &service.LotteryDraw{
			CampaignID:    campaign.ID,
			UserID:        user.ID,
			Won:           true,
			LotteryCodeID: &code.ID,
			ResultLabel:   "奖品 1",
			CreatedAt:     now,
		}
		require.NoError(t, repo.CreateDraw(txCtx, draw))
		require.NoError(t, repo.AssignCodeToDraw(txCtx, code.ID, user.ID, draw.ID, now))
		_, err = repo.IncrementCampaignCounters(txCtx, campaign.ID, 1, 1, true, &now)
		require.NoError(t, err)
		return nil
	}))

	got, err := repo.GetCampaign(ctx, campaign.ID)
	require.NoError(t, err)
	require.Equal(t, service.LotteryStatusFinished, got.Status)
	require.Equal(t, 1, got.JoinedCount)
	require.Equal(t, 1, got.WinnerCount)
	require.Len(t, got.Codes, 1)
	require.Equal(t, user.ID, *got.Codes[0].AssignedUserID)
	require.Len(t, got.Draws, 1)
	require.True(t, got.Draws[0].Won)
	require.NotNil(t, got.Draws[0].User)
	require.Equal(t, user.Email, got.Draws[0].User.Email)

	_, err = repo.GetDrawByCampaignAndUser(ctx, campaign.ID, user.ID)
	require.NoError(t, err)
	_, err = repo.PickUnassignedCode(ctx, campaign.ID)
	require.ErrorIs(t, err, service.ErrLotteryNoCodeAvailable)
}
