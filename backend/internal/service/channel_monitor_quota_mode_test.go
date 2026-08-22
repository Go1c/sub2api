//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/stretchr/testify/require"
)

type quotaModeRepoStub struct {
	ChannelMonitorRepository
	monitor   *ChannelMonitor
	history   []*ChannelMonitorHistoryRow
	markedIDs []int64
	updated   []*ChannelMonitor
}

func (r *quotaModeRepoStub) GetByID(_ context.Context, id int64) (*ChannelMonitor, error) {
	if r.monitor == nil || r.monitor.ID != id {
		return nil, ErrChannelMonitorNotFound
	}
	clone := *r.monitor
	return &clone, nil
}

func (r *quotaModeRepoStub) InsertHistoryBatch(_ context.Context, rows []*ChannelMonitorHistoryRow) error {
	r.history = append(r.history, rows...)
	return nil
}

func (r *quotaModeRepoStub) MarkChecked(_ context.Context, id int64, _ time.Time) error {
	r.markedIDs = append(r.markedIDs, id)
	return nil
}

func (r *quotaModeRepoStub) Update(_ context.Context, m *ChannelMonitor) error {
	clone := *m
	r.updated = append(r.updated, &clone)
	return nil
}

func newQuotaModeService(repo *quotaModeRepoStub) *ChannelMonitorService {
	return NewChannelMonitorService(repo, &duplicateChannelMonitorEncryptor{})
}

func newQuotaModeFetcher(accounts map[int64]*Account, usage *stubMonitorUsageSource) *ChannelMonitorQuotaFetcher {
	if accounts == nil {
		accounts = make(map[int64]*Account)
	}
	if usage == nil {
		usage = &stubMonitorUsageSource{}
	}
	return &ChannelMonitorQuotaFetcher{
		usage:    usage,
		accounts: &stubMonitorAccountSource{accounts: accounts},
		cache:    make(map[int64]monitorQuotaCacheEntry),
	}
}

func quotaInt64Ptr(v int64) *int64 { return &v }
func quotaStrPtr(v string) *string { return &v }

func TestRunCheck_QuotaModeProducesSingleQuotaResult(t *testing.T) {
	repo := &quotaModeRepoStub{monitor: &ChannelMonitor{
		ID:              1,
		Name:            "claude-quota",
		Provider:        MonitorProviderAnthropic,
		APIMode:         MonitorAPIModeChatCompletions,
		PrimaryModel:    "quota",
		Enabled:         true,
		IntervalSeconds: 60,
		CheckMode:       MonitorCheckModeQuota,
		AccountID:       quotaInt64Ptr(9),
	}}
	svc := newQuotaModeService(repo)
	usage := &stubMonitorUsageSource{usage: &UsageInfo{
		FiveHour: &UsageProgress{Utilization: 30},
	}}
	svc.SetQuotaFetcher(newQuotaModeFetcher(map[int64]*Account{
		9: {ID: 9, Platform: domain.PlatformAnthropic, Type: AccountTypeOAuth},
	}, usage))

	results, err := svc.RunCheck(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, results, 1)

	res := results[0]
	require.Equal(t, "quota", res.Model)
	require.Equal(t, MonitorStatusOperational, res.Status)
	require.Nil(t, res.LatencyMs)
	require.Nil(t, res.PingLatencyMs)
	require.NotNil(t, res.Quota)
	require.True(t, res.Quota.Success)
	require.Equal(t, "usage", res.Quota.Source)

	require.Len(t, repo.history, 1)
	require.Equal(t, "quota", repo.history[0].Model)
	require.NotNil(t, repo.history[0].Quota)
	require.Equal(t, []int64{1}, repo.markedIDs)
}

