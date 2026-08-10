package repository

import (
	"context"
	stdsql "database/sql"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/setting"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// settingsRepoLogComponent 标识本文件所有日志的来源，便于按 component 过滤。
const settingsRepoLogComponent = "repository.setting"

// pgUniqueViolation 是 PostgreSQL 唯一约束冲突的 SQLSTATE。
// 出现在 UPDATE 路径上时被视为生产异常（trigger/rule/索引状态等），需要降级到 upsert 并大声告警。
const pgUniqueViolation = "23505"

const (
	settingBatchSavepointSQL = `SAVEPOINT sub2api_settings_batch`
	settingBatchRollbackSQL  = `ROLLBACK TO SAVEPOINT sub2api_settings_batch`
	settingBatchReleaseSQL   = `RELEASE SAVEPOINT sub2api_settings_batch`
)

type settingRepository struct {
	client *ent.Client
}

func NewSettingRepository(client *ent.Client) service.SettingRepository {
	return &settingRepository{client: client}
}

func (r *settingRepository) Get(ctx context.Context, key string) (*service.Setting, error) {
	m, err := r.client.Setting.Query().Where(setting.KeyEQ(key)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, service.ErrSettingNotFound
		}
		return nil, err
	}
	return &service.Setting{
		ID:        m.ID,
		Key:       m.Key,
		Value:     m.Value,
		UpdatedAt: m.UpdatedAt,
	}, nil
}

func (r *settingRepository) GetValue(ctx context.Context, key string) (string, error) {
	setting, err := r.Get(ctx, key)
	if err != nil {
		return "", err
	}
	return setting.Value, nil
}

func (r *settingRepository) Set(ctx context.Context, key, value string) error {
	return r.SetMultiple(ctx, map[string]string{key: value})
}

func (r *settingRepository) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	if len(keys) == 0 {
		return map[string]string{}, nil
	}
	settings, err := r.client.Setting.Query().Where(setting.KeyIn(keys...)).All(ctx)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, s := range settings {
		result[s.Key] = s.Value
	}
	return result, nil
}

