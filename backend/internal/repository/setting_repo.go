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
	"github.com/Wei-Shaw/sub2api/internal/service"
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
		return setMultipleWithClient(ctx, tx.Client(), keys, settings, now)
	}

	tx, err := r.client.Tx(ctx)
	if err != nil {
		if errors.Is(err, ent.ErrTxStarted) {
			return setMultipleWithClient(ctx, r.client, keys, settings, now)
		}
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if err := setMultipleWithClient(ent.NewTxContext(ctx, tx), tx.Client(), keys, settings, now); err != nil {
		return err
	}
	return tx.Commit()
}

func setMultipleWithClient(ctx context.Context, client *ent.Client, keys []string, settings map[string]string, now time.Time) error {
	for _, key := range keys {
		result, err := execSettingSQL(ctx, client,
			`UPDATE "settings" SET "value" = $1, "updated_at" = $2 WHERE "key" = $3`,
			settings[key], now, key,
		)
		if err != nil {
			return fmt.Errorf("update setting %q: %w", key, err)
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("update setting %q rows affected: %w", key, err)
		}
		if rowsAffected > 0 {
			continue
		}

		if _, err := execSettingSQL(ctx, client,
			`INSERT INTO "settings" ("key", "value", "updated_at") VALUES ($1, $2, $3) ON CONFLICT ("key") DO UPDATE SET "value" = EXCLUDED."value", "updated_at" = EXCLUDED."updated_at"`,
			key, settings[key], now,
		); err != nil {
			return fmt.Errorf("insert setting %q: %w", key, err)
		}
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
