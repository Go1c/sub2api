package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type accountUsageCodexProbeRepo struct {
	stubOpenAIAccountRepo
	updateExtraCh chan map[string]any
	rateLimitCh   chan time.Time
}

func (r *accountUsageCodexProbeRepo) UpdateExtra(_ context.Context, _ int64, updates map[string]any) error {
	if r.updateExtraCh != nil {
		copied := make(map[string]any, len(updates))
		for k, v := range updates {
			copied[k] = v
		}
		r.updateExtraCh <- copied
	}
	return nil
}

func (r *accountUsageCodexProbeRepo) SetRateLimited(_ context.Context, _ int64, resetAt time.Time) error {
	if r.rateLimitCh != nil {
		r.rateLimitCh <- resetAt
	}
	return nil
}

func TestShouldRefreshOpenAICodexSnapshot(t *testing.T) {
	t.Parallel()

	rateLimitedUntil := time.Now().Add(5 * time.Minute)
	now := time.Now()
	usage := &UsageInfo{
		FiveHour: &UsageProgress{Utilization: 0},
		SevenDay: &UsageProgress{Utilization: 0},
	}

	if !shouldRefreshOpenAICodexSnapshot(&Account{RateLimitResetAt: &rateLimitedUntil}, usage, now) {
		t.Fatal("expected rate-limited account to force codex snapshot refresh")
	}

	if shouldRefreshOpenAICodexSnapshot(&Account{}, usage, now) {
		t.Fatal("expected complete non-rate-limited usage to skip codex snapshot refresh")
	}

	if !shouldRefreshOpenAICodexSnapshot(&Account{}, &UsageInfo{FiveHour: nil, SevenDay: &UsageProgress{}}, now) {
		t.Fatal("expected missing 5h snapshot to require refresh")
	}

	staleAt := now.Add(-(openAIProbeCacheTTL + time.Minute)).Format(time.RFC3339)
	if !shouldRefreshOpenAICodexSnapshot(&Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Extra: map[string]any{
			"openai_oauth_responses_websockets_v2_enabled": true,
			"codex_usage_updated_at":                       staleAt,
		},
	}, usage, now) {
		t.Fatal("expected stale ws snapshot to trigger refresh")
	}
}

func TestExtractOpenAICodexProbeUpdatesAccepts429WithCodexHeaders(t *testing.T) {
	t.Parallel()

	headers := make(http.Header)
	headers.Set("x-codex-primary-used-percent", "100")
	headers.Set("x-codex-primary-reset-after-seconds", "604800")
	headers.Set("x-codex-primary-window-minutes", "10080")
	// secondary maps to the 5h window when window minutes are known (300 vs 10080).
	headers.Set("x-codex-secondary-used-percent", "100")
	headers.Set("x-codex-secondary-reset-after-seconds", "18000")
	headers.Set("x-codex-secondary-window-minutes", "300")

	updates, err := extractOpenAICodexProbeUpdates(&http.Response{StatusCode: http.StatusTooManyRequests, Header: headers})
	if err != nil {
		t.Fatalf("extractOpenAICodexProbeUpdates() error = %v", err)
	}
	if len(updates) == 0 {
		t.Fatal("expected codex probe updates from 429 headers")
	}
	if got := updates["codex_5h_used_percent"]; got != 100.0 {
		t.Fatalf("codex_5h_used_percent = %v, want 100", got)
	}
	if got := updates["codex_7d_used_percent"]; got != 100.0 {
		t.Fatalf("codex_7d_used_percent = %v, want 100", got)
	}
}

