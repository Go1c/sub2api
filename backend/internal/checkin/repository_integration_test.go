//go:build integration

package checkin

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/migrations"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

var checkInIntegrationDB *sql.DB

type zeroRandom struct{}

func (zeroRandom) Int63n(int64) (int64, error) { return 0, nil }

func TestMain(m *testing.M) {
	ctx := context.Background()
	if exec.CommandContext(ctx, "docker", "info").Run() != nil {
		if os.Getenv("CI") != "" {
			log.Print("docker is not available (CI=true); failing check-in integration tests")
			os.Exit(1)
		}
		log.Print("docker is not available; skipping check-in integration tests")
		os.Exit(0)
	}

	container, err := tcpostgres.Run(
		ctx,
		"postgres:18.1-alpine3.23",
		tcpostgres.WithDatabase("checkin_test"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		log.Printf("start check-in postgres container: %v", err)
		os.Exit(1)
	}

	dsn, err := container.ConnectionString(ctx, "sslmode=disable", "TimeZone=UTC")
	if err != nil {
		log.Printf("get check-in postgres DSN: %v", err)
		os.Exit(1)
	}
	checkInIntegrationDB, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Printf("open check-in postgres: %v", err)
		os.Exit(1)
	}
	checkInIntegrationDB.SetMaxOpenConns(16)
	if err := checkInIntegrationDB.PingContext(ctx); err != nil {
		log.Printf("ping check-in postgres: %v", err)
		os.Exit(1)
	}
	if err := setupCheckInIntegrationSchema(ctx, checkInIntegrationDB); err != nil {
		log.Printf("setup check-in schema: %v", err)
		os.Exit(1)
	}

	code := m.Run()
	_ = checkInIntegrationDB.Close()
	_ = container.Terminate(ctx)
	os.Exit(code)
}