func TestRunCheck_QuotaModeUnlinkedAccountDegrades(t *testing.T) {
	repo := &quotaModeRepoStub{monitor: &ChannelMonitor{
		ID:              2,
		Provider:        MonitorProviderOpenAI,
		APIMode:         MonitorAPIModeChatCompletions,
		Endpoint:        "",
		PrimaryModel:    "quota",
		Enabled:         true,
		IntervalSeconds: 60,
		CheckMode:       MonitorCheckModeQuota,
		AccountID:       nil,
	}}
	svc := newQuotaModeService(repo)
	svc.SetQuotaFetcher(newQuotaModeFetcher(nil, nil))

	results, err := svc.RunCheck(context.Background(), 2)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, MonitorStatusDegraded, results[0].Status)
	require.Contains(t, results[0].Message, "linked account not found")
	require.False(t, results[0].Quota.Success)
}

func TestRunCheck_QuotaModeNilFetcherFailsClosed(t *testing.T) {
	repo := &quotaModeRepoStub{monitor: &ChannelMonitor{
		ID:              3,
		Provider:        MonitorProviderGemini,
		APIMode:         MonitorAPIModeChatCompletions,
		PrimaryModel:    "quota",
		Enabled:         true,
		IntervalSeconds: 60,
		CheckMode:       MonitorCheckModeQuota,
		AccountID:       quotaInt64Ptr(5),
	}}
	svc := newQuotaModeService(repo)

	results, err := svc.RunCheck(context.Background(), 3)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, MonitorStatusError, results[0].Status)
	require.Contains(t, results[0].Message, "not configured")
}

func TestRunCheck_QuotaProbeAttachesSnapshotToPrimaryRowOnly(t *testing.T) {
	h := &openAICaptureHandler{}
	endpoint := setupFakeOpenAI(t, h)
	repo := &quotaModeRepoStub{monitor: &ChannelMonitor{
		ID:              4,
		Provider:        MonitorProviderOpenAI,
		APIMode:         MonitorAPIModeChatCompletions,
		Endpoint:        endpoint,
		APIKey:          "OLD:sk-openai",
		PrimaryModel:    "gpt-test",
		ExtraModels:     []string{"gpt-extra"},
		Enabled:         true,
		IntervalSeconds: 60,
		CheckMode:       MonitorCheckModeQuotaProbe,
		AccountID:       quotaInt64Ptr(12),
	}}
	svc := newQuotaModeService(repo)
	usage := &stubMonitorUsageSource{usage: &UsageInfo{
		FiveHour: &UsageProgress{Utilization: 20},
	}}
	svc.SetQuotaFetcher(newQuotaModeFetcher(map[int64]*Account{
		12: {ID: 12, Platform: domain.PlatformOpenAI, Type: AccountTypeOAuth},
	}, usage))

	results, err := svc.RunCheck(context.Background(), 4)
	require.NoError(t, err)
	require.Len(t, results, 2)

	require.Equal(t, MonitorStatusOperational, results[0].Status)
	require.NotNil(t, results[0].Quota)
	require.True(t, results[0].Quota.Success)
	require.Equal(t, "usage", results[0].Quota.Source)
	require.Nil(t, results[1].Quota, "extra model rows must not carry quota")

	require.Len(t, repo.history, 2)
	require.NotNil(t, repo.history[0].Quota)
	require.Equal(t, "gpt-test", repo.history[0].Model)
	require.Nil(t, repo.history[1].Quota)
}

func TestRunCheck_QuotaProbeQuotaFailureKeepsProbeStatus(t *testing.T) {
	h := &openAICaptureHandler{}
	endpoint := setupFakeOpenAI(t, h)
	repo := &quotaModeRepoStub{monitor: &ChannelMonitor{
		ID:              5,
		Provider:        MonitorProviderOpenAI,
		APIMode:         MonitorAPIModeChatCompletions,
		Endpoint:        endpoint,
		APIKey:          "OLD:sk-openai",
		PrimaryModel:    "gpt-test",
		Enabled:         true,
		IntervalSeconds: 60,
		CheckMode:       MonitorCheckModeQuotaProbe,
		AccountID:       nil,
	}}
	svc := newQuotaModeService(repo)
	svc.SetQuotaFetcher(newQuotaModeFetcher(nil, nil))

	results, err := svc.RunCheck(context.Background(), 5)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, MonitorStatusOperational, results[0].Status, "quota failure must not flip probe status")
	require.False(t, results[0].Quota.Success)
}