func TestBuildCodexUsageProgressFromExtra_UsesCanonicalUsedPercent(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 5, 30, 7, 4, 9, 0, time.UTC)
	extra := map[string]any{
		"codex_5h_used_percent": 94.0,
		"codex_5h_reset_at":     now.Add(2 * time.Hour).Format(time.RFC3339),
		"codex_7d_used_percent": 93.0,
		"codex_7d_reset_at":     now.Add(5 * 24 * time.Hour).Format(time.RFC3339),
	}

	fiveHour := buildCodexUsageProgressFromExtra(extra, "5h", now)
	if fiveHour == nil {
		t.Fatal("expected non-nil 5h progress")
	}
	if fiveHour.Utilization != 94.0 {
		t.Fatalf("5h Utilization = %v, want 94", fiveHour.Utilization)
	}

	sevenDay := buildCodexUsageProgressFromExtra(extra, "7d", now)
	if sevenDay == nil {
		t.Fatal("expected non-nil 7d progress")
	}
	if sevenDay.Utilization != 93.0 {
		t.Fatalf("7d Utilization = %v, want 93", sevenDay.Utilization)
	}
}

func TestAccountUsageService_PersistOpenAICodexProbeSnapshotOnlyUpdatesExtra(t *testing.T) {
	t.Parallel()

	repo := &accountUsageCodexProbeRepo{
		updateExtraCh: make(chan map[string]any, 1),
		rateLimitCh:   make(chan time.Time, 1),
	}
	svc := &AccountUsageService{accountRepo: repo}
	svc.persistOpenAICodexProbeSnapshot(321, map[string]any{
		"codex_7d_used_percent": 100.0,
		"codex_7d_reset_at":     time.Now().Add(2 * time.Hour).UTC().Truncate(time.Second).Format(time.RFC3339),
	})

	select {
	case updates := <-repo.updateExtraCh:
		if got := updates["codex_7d_used_percent"]; got != 100.0 {
			t.Fatalf("codex_7d_used_percent = %v, want 100", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("等待 codex 探测快照写入 extra 超时")
	}

	select {
	case got := <-repo.rateLimitCh:
		t.Fatalf("不应将探测快照写入运行时限流状态: %v", got)
	case <-time.After(200 * time.Millisecond):
	}
}

func TestAccountUsageService_GetOpenAIUsage_DoesNotPromoteCodexExtraToRateLimit(t *testing.T) {
	t.Parallel()

	resetAt := time.Now().Add(6 * 24 * time.Hour).UTC().Truncate(time.Second)
	repo := &accountUsageCodexProbeRepo{
		rateLimitCh: make(chan time.Time, 1),
	}
	svc := &AccountUsageService{accountRepo: repo}
	account := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Extra: map[string]any{
			"codex_5h_used_percent": 1.0,
			"codex_5h_reset_at":     time.Now().Add(2 * time.Hour).UTC().Truncate(time.Second).Format(time.RFC3339),
			"codex_7d_used_percent": 100.0,
			"codex_7d_reset_at":     resetAt.Format(time.RFC3339),
		},
	}

	usage, err := svc.getOpenAIUsage(context.Background(), account, false)
	if err != nil {
		t.Fatalf("getOpenAIUsage() error = %v", err)
	}
	if usage.SevenDay == nil || usage.SevenDay.Utilization != 100.0 {
		t.Fatalf("预期 7 天用量仍然可见，实际为 %#v", usage.SevenDay)
	}
	if account.RateLimitResetAt != nil {
		t.Fatalf("不应让已耗尽的 codex extra 改写运行时限流状态: %v", account.RateLimitResetAt)
	}
	select {
	case got := <-repo.rateLimitCh:
		t.Fatalf("不应将已耗尽的 codex extra 持久化为运行时限流状态: %v", got)
	case <-time.After(200 * time.Millisecond):
	}
}

func TestBuildCodexUsageProgressFromExtra_ZerosExpiredWindow(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 3, 16, 12, 0, 0, 0, time.UTC)

	t.Run("expired 5h window zeroes utilization", func(t *testing.T) {
		extra := map[string]any{
			"codex_5h_used_percent": 42.0,
			"codex_5h_reset_at":     "2026-03-16T10:00:00Z", // 2h ago
		}
		progress := buildCodexUsageProgressFromExtra(extra, "5h", now)
		if progress == nil {
			t.Fatal("expected non-nil progress")
			return
		}
		if progress.Utilization != 0 {
			t.Fatalf("expected Utilization=0 for expired window, got %v", progress.Utilization)
		}
		if progress.RemainingSeconds != 0 {
			t.Fatalf("expected RemainingSeconds=0, got %v", progress.RemainingSeconds)
		}
	})

	t.Run("active 5h window keeps utilization", func(t *testing.T) {
		resetAt := now.Add(2 * time.Hour).Format(time.RFC3339)
		extra := map[string]any{
			"codex_5h_used_percent": 42.0,
			"codex_5h_reset_at":     resetAt,
		}
		progress := buildCodexUsageProgressFromExtra(extra, "5h", now)
		if progress == nil {
			t.Fatal("expected non-nil progress")
			return
		}
		if progress.Utilization != 42.0 {
			t.Fatalf("expected Utilization=42, got %v", progress.Utilization)
		}
	})

	t.Run("expired 7d window zeroes utilization", func(t *testing.T) {
		extra := map[string]any{
			"codex_7d_used_percent": 88.0,
			"codex_7d_reset_at":     "2026-03-15T00:00:00Z", // yesterday
		}
		progress := buildCodexUsageProgressFromExtra(extra, "7d", now)
		if progress == nil {
			t.Fatal("expected non-nil progress")
			return
		}
		if progress.Utilization != 0 {
			t.Fatalf("expected Utilization=0 for expired 7d window, got %v", progress.Utilization)
		}
	})
}

