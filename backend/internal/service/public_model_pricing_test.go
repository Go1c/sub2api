package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

type publicModelPricingRepoStub struct {
	values map[string]string
}

func (s *publicModelPricingRepoStub) Get(ctx context.Context, key string) (*Setting, error) {
	return nil, ErrSettingNotFound
}

func (s *publicModelPricingRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	if value, ok := s.values[key]; ok {
		return value, nil
	}
	return "", ErrSettingNotFound
}

func (s *publicModelPricingRepoStub) Set(ctx context.Context, key, value string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[key] = value
	return nil
}

func (s *publicModelPricingRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	return map[string]string{}, nil
}

func (s *publicModelPricingRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	return nil
}

func (s *publicModelPricingRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	return map[string]string{}, nil
}

func (s *publicModelPricingRepoStub) Delete(ctx context.Context, key string) error {
	delete(s.values, key)
	return nil
}

func TestDefaultPublicModelPricingConfigOnlyIncludesRequestedModels(t *testing.T) {
	cfg := DefaultPublicModelPricingConfig()
	want := []string{"Claude Opus 4.7", "Claude Sonnet 4.6", "GPT5.5", "GPT5.4"}

	if len(cfg.Rows) != len(want) {
		t.Fatalf("expected %d default rows, got %d", len(want), len(cfg.Rows))
	}
	for i, model := range want {
		if cfg.Rows[i].Model != model {
			t.Fatalf("row %d model = %q, want %q", i, cfg.Rows[i].Model, model)
		}
		if !cfg.Rows[i].Enabled {
			t.Fatalf("row %d should be enabled by default", i)
		}
	}
}

func TestSettingServiceGetPublicModelPricingFallsBackToDefaults(t *testing.T) {
	svc := NewSettingService(&publicModelPricingRepoStub{values: map[string]string{}}, &config.Config{})

	cfg, err := svc.GetPublicModelPricing(context.Background())
	if err != nil {
		t.Fatalf("GetPublicModelPricing returned error: %v", err)
	}
	if len(cfg.Rows) != 4 {
		t.Fatalf("expected 4 default rows, got %d", len(cfg.Rows))
	}
}

func TestSettingServiceSetPublicModelPricingRejectsEmptyModel(t *testing.T) {
	svc := NewSettingService(&publicModelPricingRepoStub{values: map[string]string{}}, &config.Config{})

	_, err := svc.SetPublicModelPricing(context.Background(), PublicModelPricingConfig{
		Rows: []PublicModelPricingRow{{Model: " "}},
	})
	if err == nil {
		t.Fatal("expected validation error for empty model")
	}
}

func TestPublicModelPricingEnabledRowsFiltersDisabledRows(t *testing.T) {
	cfg := PublicModelPricingConfig{
		Currency: "CNY",
		Unit:     "1M tokens",
		Rows: []PublicModelPricingRow{
			{Model: "Claude Opus 4.7", Enabled: true},
			{Model: "GPT5.4", Enabled: false},
		},
	}

	got := cfg.EnabledRows()
	if len(got.Rows) != 1 {
		t.Fatalf("expected 1 enabled row, got %d", len(got.Rows))
	}
	if got.Rows[0].Model != "Claude Opus 4.7" {
		t.Fatalf("enabled row model = %q", got.Rows[0].Model)
	}
}