func TestAttachQuotaSnapshot_NoteOnlyWhenProbeMessageEmpty(t *testing.T) {
	results := []*CheckResult{
		{Model: "primary", Status: MonitorStatusOperational, Message: "challenge passed"},
		{Model: "extra"},
	}
	failed := &domain.MonitorQuotaSnapshot{Success: false, Error: "boom"}

	attachQuotaSnapshot(results, failed)

	require.Equal(t, "challenge passed", results[0].Message, "existing probe message wins")
	require.Equal(t, failed, results[0].Quota)
	require.Nil(t, results[1].Quota)

	quiet := []*CheckResult{{Model: "primary", Status: MonitorStatusOperational}}
	attachQuotaSnapshot(quiet, failed)
	require.Contains(t, quiet[0].Message, "quota fetch failed: boom")

	attachQuotaSnapshot(nil, failed)
	attachQuotaSnapshot(results, nil)
}

func TestValidateCreateParams_CheckModeMatrix(t *testing.T) {
	accountID := int64(9)

	cases := []struct {
		name    string
		params  ChannelMonitorCreateParams
		wantErr error
	}{
		{
			name: "probe requires endpoint",
			params: ChannelMonitorCreateParams{
				Provider: MonitorProviderOpenAI, CheckMode: MonitorCheckModeProbe,
				APIKey: "sk", IntervalSeconds: 60, PrimaryModel: "gpt-5",
			},
			wantErr: ErrChannelMonitorInvalidEndpoint,
		},
		{
			name: "probe requires api key",
			params: ChannelMonitorCreateParams{
				Provider: MonitorProviderOpenAI, CheckMode: MonitorCheckModeProbe,
				Endpoint: "https://api.openai.com", IntervalSeconds: 60, PrimaryModel: "gpt-5",
			},
			wantErr: ErrChannelMonitorMissingAPIKey,
		},
		{
			name: "quota drops endpoint and api key requirements",
			params: ChannelMonitorCreateParams{
				Provider: MonitorProviderGemini, CheckMode: MonitorCheckModeQuota,
				IntervalSeconds: 60, AccountID: &accountID,
			},
			wantErr: nil,
		},
		{
			name: "quota requires account",
			params: ChannelMonitorCreateParams{
				Provider: MonitorProviderOpenAI, CheckMode: MonitorCheckModeQuota,
				IntervalSeconds: 60, PrimaryModel: "quota",
			},
			wantErr: ErrChannelMonitorAccountRequired,
		},
		{
			name: "quota_probe requires endpoint and api key too",
			params: ChannelMonitorCreateParams{
				Provider: MonitorProviderOpenAI, CheckMode: MonitorCheckModeQuotaProbe,
				IntervalSeconds: 60, AccountID: &accountID, PrimaryModel: "gpt-5",
			},
			wantErr: ErrChannelMonitorInvalidEndpoint,
		},
		{
			name: "unknown mode rejected",
			params: ChannelMonitorCreateParams{
				Provider: MonitorProviderOpenAI, CheckMode: "auto",
				Endpoint: "https://api.openai.com", APIKey: "sk",
				IntervalSeconds: 60, PrimaryModel: "gpt-5",
			},
			wantErr: ErrChannelMonitorInvalidCheckMode,
		},
		{
			name: "quota_probe requires primary model",
			params: ChannelMonitorCreateParams{
				Provider: MonitorProviderOpenAI, CheckMode: MonitorCheckModeQuotaProbe,
				Endpoint: "https://api.openai.com", APIKey: "sk",
				IntervalSeconds: 60, AccountID: &accountID,
			},
			wantErr: ErrChannelMonitorMissingPrimaryModel,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateCreateParams(tc.params)
			if tc.wantErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tc.wantErr)
			}
		})
	}
}