func TestAccountUsageService_GetUsage_AttachesUpstreamBalanceForEnabledAPIKey(t *testing.T) {
	t.Parallel()

	var gotAuth string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/dashboard/billing/credit_grants" {
			http.NotFound(w, r)
			return
		}
		gotAuth = r.Header.Get("x-api-key")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"total_available": 12.34,
		})
	}))
	defer upstream.Close()

	repo := &stubOpenAIAccountRepo{
		accounts: []Account{
			{
				ID:       987,
				Platform: PlatformAnthropic,
				Type:     AccountTypeAPIKey,
				Credentials: map[string]any{
					"base_url": upstream.URL,
					"api_key":  "sk-ant-test",
				},
				Extra: map[string]any{
					"upstream_balance_enabled": true,
				},
			},
		},
	}
	svc := &AccountUsageService{accountRepo: repo}

	usage, err := svc.GetUsage(context.Background(), 987)
	if err != nil {
		t.Fatalf("GetUsage() error = %v", err)
	}
	if gotAuth != "sk-ant-test" {
		t.Fatalf("expected x-api-key header, got %q", gotAuth)
	}
	if usage.UpstreamBalance == nil {
		t.Fatal("expected upstream balance result")
	}
	if !usage.UpstreamBalance.Success {
		t.Fatalf("expected upstream balance success, got %#v", usage.UpstreamBalance)
	}
	if usage.UpstreamBalance.Balance == nil || *usage.UpstreamBalance.Balance != 12.34 {
		t.Fatalf("expected parsed balance 12.34, got %#v", usage.UpstreamBalance.Balance)
	}
}

func TestAccountUsageService_GetUsage_ContinuesPastHTMLToUsageEndpoint(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dashboard/billing/credit_grants":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(`<!doctype html><html lang="zh"><body>dashboard</body></html>`))
		case "/v1/usage":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"wallet_balance": 45.67,
					"unit":           "USD",
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()

	repo := &stubOpenAIAccountRepo{
		accounts: []Account{
			{
				ID:       988,
				Platform: PlatformAnthropic,
				Type:     AccountTypeAPIKey,
				Credentials: map[string]any{
					"base_url": upstream.URL,
					"api_key":  "sk-ant-test",
				},
				Extra: map[string]any{
					"upstream_balance_enabled": true,
				},
			},
		},
	}
	svc := &AccountUsageService{accountRepo: repo}

	usage, err := svc.GetUsage(context.Background(), 988)
	if err != nil {
		t.Fatalf("GetUsage() error = %v", err)
	}
	if usage.UpstreamBalance == nil {
		t.Fatal("expected upstream balance result")
	}
	if !usage.UpstreamBalance.Success {
		t.Fatalf("expected upstream balance success, got %#v", usage.UpstreamBalance)
	}
	if usage.UpstreamBalance.Path != "/v1/usage" {
		t.Fatalf("expected /v1/usage probe to win, got %q", usage.UpstreamBalance.Path)
	}
	if usage.UpstreamBalance.Balance == nil || *usage.UpstreamBalance.Balance != 45.67 {
		t.Fatalf("expected parsed wallet balance 45.67, got %#v", usage.UpstreamBalance.Balance)
	}
}