func setupCheckInIntegrationSchema(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE users (
			id BIGSERIAL PRIMARY KEY,
			email VARCHAR(255) NOT NULL,
			username VARCHAR(255) NOT NULL DEFAULT '',
			status VARCHAR(20) NOT NULL DEFAULT 'active',
			balance NUMERIC(20,8) NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			deleted_at TIMESTAMPTZ
		)`); err != nil {
		return err
	}
	migration, err := migrations.FS.ReadFile("929_daily_checkin.sql")
	if err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, string(migration)); err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `
		CREATE TABLE redeem_codes (
			id BIGSERIAL PRIMARY KEY,
			code VARCHAR(32) UNIQUE NOT NULL,
			type VARCHAR(20) NOT NULL,
			value NUMERIC(20,8) NOT NULL,
			status VARCHAR(20) NOT NULL,
			used_by BIGINT,
			used_at TIMESTAMPTZ,
			notes TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`)
	return err
}

func resetCheckInIntegrationState(t *testing.T, minReward, maxReward, dailyCap string) {
	t.Helper()
	// lib/pq 对带参数的语句走 prepared statement，会拒绝多命令；
	// TRUNCATE 与带参 UPDATE 必须拆成两次执行。
	_, err := checkInIntegrationDB.ExecContext(context.Background(), `
		TRUNCATE TABLE daily_checkin_records, daily_checkin_daily_counters, users, redeem_codes RESTART IDENTITY CASCADE`)
	require.NoError(t, err)
	_, err = checkInIntegrationDB.ExecContext(context.Background(), `
		UPDATE daily_checkin_settings
		SET enabled = TRUE,
			min_reward = $1,
			max_reward = $2,
			timezone = 'UTC',
			daily_cap = $3,
			milestones = '[]'::jsonb,
			updated_at = NOW()
		WHERE id = 1`, minReward, maxReward, dailyCap)
	require.NoError(t, err)
}

func createCheckInIntegrationUser(t *testing.T, email string) int64 {
	t.Helper()
	var userID int64
	err := checkInIntegrationDB.QueryRowContext(context.Background(), `
		INSERT INTO users (email, username)
		VALUES ($1, $2)
		RETURNING id`, email, "integration").Scan(&userID)
	require.NoError(t, err)
	return userID
}

func runConcurrentCheckIns(repo *sqlRepository, userIDs []int64, times []time.Time) ([]CheckInResult, []error) {
	results := make([]CheckInResult, len(userIDs))
	errors := make([]error, len(userIDs))
	start := make(chan struct{})
	var wait sync.WaitGroup
	for index := range userIDs {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			<-start
			results[index], errors[index] = repo.CheckIn(
				context.Background(),
				userIDs[index],
				times[index],
				ClientInfo{IP: "127.0.0.1", UserAgent: "integration"},
			)
		}(index)
	}
	close(start)
	wait.Wait()
	return results, errors
}

func TestRepositoryConcurrentDuplicateCheckInAwardsOnce(t *testing.T) {
	resetCheckInIntegrationState(t, "0.1", "0.1", "0")
	userID := createCheckInIntegrationUser(t, "duplicate@example.test")
	repo := newSQLRepository(checkInIntegrationDB, zeroRandom{})
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)

	results, errors := runConcurrentCheckIns(repo, []int64{userID, userID}, []time.Time{now, now})
	require.NoError(t, errors[0])
	require.NoError(t, errors[1])
	require.NotEqual(t, results[0].AlreadyCheckedIn, results[1].AlreadyCheckedIn)
	require.Equal(t, results[0].Record.ID, results[1].Record.ID)

	var recordCount int
	var balance string
	require.NoError(t, checkInIntegrationDB.QueryRow(`SELECT COUNT(*) FROM daily_checkin_records WHERE user_id = $1`, userID).Scan(&recordCount))
	require.NoError(t, checkInIntegrationDB.QueryRow(`SELECT balance::text FROM users WHERE id = $1`, userID).Scan(&balance))
	require.Equal(t, 1, recordCount)
	require.Equal(t, "0.10000000", balance)
}

func TestRepositoryConcurrentDailyBudgetNeverOverpays(t *testing.T) {
	resetCheckInIntegrationState(t, "1", "1", "1")
	firstUser := createCheckInIntegrationUser(t, "budget-1@example.test")
	secondUser := createCheckInIntegrationUser(t, "budget-2@example.test")
	repo := newSQLRepository(checkInIntegrationDB, zeroRandom{})
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)

	results, errors := runConcurrentCheckIns(repo, []int64{firstUser, secondUser}, []time.Time{now, now})
	require.NoError(t, errors[0])
	require.NoError(t, errors[1])
	statusCounts := map[string]int{}
	for _, result := range results {
		statusCounts[result.Record.Status]++
	}
	require.Equal(t, 1, statusCounts[StatusAwarded])
	require.Equal(t, 1, statusCounts[StatusBudgetExhausted])

	var awardedTotal, balances string
	require.NoError(t, checkInIntegrationDB.QueryRow(`SELECT awarded_total::text FROM daily_checkin_daily_counters`).Scan(&awardedTotal))
	require.NoError(t, checkInIntegrationDB.QueryRow(`SELECT SUM(balance)::text FROM users`).Scan(&balances))
	require.Equal(t, "1.00000000", awardedTotal)
	require.Equal(t, "1.00000000", balances)
}

func TestRepositoryRecordFailureRollsBackBalanceAndBudget(t *testing.T) {
	resetCheckInIntegrationState(t, "0.5", "0.5", "5")
	userID := createCheckInIntegrationUser(t, "rollback@example.test")
	_, err := checkInIntegrationDB.Exec(`
		CREATE FUNCTION reject_checkin_record() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN
			RAISE EXCEPTION 'forced check-in record failure';
		END;
		$$;
		CREATE TRIGGER reject_checkin_record_trigger
		BEFORE INSERT ON daily_checkin_records
		FOR EACH ROW EXECUTE FUNCTION reject_checkin_record()`)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = checkInIntegrationDB.Exec(`DROP TRIGGER IF EXISTS reject_checkin_record_trigger ON daily_checkin_records`)
		_, _ = checkInIntegrationDB.Exec(`DROP FUNCTION IF EXISTS reject_checkin_record()`)
	})

	repo := newSQLRepository(checkInIntegrationDB, zeroRandom{})
	_, err = repo.CheckIn(context.Background(), userID, time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC), ClientInfo{})
	require.ErrorContains(t, err, "forced check-in record failure")

	var balance string
	var records, counters, redeemCodes int
	require.NoError(t, checkInIntegrationDB.QueryRow(`SELECT balance::text FROM users WHERE id = $1`, userID).Scan(&balance))
	require.NoError(t, checkInIntegrationDB.QueryRow(`SELECT COUNT(*) FROM daily_checkin_records`).Scan(&records))
	require.NoError(t, checkInIntegrationDB.QueryRow(`SELECT COUNT(*) FROM daily_checkin_daily_counters`).Scan(&counters))
	require.NoError(t, checkInIntegrationDB.QueryRow(`SELECT COUNT(*) FROM redeem_codes`).Scan(&redeemCodes))
	require.Equal(t, "0.00000000", balance)
	require.Zero(t, records)
	require.Zero(t, counters)
	require.Zero(t, redeemCodes)
}

func TestRepositoryCheckInWritesUsedRedeemCode(t *testing.T) {
	resetCheckInIntegrationState(t, "0.1", "0.1", "0")
	userID := createCheckInIntegrationUser(t, "redeem@example.test")
	repo := newSQLRepository(checkInIntegrationDB, zeroRandom{})

	result, err := repo.CheckIn(context.Background(), userID, time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC), ClientInfo{})
	require.NoError(t, err)
	require.Equal(t, StatusAwarded, result.Record.Status)

	var count int
	var codeType, status, value, notes, code string
	var usedBy int64
	require.NoError(t, checkInIntegrationDB.QueryRow(`SELECT COUNT(*) FROM redeem_codes`).Scan(&count))
	require.Equal(t, 1, count)
	require.NoError(t, checkInIntegrationDB.QueryRow(`
		SELECT code, type, status, value::text, used_by, notes
		FROM redeem_codes`).Scan(&code, &codeType, &status, &value, &usedBy, &notes))
	require.Len(t, code, 32)
	require.Equal(t, RedeemTypeCheckinBalance, codeType)
	require.Equal(t, redeemCodeStatusUsed, status)
	require.Equal(t, "0.10000000", value)
	require.Equal(t, userID, usedBy)
	require.Equal(t, fmt.Sprintf("daily_checkin:%d", result.Record.ID), notes)
}

func TestRepositoryCheckInWritesSplitMilestoneRedeemCodes(t *testing.T) {
	resetCheckInIntegrationState(t, "0.1", "0.1", "0")
	_, err := checkInIntegrationDB.ExecContext(context.Background(), `
		UPDATE daily_checkin_settings
		SET milestones = '[{"day":1,"bonus":"2.0000"}]'::jsonb
		WHERE id = 1`)
	require.NoError(t, err)

	userID := createCheckInIntegrationUser(t, "milestone-redeem@example.test")
	repo := newSQLRepository(checkInIntegrationDB, zeroRandom{})
	result, err := repo.CheckIn(context.Background(), userID, time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC), ClientInfo{})
	require.NoError(t, err)
	require.Equal(t, StatusAwarded, result.Record.Status)
	require.Equal(t, 1, result.Record.MilestoneDay)
	require.Equal(t, "0.1000", formatAmount(result.Record.BaseReward))
	require.Equal(t, "2.0000", formatAmount(result.Record.MilestoneBonus))
	require.Equal(t, "2.1000", formatAmount(result.Record.ActualReward))

	rows, err := checkInIntegrationDB.Query(`
		SELECT code, type, status, value::text, used_by, notes
		FROM redeem_codes
		ORDER BY type`)
	require.NoError(t, err)
	defer rows.Close()

	type redeemRow struct {
		code     string
		codeType string
		status   string
		value    string
		usedBy   int64
		notes    string
	}
	var codes []redeemRow
	for rows.Next() {
		var row redeemRow
		require.NoError(t, rows.Scan(&row.code, &row.codeType, &row.status, &row.value, &row.usedBy, &row.notes))
		codes = append(codes, row)
	}
	require.NoError(t, rows.Err())
	require.Len(t, codes, 2)

	require.Equal(t, RedeemTypeCheckinBalance, codes[0].codeType)
	require.Equal(t, redeemCodeStatusUsed, codes[0].status)
	require.Equal(t, "0.10000000", codes[0].value)
	require.Equal(t, userID, codes[0].usedBy)
	require.Equal(t, fmt.Sprintf("daily_checkin:%d", result.Record.ID), codes[0].notes)
	require.Len(t, codes[0].code, 32)

	require.Equal(t, RedeemTypeCheckinMilestone, codes[1].codeType)
	require.Equal(t, redeemCodeStatusUsed, codes[1].status)
	require.Equal(t, "2.00000000", codes[1].value)
	require.Equal(t, userID, codes[1].usedBy)
	require.Equal(t, fmt.Sprintf("daily_checkin_milestone:%d:1", result.Record.ID), codes[1].notes)
	require.Len(t, codes[1].code, 32)

	var balance string
	require.NoError(t, checkInIntegrationDB.QueryRow(`SELECT balance::text FROM users WHERE id = $1`, userID).Scan(&balance))
	require.Equal(t, "2.10000000", balance)
}

func TestRepositoryConcurrentRewardsAtomicallyIncreaseBalance(t *testing.T) {
	resetCheckInIntegrationState(t, "1", "1", "0")
	userID := createCheckInIntegrationUser(t, "atomic@example.test")
	repo := newSQLRepository(checkInIntegrationDB, zeroRandom{})
	times := []time.Time{
		time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC),
		time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC),
	}

	_, errors := runConcurrentCheckIns(repo, []int64{userID, userID}, times)
	for index, err := range errors {
		require.NoError(t, err, fmt.Sprintf("check-in %d", index))
	}
	var balance string
	var records int
	require.NoError(t, checkInIntegrationDB.QueryRow(`SELECT balance::text FROM users WHERE id = $1`, userID).Scan(&balance))
	require.NoError(t, checkInIntegrationDB.QueryRow(`SELECT COUNT(*) FROM daily_checkin_records WHERE user_id = $1`, userID).Scan(&records))
	require.Equal(t, "2.00000000", balance)
	require.Equal(t, 2, records)
}