func TestNormalizeMonitorPrimaryModel_QuotaDefault(t *testing.T) {
	require.Equal(t, "quota", normalizeMonitorPrimaryModel(MonitorProviderOpenAI, MonitorCheckModeQuota, ""))
	require.Equal(t, "quota", normalizeMonitorPrimaryModel(MonitorProviderGemini, MonitorCheckModeQuota, "  "))
	require.Equal(t, "", normalizeMonitorPrimaryModel(MonitorProviderOpenAI, MonitorCheckModeQuotaProbe, ""))
	require.Equal(t, "quota", normalizeMonitorPrimaryModel(MonitorProviderGrok, MonitorCheckModeQuota, ""))
	require.Equal(t, MonitorDefaultGrokModel, normalizeMonitorPrimaryModel(MonitorProviderGrok, MonitorCheckModeProbe, ""))
	require.Equal(t, MonitorDefaultGrokModel, normalizeMonitorPrimaryModel(MonitorProviderGrok, MonitorCheckModeQuotaProbe, ""))
	require.Equal(t, "gpt-5", normalizeMonitorPrimaryModel(MonitorProviderOpenAI, MonitorCheckModeQuotaProbe, "gpt-5"))
}

func TestProviderProbeCapabilityMatrix(t *testing.T) {
	for _, p := range []string{
		MonitorProviderOpenAI, MonitorProviderAnthropic, MonitorProviderGemini, MonitorProviderGrok,
	} {
		require.True(t, providerSupportsProbe(p), p)
		require.NoError(t, validateProvider(p), p)
		require.NoError(t, validateCheckMode(p, MonitorCheckModeProbe), p)
		require.NoError(t, validateCheckMode(p, MonitorCheckModeQuota), p)
		require.NoError(t, validateCheckMode(p, MonitorCheckModeQuotaProbe), p)
	}
	require.ErrorIs(t, validateProvider("kimi"), ErrChannelMonitorInvalidProvider)
	require.ErrorIs(t, validateProvider("antigravity"), ErrChannelMonitorInvalidProvider)
}

func TestValidateLinkedAccount_Matrix(t *testing.T) {
	svc := NewChannelMonitorService(nil, nil)
	fetcher := newQuotaModeFetcher(map[int64]*Account{
		1: {ID: 1, Platform: domain.PlatformOpenAI, Type: AccountTypeOAuth},
	}, nil)
	svc.SetQuotaFetcher(fetcher)

	require.NoError(t, svc.validateLinkedAccount(context.Background(), MonitorProviderOpenAI, nil))
	require.NoError(t, svc.validateLinkedAccount(context.Background(), MonitorProviderOpenAI, quotaInt64Ptr(0)))
	require.NoError(t, svc.validateLinkedAccount(context.Background(), MonitorProviderOpenAI, quotaInt64Ptr(1)))
	require.ErrorIs(t, svc.validateLinkedAccount(context.Background(), MonitorProviderAnthropic, quotaInt64Ptr(1)), ErrChannelMonitorProviderIncompatible)
	require.ErrorIs(t, svc.validateLinkedAccount(context.Background(), MonitorProviderOpenAI, quotaInt64Ptr(404)), ErrChannelMonitorAccountRequired)

	noFetcher := NewChannelMonitorService(nil, nil)
	require.ErrorIs(t, noFetcher.validateLinkedAccount(context.Background(), MonitorProviderOpenAI, quotaInt64Ptr(1)), ErrChannelMonitorAccountRequired)
}

func TestRevalidateLinkedAccount_QuotaErrorsProbeUnbinds(t *testing.T) {
	fetcher := newQuotaModeFetcher(nil, nil)
	svc := NewChannelMonitorService(nil, nil)
	svc.SetQuotaFetcher(fetcher)

	quota := &ChannelMonitor{Provider: MonitorProviderOpenAI, CheckMode: MonitorCheckModeQuota, AccountID: quotaInt64Ptr(9)}
	require.ErrorIs(t, svc.revalidateLinkedAccount(context.Background(), quota), ErrChannelMonitorAccountRequired)
	require.NotNil(t, quota.AccountID)

	probe := &ChannelMonitor{Provider: MonitorProviderOpenAI, CheckMode: MonitorCheckModeProbe, AccountID: quotaInt64Ptr(9)}
	require.NoError(t, svc.revalidateLinkedAccount(context.Background(), probe))
	require.Nil(t, probe.AccountID, "probe mode should silently unbind stale account")

	quotaNoAccount := &ChannelMonitor{Provider: MonitorProviderOpenAI, CheckMode: MonitorCheckModeQuota}
	require.ErrorIs(t, svc.revalidateLinkedAccount(context.Background(), quotaNoAccount), ErrChannelMonitorAccountRequired)
}