func TestAccountUsageService_GetUsage_StripsV1BaseURLForUsageEndpoint(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/usage" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"account_balance": 8.9,
			"currency":        "USD",
		})
	}))
	defer upstream.Close()

	repo := &stubOpenAIAccountRepo{
		accounts: []Account{
			{
				ID:       989,
				Platform: PlatformOpenAI,
				Type:     AccountTypeAPIKey,
				Credentials: map[string]any{
					"base_url": upstream.URL + "/v1",
					"api_key":  "sk-openai-test",
				},
				Extra: map[string]any{
					"upstream_balance_enabled": true,
				},
			},
		},
	}
	svc := &AccountUsageService{accountRepo: repo}

	usage, err := svc.GetUsage(context.Background(), 989)
	if err != nil {
		t.Fatalf("GetUsage() error = %v", err)
	}
	if usage.UpstreamBalance == nil {
		t.Fatal("expected upstream balance result")
	}
	if !usage.UpstreamBalance.Success {
		t.Fatalf("expected upstream balance success, got %#v", usage.UpstreamBalance)
	}
	if usage.UpstreamBalance.Balance == nil || *usage.UpstreamBalance.Balance != 8.9 {
		t.Fatalf("expected parsed account balance 8.9, got %#v", usage.UpstreamBalance.Balance)
	}
}

func TestAccountUsageService_GetUsage_DoesNotTreatNewAPITokenQuotaAsUserBalance(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/usage/token/":
		default:
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer sk-newapi-test" {
			http.Error(w, "missing token auth", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data": map[string]any{
				"object":              "token_usage",
				"total_available":     123456,
				"total_used":          1000,
				"unlimited_quota":     false,
				"total_usd_available": 0.246912,
			},
		})
	}))
	defer upstream.Close()

	repo := &stubOpenAIAccountRepo{
		accounts: []Account{
			{
				ID:       990,
				Platform: PlatformOpenAI,
				Type:     AccountTypeAPIKey,
				Credentials: map[string]any{
					"base_url": upstream.URL,
					"api_key":  "sk-newapi-test",
				},
				Extra: map[string]any{
					"upstream_balance_enabled": true,
				},
			},
		},
	}
	svc := &AccountUsageService{accountRepo: repo}

	usage, err := svc.GetUsage(context.Background(), 990)
	if err != nil {
		t.Fatalf("GetUsage() error = %v", err)
	}
	if usage.UpstreamBalance == nil {
		t.Fatal("expected upstream balance result")
	}
	if usage.UpstreamBalance.Success {
		t.Fatalf("token quota must not be reported as user balance: %#v", usage.UpstreamBalance)
	}
	if usage.UpstreamBalance.Path != "/api/usage/token/" {
		t.Fatalf("expected /api/usage/token/ diagnostic, got %q", usage.UpstreamBalance.Path)
	}
	if usage.UpstreamBalance.Balance != nil {
		t.Fatalf("expected no user balance from token quota, got %#v", usage.UpstreamBalance.Balance)
	}
	if usage.UpstreamBalance.Message == "" {
		t.Fatal("expected diagnostic message")
	}
	if usage.UpstreamBalance.Message != newAPITokenQuotaNotUserBalanceMessage {
		t.Fatalf("expected concise credential diagnostic, got %q", usage.UpstreamBalance.Message)
	}
	if usage.UpstreamBalance.StatusCode != 0 {
		t.Fatalf("expected no HTTP status for credential diagnostic, got %d", usage.UpstreamBalance.StatusCode)
	}
	if usage.UpstreamBalance.Raw != "" {
		t.Fatalf("expected token quota raw body to be hidden, got %q", usage.UpstreamBalance.Raw)
	}
}