func (r *settingRepository) SetMultiple(ctx context.Context, settings map[string]string) error {
	if len(settings) == 0 {
		return nil
	}

	keys := make([]string, 0, len(settings))
	for key := range settings {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	now := time.Now()

	if tx := ent.TxFromContext(ctx); tx != nil {
		logger.LegacyPrintf(settingsRepoLogComponent,
			"SetMultiple path=existing-tx keyCount=%d firstKey=%q", len(keys), keys[0])
		return setMultipleWithClient(ctx, tx.Client(), keys, settings, now)
	}

	tx, err := r.client.Tx(ctx)
	if err != nil {
		if errors.Is(err, ent.ErrTxStarted) {
			logger.LegacyPrintf(settingsRepoLogComponent,
				"SetMultiple path=tx-already-started-fallback keyCount=%d firstKey=%q", len(keys), keys[0])
			return setMultipleWithClient(ctx, r.client, keys, settings, now)
		}
		return err
	}
	defer func() { _ = tx.Rollback() }()

	logger.LegacyPrintf(settingsRepoLogComponent,
		"SetMultiple path=new-tx keyCount=%d firstKey=%q", len(keys), keys[0])
	if err := setMultipleWithClient(ent.NewTxContext(ctx, tx), tx.Client(), keys, settings, now); err != nil {
		return err
	}
	return tx.Commit()
}

// setMultipleWithClient 对每个 key 先 UPDATE，0 行才 INSERT ON CONFLICT。
// 整批写入放在 savepoint 内：PostgreSQL 的任意语句错误都会中止当前事务，
// 必须先 ROLLBACK TO SAVEPOINT，才能在 23505 异常路径上执行 raw upsert fallback。
func setMultipleWithClient(ctx context.Context, client *ent.Client, keys []string, settings map[string]string, now time.Time) error {
	if err := execSettingControlSQL(ctx, client, settingBatchSavepointSQL); err != nil {
		return fmt.Errorf("create settings batch savepoint: %w", err)
	}

	for _, key := range keys {
		if err := writeSettingWithUpdateFirst(ctx, client, key, settings[key], now); err != nil {
			if rollbackErr := execSettingControlSQL(ctx, client, settingBatchRollbackSQL); rollbackErr != nil {
				return errors.Join(err, fmt.Errorf("rollback settings batch savepoint: %w", rollbackErr))
			}
			if !isUniqueViolation(err) {
				return errors.Join(err, releaseSettingBatchSavepoint(ctx, client))
			}

			logger.LegacyPrintf(settingsRepoLogComponent,
				"WARN: UPDATE on settings returned unique violation (sqlstate=%s, msg=%v) — "+
					"rolled back batch savepoint and retrying with raw upserts. "+
					"This indicates an unexpected DB-side trigger, rule, or index state; please audit pg_trigger/pg_rules/pg_index for table=settings.",
				pgUniqueViolation, err)
			for _, fallbackKey := range keys {
				if fallbackErr := upsertSettingRaw(ctx, client, fallbackKey, settings[fallbackKey], now); fallbackErr != nil {
					rollbackErr := execSettingControlSQL(ctx, client, settingBatchRollbackSQL)
					return errors.Join(fallbackErr, rollbackErr, releaseSettingBatchSavepoint(ctx, client))
				}
			}
			return releaseSettingBatchSavepoint(ctx, client)
		}
	}
	return releaseSettingBatchSavepoint(ctx, client)
}

// writeSettingWithUpdateFirst 写一个 setting key，UPDATE-first，未命中时 upsert。
func writeSettingWithUpdateFirst(ctx context.Context, client *ent.Client, key, value string, now time.Time) error {
	result, err := execSettingSQL(ctx, client,
		`UPDATE "settings" SET "value" = $1, "updated_at" = $2 WHERE "key" = $3`,
		value, now, key,
	)
	if err != nil {
		return fmt.Errorf("update setting %q: %w", key, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("update setting %q rows affected: %w", key, err)
	}
	if rowsAffected > 0 {
		return nil
	}

	return upsertSettingRaw(ctx, client, key, value, now)
}

// upsertSettingRaw 走原生 INSERT ... ON CONFLICT，幂等。
// 生产 DB 已验证该语句正确，可作为兜底写入路径。
func upsertSettingRaw(ctx context.Context, client *ent.Client, key, value string, now time.Time) error {
	if _, err := execSettingSQL(ctx, client,
		`INSERT INTO "settings" ("key", "value", "updated_at") VALUES ($1, $2, $3) ON CONFLICT ("key") DO UPDATE SET "value" = EXCLUDED."value", "updated_at" = EXCLUDED."updated_at"`,
		key, value, now,
	); err != nil {
		return fmt.Errorf("insert setting %q: %w", key, err)
	}
	return nil
}

func execSettingControlSQL(ctx context.Context, client *ent.Client, query string) error {
	var result stdsql.Result
	if err := client.Driver().Exec(ctx, query, []any{}, &result); err != nil {
		return err
	}
	return nil
}

func releaseSettingBatchSavepoint(ctx context.Context, client *ent.Client) error {
	if err := execSettingControlSQL(ctx, client, settingBatchReleaseSQL); err != nil {
		return fmt.Errorf("release settings batch savepoint: %w", err)
	}
	return nil
}

func execSettingSQL(ctx context.Context, client *ent.Client, query string, args ...any) (stdsql.Result, error) {
	var result stdsql.Result
	if err := client.Driver().Exec(ctx, query, args, &result); err != nil {
		return nil, err
	}
	if result == nil {
		return nil, fmt.Errorf("settings SQL returned nil result")
	}
	return result, nil
}

func (r *settingRepository) GetAll(ctx context.Context) (map[string]string, error) {
	settings, err := r.client.Setting.Query().All(ctx)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, s := range settings {
		result[s.Key] = s.Value
	}
	return result, nil
}

func (r *settingRepository) Delete(ctx context.Context, key string) error {
	_, err := r.client.Setting.Delete().Where(setting.KeyEQ(key)).Exec(ctx)
	return err
}