func TestRevalidateLinkedAccount_PlatformMismatch(t *testing.T) {
	svc := NewChannelMonitorService(nil, nil)
	svc.SetQuotaFetcher(newQuotaModeFetcher(map[int64]*Account{
		2: {ID: 2, Platform: domain.PlatformGemini},
	}, nil))

	quota := &ChannelMonitor{Provider: MonitorProviderOpenAI, CheckMode: MonitorCheckModeQuota, AccountID: quotaInt64Ptr(2)}
	require.ErrorIs(t, svc.revalidateLinkedAccount(context.Background(), quota), ErrChannelMonitorProviderIncompatible)

	probe := &ChannelMonitor{Provider: MonitorProviderOpenAI, CheckMode: MonitorCheckModeProbe, AccountID: quotaInt64Ptr(2)}
	require.NoError(t, svc.revalidateLinkedAccount(context.Background(), probe))
	require.Nil(t, probe.AccountID)
}

func TestMonitorAccountQuotaCapability_Matrix(t *testing.T) {
	cases := []struct {
		name    string
		account *Account
		wantErr error
	}{
		{
			name:    "anthropic api key cannot query usage",
			account: &Account{ID: 8, Platform: domain.PlatformAnthropic, Type: AccountTypeAPIKey},
			wantErr: ErrChannelMonitorAccountNotSupportable,
		},
		{
			name:    "anthropic oauth ok",
			account: &Account{ID: 9, Platform: domain.PlatformAnthropic, Type: AccountTypeOAuth},
		},
		{
			name:    "anthropic setup token ok",
			account: &Account{ID: 10, Platform: domain.PlatformAnthropic, Type: AccountTypeSetupToken},
		},
		{
			name:    "openai api key cannot query usage",
			account: &Account{ID: 11, Platform: domain.PlatformOpenAI, Type: AccountTypeAPIKey},
			wantErr: ErrChannelMonitorAccountNotSupportable,
		},
		{
			name:    "openai oauth ok",
			account: &Account{ID: 12, Platform: domain.PlatformOpenAI, Type: AccountTypeOAuth},
		},
		{
			name:    "gemini api key ok",
			account: &Account{ID: 13, Platform: domain.PlatformGemini, Type: AccountTypeAPIKey},
		},
		{
			name:    "grok ok",
			account: &Account{ID: 14, Platform: domain.PlatformGrok},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := monitorAccountQuotaCapability(tc.account)
			if tc.wantErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tc.wantErr)
			}
		})
	}
}

func TestValidateLinkedAccount_CapabilityRejected(t *testing.T) {
	svc := NewChannelMonitorService(nil, nil)
	svc.SetQuotaFetcher(newQuotaModeFetcher(map[int64]*Account{
		1: {ID: 1, Platform: domain.PlatformOpenAI, Type: AccountTypeAPIKey},
	}, nil))

	err := svc.validateLinkedAccount(context.Background(), MonitorProviderOpenAI, quotaInt64Ptr(1))
	require.ErrorIs(t, err, ErrChannelMonitorAccountNotSupportable)
}

func TestRevalidateLinkedAccount_Capability(t *testing.T) {
	svc := NewChannelMonitorService(nil, nil)
	svc.SetQuotaFetcher(newQuotaModeFetcher(map[int64]*Account{
		2: {ID: 2, Platform: domain.PlatformOpenAI, Type: AccountTypeAPIKey},
	}, nil))

	quota := &ChannelMonitor{Provider: MonitorProviderOpenAI, CheckMode: MonitorCheckModeQuota, AccountID: quotaInt64Ptr(2)}
	require.ErrorIs(t, svc.revalidateLinkedAccount(context.Background(), quota), ErrChannelMonitorAccountNotSupportable)
	require.NotNil(t, quota.AccountID, "quota mode keeps the binding for the admin to fix")

	probe := &ChannelMonitor{Provider: MonitorProviderOpenAI, CheckMode: MonitorCheckModeProbe, AccountID: quotaInt64Ptr(2)}
	require.NoError(t, svc.revalidateLinkedAccount(context.Background(), probe))
	require.Nil(t, probe.AccountID, "probe mode should silently unbind unusable account")
}