func TestAccountUsageService_GetUsage_NewAPIProviderRequiresUserBalanceCredentials(t *testing.T) {
	t.Parallel()

	repo := &stubOpenAIAccountRepo{
		accounts: []Account{
			{
				ID:       995,
				Platform: PlatformOpenAI,
				Type:     AccountTypeAPIKey,
				Credentials: map[string]any{
					"base_url": "https://newapi.example",
					"api_key":  "sk-newapi-test",
				},
				Extra: map[string]any{
					"upstream_balance_enabled":  true,
					"upstream_balance_provider": "newapi",
				},
			},
		},
	}
	svc := &AccountUsageService{accountRepo: repo}

	usage, err := svc.GetUsage(context.Background(), 995)
	if err != nil {
		t.Fatalf("GetUsage() error = %v", err)
	}
	if usage.UpstreamBalance == nil {
		t.Fatal("expected upstream balance result")
	}
	if usage.UpstreamBalance.Success {
		t.Fatalf("expected credential diagnostic, got %#v", usage.UpstreamBalance)
	}
	if usage.UpstreamBalance.Path != "" || usage.UpstreamBalance.StatusCode != 0 || usage.UpstreamBalance.Raw != "" {
		t.Fatalf("expected local diagnostic without probing token quota, got %#v", usage.UpstreamBalance)
	}
	if usage.UpstreamBalance.Message != newAPITokenQuotaNotUserBalanceMessage {
		t.Fatalf("expected NewAPI credential message, got %q", usage.UpstreamBalance.Message)
	}
}

func TestAccountUsageService_GetUsage_ParsesNewAPIUserSelfQuotaBalance(t *testing.T) {
	t.Parallel()

	var gotAuth string
	var gotUser string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/user/self" {
			http.NotFound(w, r)
			return
		}
		gotAuth = r.Header.Get("Authorization")
		gotUser = r.Header.Get("New-Api-User")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data": map[string]any{
				"id":         123,
				"quota":      750000,
				"used_quota": 250000,
			},
		})
	}))
	defer upstream.Close()

	repo := &stubOpenAIAccountRepo{
		accounts: []Account{
			{
				ID:       991,
				Platform: PlatformOpenAI,
				Type:     AccountTypeAPIKey,
				Credentials: map[string]any{
					"base_url": upstream.URL,
					"api_key":  "sk-newapi-test",
				},
				Extra: map[string]any{
					"upstream_balance_enabled":      true,
					"upstream_balance_provider":     "newapi",
					"upstream_balance_access_token": "user-access-token",
					"upstream_balance_user_id":      "123",
				},
			},
		},
	}
	svc := &AccountUsageService{accountRepo: repo}

	usage, err := svc.GetUsage(context.Background(), 991)
	if err != nil {
		t.Fatalf("GetUsage() error = %v", err)
	}
	if gotAuth != "Bearer user-access-token" {
		t.Fatalf("expected New API access token auth, got %q", gotAuth)
	}
	if gotUser != "123" {
		t.Fatalf("expected New-Api-User header, got %q", gotUser)
	}
	if usage.UpstreamBalance == nil {
		t.Fatal("expected upstream balance result")
	}
	if !usage.UpstreamBalance.Success {
		t.Fatalf("expected upstream balance success, got %#v", usage.UpstreamBalance)
	}
	if usage.UpstreamBalance.Path != "/api/user/self" {
		t.Fatalf("expected /api/user/self probe to win, got %q", usage.UpstreamBalance.Path)
	}
	if usage.UpstreamBalance.Balance == nil || *usage.UpstreamBalance.Balance != 1.5 {
		t.Fatalf("expected parsed user quota balance $1.50, got %#v", usage.UpstreamBalance.Balance)
	}
	if usage.UpstreamBalance.Currency != "USD" {
		t.Fatalf("expected USD currency, got %q", usage.UpstreamBalance.Currency)
	}
}

func TestAccountUsageService_GetUsage_ParsesNewAPIUserSelfQuotaBalanceWithSessionCookie(t *testing.T) {
	t.Parallel()

	var gotAuth string
	var gotCookie string
	var gotUser string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/user/self" {
			http.NotFound(w, r)
			return
		}
		gotAuth = r.Header.Get("Authorization")
		gotCookie = r.Header.Get("Cookie")
		gotUser = r.Header.Get("New-Api-User")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data": map[string]any{
				"id":         11,
				"quota":      162930803,
				"used_quota": 602069197,
			},
		})
	}))
	defer upstream.Close()

	repo := &stubOpenAIAccountRepo{
		accounts: []Account{
			{
				ID:       994,
				Platform: PlatformOpenAI,
				Type:     AccountTypeAPIKey,
				Credentials: map[string]any{
					"base_url": upstream.URL,
					"api_key":  "sk-newapi-test",
				},
				Extra: map[string]any{
					"upstream_balance_enabled":      true,
					"upstream_balance_provider":     "newapi",
					"upstream_balance_access_token": "cookie:session=session-value",
					"upstream_balance_user_id":      "11",
				},
			},
		},
	}
	svc := &AccountUsageService{accountRepo: repo}

	usage, err := svc.GetUsage(context.Background(), 994)
	if err != nil {
		t.Fatalf("GetUsage() error = %v", err)
	}
	if gotAuth != "" {
		t.Fatalf("expected no bearer auth for session cookie, got %q", gotAuth)
	}
	if gotCookie != "session=session-value" {
		t.Fatalf("expected session cookie auth, got %q", gotCookie)
	}
	if gotUser != "11" {
		t.Fatalf("expected New-Api-User header, got %q", gotUser)
	}
	if usage.UpstreamBalance == nil {
		t.Fatal("expected upstream balance result")
	}
	if !usage.UpstreamBalance.Success {
		t.Fatalf("expected upstream balance success, got %#v", usage.UpstreamBalance)
	}
	if usage.UpstreamBalance.Balance == nil || *usage.UpstreamBalance.Balance != 325.861606 {
		t.Fatalf("expected parsed user quota balance, got %#v", usage.UpstreamBalance.Balance)
	}
}

func TestAccountUsageService_GetUsage_DoesNotTreatSub2APIKeyQuotaAsWalletBalance(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/usage" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"isValid": true,
			"mode":    "quota_limited",
			"quota": map[string]any{
				"limit":     10,
				"remaining": 7,
				"unit":      "USD",
				"used":      3,
			},
			"remaining": 7,
			"unit":      "USD",
		})
	}))
	defer upstream.Close()

	repo := &stubOpenAIAccountRepo{
		accounts: []Account{
			{
				ID:       992,
				Platform: PlatformOpenAI,
				Type:     AccountTypeAPIKey,
				Credentials: map[string]any{
					"base_url": upstream.URL,
					"api_key":  "sk-sub2api-test",
				},
				Extra: map[string]any{
					"upstream_balance_enabled": true,
				},
			},
		},
	}
	svc := &AccountUsageService{accountRepo: repo}

	usage, err := svc.GetUsage(context.Background(), 992)
	if err != nil {
		t.Fatalf("GetUsage() error = %v", err)
	}
	if usage.UpstreamBalance == nil {
		t.Fatal("expected upstream balance result")
	}
	if usage.UpstreamBalance.Success {
		t.Fatalf("key quota must not be reported as wallet balance: %#v", usage.UpstreamBalance)
	}
	if usage.UpstreamBalance.Balance != nil {
		t.Fatalf("expected no wallet balance from quota_limited key quota, got %#v", usage.UpstreamBalance.Balance)
	}
}