func TestApplyMonitorUpdate_ProviderOnlyKeepsLegalCheckMode(t *testing.T) {
	probeOpenAI := func() *ChannelMonitor {
		return &ChannelMonitor{
			Provider: MonitorProviderOpenAI, APIMode: MonitorAPIModeChatCompletions,
			Endpoint: "https://api.openai.com", PrimaryModel: "gpt-5",
			CheckMode: MonitorCheckModeProbe,
		}
	}

	provider := MonitorProviderAnthropic
	err := applyMonitorUpdate(probeOpenAI(), ChannelMonitorUpdateParams{Provider: &provider})
	require.NoError(t, err)

	accountID := int64(3)
	err = applyMonitorUpdate(probeOpenAI(), ChannelMonitorUpdateParams{
		Provider: &provider, CheckMode: quotaStrPtr(MonitorCheckModeQuota), AccountID: &accountID,
	})
	require.NoError(t, err)

	legacy := &ChannelMonitor{
		Provider: MonitorProviderOpenAI, APIMode: MonitorAPIModeChatCompletions,
		Endpoint: "https://api.openai.com", PrimaryModel: "gpt-5",
		CheckMode: "bogus",
	}
	newName := "renamed"
	require.NoError(t, applyMonitorUpdate(legacy, ChannelMonitorUpdateParams{Name: &newName}))
}

func TestValidateProbeAPIKey_QuotaToProbeRequiresFreshKey(t *testing.T) {
	svc := NewChannelMonitorService(nil, &duplicateChannelMonitorEncryptor{})

	quota := &ChannelMonitor{Provider: MonitorProviderOpenAI, CheckMode: MonitorCheckModeQuota, APIKey: "NEW:"}
	require.NoError(t, svc.validateProbeAPIKey(quota, ""))

	quota.CheckMode = MonitorCheckModeProbe
	require.ErrorIs(t, svc.validateProbeAPIKey(quota, ""), ErrChannelMonitorMissingAPIKey)
	require.NoError(t, svc.validateProbeAPIKey(quota, "sk-fresh"))
	require.NoError(t, svc.validateProbeAPIKey(
		&ChannelMonitor{Provider: MonitorProviderOpenAI, CheckMode: MonitorCheckModeProbe, APIKey: "OLD:sk-live"}, ""))
}

func TestDuplicateChannelMonitorQuotaModeReencryptsEmptyKey(t *testing.T) {
	accountID := int64(9)
	source := &ChannelMonitor{
		ID:              42,
		Name:            "claude-quota",
		Provider:        MonitorProviderAnthropic,
		APIMode:         MonitorAPIModeChatCompletions,
		Endpoint:        "",
		APIKey:          "OLD:",
		PrimaryModel:    "quota",
		Enabled:         true,
		IntervalSeconds: 60,
		CheckMode:       MonitorCheckModeQuota,
		AccountID:       &accountID,
	}
	repo := &duplicateChannelMonitorRepoStub{source: source}
	svc := NewChannelMonitorService(repo, &duplicateChannelMonitorEncryptor{})

	dup, err := svc.Duplicate(context.Background(), 42, 7, "admin:7", "op-1")
	require.NoError(t, err)
	require.Equal(t, MonitorCheckModeQuota, dup.CheckMode)
	require.NotNil(t, dup.AccountID)
	require.Equal(t, accountID, *dup.AccountID)
	require.Empty(t, dup.APIKey, "plaintext stays empty for quota monitors")
	require.Len(t, repo.created, 1)
	require.Equal(t, "NEW:", repo.created[0].APIKey, "empty key must be re-encrypted, not dropped")
}