func TestAccountUsageService_GetUsage_ParsesSub2APIUserProfileBalance(t *testing.T) {
	t.Parallel()

	var gotAuth string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/user/profile" {
			http.NotFound(w, r)
			return
		}
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code":    0,
			"message": "success",
			"data": map[string]any{
				"id":      77,
				"balance": 12.5,
			},
		})
	}))
	defer upstream.Close()

	repo := &stubOpenAIAccountRepo{
		accounts: []Account{
			{
				ID:       993,
				Platform: PlatformOpenAI,
				Type:     AccountTypeAPIKey,
				Credentials: map[string]any{
					"base_url": upstream.URL,
					"api_key":  "sk-sub2api-test",
				},
				Extra: map[string]any{
					"upstream_balance_enabled":      true,
					"upstream_balance_provider":     "sub2api",
					"upstream_balance_access_token": "sub2api-access-token",
					"upstream_balance_user_id":      "77",
				},
			},
		},
	}
	svc := &AccountUsageService{accountRepo: repo}

	usage, err := svc.GetUsage(context.Background(), 993)
	if err != nil {
		t.Fatalf("GetUsage() error = %v", err)
	}
	if gotAuth != "Bearer sub2api-access-token" {
		t.Fatalf("expected Sub2API access token auth, got %q", gotAuth)
	}
	if usage.UpstreamBalance == nil {
		t.Fatal("expected upstream balance result")
	}
	if !usage.UpstreamBalance.Success {
		t.Fatalf("expected upstream balance success, got %#v", usage.UpstreamBalance)
	}
	if usage.UpstreamBalance.Path != "/api/v1/user/profile" {
		t.Fatalf("expected /api/v1/user/profile probe to win, got %q", usage.UpstreamBalance.Path)
	}
	if usage.UpstreamBalance.Balance == nil || *usage.UpstreamBalance.Balance != 12.5 {
		t.Fatalf("expected parsed user balance 12.5, got %#v", usage.UpstreamBalance.Balance)
	}
}

func TestAccountUsageService_FetchUpstreamBalanceLoginCredentials_NewAPI(t *testing.T) {
	t.Parallel()

	var gotUsername string
	var gotPassword string
	var gotSelfAuth string
	var gotSelfUser string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/user/login":
		case "/api/user/self":
			gotSelfAuth = r.Header.Get("Authorization")
			gotSelfUser = r.Header.Get("New-Api-User")
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"data": map[string]any{
					"id":    42,
					"quota": 500000,
				},
			})
			return
		default:
			http.NotFound(w, r)
			return
		}
		var payload map[string]string
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode login payload: %v", err)
		}
		gotUsername = payload["username"]
		gotPassword = payload["password"]
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data": map[string]any{
				"id":           42,
				"access_token": "newapi-access-token",
			},
		})
	}))
	defer upstream.Close()

	svc := &AccountUsageService{}
	result, err := svc.FetchUpstreamBalanceLoginCredentials(context.Background(), UpstreamBalanceLoginInput{
		BaseURL:  upstream.URL,
		Provider: "newapi",
		Username: "alice@example.com",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("FetchUpstreamBalanceLoginCredentials() error = %v", err)
	}
	if gotUsername != "alice@example.com" || gotPassword != "secret" {
		t.Fatalf("unexpected login payload username=%q password=%q", gotUsername, gotPassword)
	}
	if result.Provider != "newapi" {
		t.Fatalf("expected newapi provider, got %q", result.Provider)
	}
	if result.AccessToken != "newapi-access-token" {
		t.Fatalf("expected access token, got %q", result.AccessToken)
	}
	if result.UserID != "42" {
		t.Fatalf("expected user id 42, got %q", result.UserID)
	}
	if gotSelfAuth != "Bearer newapi-access-token" {
		t.Fatalf("expected NewAPI self bearer auth, got %q", gotSelfAuth)
	}
	if gotSelfUser != "42" {
		t.Fatalf("expected New-Api-User 42, got %q", gotSelfUser)
	}
	if result.Balance == nil || *result.Balance != 1 {
		t.Fatalf("expected verified balance 1, got %#v", result.Balance)
	}
}

func TestAccountUsageService_FetchUpstreamBalanceLoginCredentials_NewAPISessionCookie(t *testing.T) {
	t.Parallel()

	var gotUsername string
	var gotPassword string
	var gotSelfCookie string
	var gotSelfUser string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/user/login":
		case "/api/user/self":
			gotSelfCookie = r.Header.Get("Cookie")
			gotSelfUser = r.Header.Get("New-Api-User")
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"data": map[string]any{
					"id":         11,
					"quota":      162930803,
					"used_quota": 602069197,
				},
			})
			return
		default:
			http.NotFound(w, r)
			return
		}
		var payload map[string]string
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode login payload: %v", err)
		}
		gotUsername = payload["username"]
		gotPassword = payload["password"]
		http.SetCookie(w, &http.Cookie{
			Name:     "session",
			Value:    "session-value",
			Path:     "/",
			HttpOnly: true,
		})
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"message": "",
			"data": map[string]any{
				"id":       11,
				"username": "go1c",
			},
		})
	}))
	defer upstream.Close()

	svc := &AccountUsageService{}
	result, err := svc.FetchUpstreamBalanceLoginCredentials(context.Background(), UpstreamBalanceLoginInput{
		BaseURL:  upstream.URL,
		Provider: "newapi",
		Username: "go1c",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("FetchUpstreamBalanceLoginCredentials() error = %v", err)
	}
	if gotUsername != "go1c" || gotPassword != "secret" {
		t.Fatalf("unexpected login payload username=%q password=%q", gotUsername, gotPassword)
	}
	if result.Provider != "newapi" {
		t.Fatalf("expected newapi provider, got %q", result.Provider)
	}
	if result.AccessToken != "cookie:session=session-value" {
		t.Fatalf("expected session cookie credential, got %q", result.AccessToken)
	}
	if result.UserID != "11" {
		t.Fatalf("expected user id 11, got %q", result.UserID)
	}
	if gotSelfCookie != "session=session-value" {
		t.Fatalf("expected session cookie balance probe, got %q", gotSelfCookie)
	}
	if gotSelfUser != "11" {
		t.Fatalf("expected New-Api-User balance probe, got %q", gotSelfUser)
	}
	if result.Balance == nil || *result.Balance != 325.861606 {
		t.Fatalf("expected verified user balance, got %#v", result.Balance)
	}
}

func TestAccountUsageService_FetchUpstreamBalanceLoginCredentials_Sub2API(t *testing.T) {
	t.Parallel()

	var gotEmail string
	var gotPassword string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/login" {
			http.NotFound(w, r)
			return
		}
		var payload map[string]string
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode login payload: %v", err)
		}
		gotEmail = payload["email"]
		gotPassword = payload["password"]
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code":    0,
			"message": "success",
			"data": map[string]any{
				"access_token": "sub2api-access-token",
				"token_type":   "Bearer",
				"user": map[string]any{
					"id":      77,
					"balance": 12.5,
				},
			},
		})
	}))
	defer upstream.Close()

	svc := &AccountUsageService{}
	result, err := svc.FetchUpstreamBalanceLoginCredentials(context.Background(), UpstreamBalanceLoginInput{
		BaseURL:  upstream.URL,
		Provider: "sub2api",
		Username: "bob@example.com",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("FetchUpstreamBalanceLoginCredentials() error = %v", err)
	}
	if gotEmail != "bob@example.com" || gotPassword != "secret" {
		t.Fatalf("unexpected login payload email=%q password=%q", gotEmail, gotPassword)
	}
	if result.Provider != "sub2api" {
		t.Fatalf("expected sub2api provider, got %q", result.Provider)
	}
	if result.AccessToken != "sub2api-access-token" {
		t.Fatalf("expected access token, got %q", result.AccessToken)
	}
	if result.UserID != "77" {
		t.Fatalf("expected user id 77, got %q", result.UserID)
	}
	if result.Balance == nil || *result.Balance != 12.5 {
		t.Fatalf("expected balance 12.5, got %#v", result.Balance)
	}
}
