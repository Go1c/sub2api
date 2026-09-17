package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEvaluateAccountPoolAutoInspectHealth_AllIPsFail(t *testing.T) {
	dto := AccountRequestHealthDTO{
		AccountID: 1,
		Mode:      RequestHealthModeIPGroup,
		Lines: []AccountRequestHealthLineDTO{
			{IP: "1.1.*.*", Outcomes: failOutcomes(8)},
			{IP: "2.2.*.*", Outcomes: failOutcomes(4)},
		},
	}
	got := evaluateAccountPoolAutoInspectHealth(dto, 50, 4)
	require.True(t, got.Unhealthy)
	require.Equal(t, 12, got.Samples)
	require.Equal(t, 0.0, got.Rate)
}

func TestEvaluateAccountPoolAutoInspectHealth_OneHealthyIPSkips(t *testing.T) {
	dto := AccountRequestHealthDTO{
		AccountID: 1,
		Mode:      RequestHealthModeIPGroup,
		Lines: []AccountRequestHealthLineDTO{
			{IP: "1.1.*.*", Outcomes: failOutcomes(8)},
			{IP: "2.2.*.*", Outcomes: okOutcomes(8)},
		},
	}
	got := evaluateAccountPoolAutoInspectHealth(dto, 50, 4)
	require.False(t, got.Unhealthy)
	require.Equal(t, 0.5, got.Rate)
}

func TestEvaluateAccountPoolAutoInspectHealth_NotEnoughSamples(t *testing.T) {
	dto := AccountRequestHealthDTO{
		AccountID: 1,
		Lines: []AccountRequestHealthLineDTO{
			{IP: "1.1.*.*", Outcomes: failOutcomes(2)},
		},
	}
	got := evaluateAccountPoolAutoInspectHealth(dto, 50, 4)
	require.False(t, got.Unhealthy)
	require.Equal(t, 2, got.Samples)
}

func TestEvaluateAccountPoolAutoInspectHealth_EmptyLinesIgnored(t *testing.T) {
	dto := AccountRequestHealthDTO{
		AccountID: 1,
		Mode:      RequestHealthModeIPGroup,
		Lines: []AccountRequestHealthLineDTO{
			{IP: "idle", Outcomes: nil},
			{IP: "hot", Outcomes: failOutcomes(6)},
		},
	}
	got := evaluateAccountPoolAutoInspectHealth(dto, 50, 4)
	require.True(t, got.Unhealthy)
}

func TestPlanAccountPoolAutoInspectRemediation_AddsGroupAndRemovesModel(t *testing.T) {
	account := Account{
		ID:       9,
		GroupIDs: []int64{1},
		Credentials: map[string]any{
			"model_mapping": map[string]any{
				"gpt-5.4":     "gpt-5.4",
				"gpt-6-astra": "gpt-6-astra",
			},
		},
	}
	plan := planAccountPoolAutoInspectRemediation(account, &AccountPoolAutoInspectConfig{
		AddGroupIDs:  []int64{1, 7},
		RemoveModels: []string{"gpt-6-astra"},
	})
	require.True(t, plan.HasWork)
	require.True(t, plan.GroupsChanged)
	require.Equal(t, []int64{1, 7}, plan.GroupIDs)
	require.Equal(t, []int64{7}, plan.AddedGroupIDs)
	require.True(t, plan.MappingChanged)
	require.Equal(t, []string{"gpt-6-astra"}, plan.RemovedModels)
	_, stillThere := plan.Mapping["gpt-6-astra"]
	require.False(t, stillThere)
	require.Equal(t, "gpt-5.4", plan.Mapping["gpt-5.4"])
}

func TestPlanAccountPoolAutoInspectRemediation_IdempotentWhenAlreadyApplied(t *testing.T) {
	account := Account{
		ID:       9,
		GroupIDs: []int64{1, 7},
		Credentials: map[string]any{
			"model_mapping": map[string]any{"gpt-5.4": "gpt-5.4"},
		},
	}
	plan := planAccountPoolAutoInspectRemediation(account, &AccountPoolAutoInspectConfig{
		AddGroupIDs:  []int64{7},
		RemoveModels: []string{"gpt-6-astra"},
	})
	require.False(t, plan.HasWork)
}

func TestCompatibleAddGroupIDs_SkipsOtherPlatforms(t *testing.T) {
	account := Account{Platform: PlatformOpenAI}
	groups := map[int64]Group{
		1: {ID: 1, Platform: PlatformOpenAI},
		2: {ID: 2, Platform: PlatformAnthropic},
		3: {ID: 3, Platform: PlatformComposite},
	}
	got := compatibleAddGroupIDs(account, []int64{1, 2, 3}, groups, true)
	require.Equal(t, []int64{1, 3}, got)
	require.True(t, groupAcceptsAccountPlatform(PlatformComposite, PlatformAnthropic))
	require.False(t, groupAcceptsAccountPlatform(PlatformOpenAI, PlatformAnthropic))
}

func TestPlanAccountPoolAutoInspectRemediation_SkipsEmptyMapping(t *testing.T) {
	account := Account{ID: 3, GroupIDs: []int64{1}, Credentials: map[string]any{"access_token": "x"}}
	plan := planAccountPoolAutoInspectRemediation(account, &AccountPoolAutoInspectConfig{
		RemoveModels: []string{"gpt-6-astra"},
	})
	require.False(t, plan.MappingChanged)
}

func TestRemoveModelsFromMapping_DropsValueMatches(t *testing.T) {
	next, removed := removeModelsFromMapping(map[string]any{
		"alias": "gpt-6-astra",
		"keep":  "gpt-5.4",
	}, []string{"gpt-6-astra"})
	require.Equal(t, []string{"alias"}, removed)
	require.Equal(t, map[string]any{"keep": "gpt-5.4"}, next)
}

func TestAccountStoppedByOAuth401(t *testing.T) {
	now := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	until := now.Add(10 * time.Minute)

	stopped, reason := accountStoppedByOAuth401(Account{
		TempUnschedulableUntil:  &until,
		TempUnschedulableReason: "OAuth 401: invalid_grant",
	}, now)
	require.True(t, stopped)
	require.Contains(t, reason, "401")

	stopped, _ = accountStoppedByOAuth401(Account{
		Status:       StatusError,
		ErrorMessage: "Authentication failed (401): expired",
	}, now)
	require.True(t, stopped)

	stopped, _ = accountStoppedByOAuth401(Account{
		Status:       StatusActive,
		ErrorMessage: "rate limited 429",
	}, now)
	require.False(t, stopped)
}

func TestNormalizeAccountPoolAutoInspectConfig(t *testing.T) {
	cfg := &AccountPoolAutoInspectConfig{
		IntervalMinutes:         0,
		SuccessRateThreshold:    0,
		MinSamples:              0,
		AddGroupIDs:             []int64{0, 7, 7, -1},
		RemoveModels:            []string{" gpt-6-astra ", "gpt-6-astra", ""},
		OAuth401CooldownMinutes: 0,
		TelegramBotToken:        "  tok  ",
	}
	normalizeAccountPoolAutoInspectConfig(cfg)
	require.Equal(t, 5, cfg.IntervalMinutes)
	require.Equal(t, 50, cfg.SuccessRateThreshold)
	require.Equal(t, 4, cfg.MinSamples)
	require.Equal(t, []int64{7}, cfg.AddGroupIDs)
	require.Equal(t, []string{"gpt-6-astra"}, cfg.RemoveModels)
	require.Equal(t, 60, cfg.OAuth401CooldownMinutes)
	require.Equal(t, "tok", cfg.TelegramBotToken)
}

func TestResolveAccountPoolAutoInspectTelegramFallsBackToOps(t *testing.T) {
	token, chat := resolveAccountPoolAutoInspectTelegram(
		&AccountPoolAutoInspectConfig{},
		&OpsAccountErrorAlertConfig{TelegramBotToken: "ops-token", TelegramChatID: "-100"},
	)
	require.Equal(t, "ops-token", token)
	require.Equal(t, "-100", chat)

	token, chat = resolveAccountPoolAutoInspectTelegram(
		&AccountPoolAutoInspectConfig{TelegramBotToken: "own", TelegramChatID: "1"},
		&OpsAccountErrorAlertConfig{TelegramBotToken: "ops-token", TelegramChatID: "-100"},
	)
	require.Equal(t, "own", token)
	require.Equal(t, "1", chat)
}

func failOutcomes(n int) []RequestHealthOutcomeDTO {
	out := make([]RequestHealthOutcomeDTO, n)
	for i := range out {
		out[i] = RequestHealthOutcomeDTO{Slot: RequestHealthSlotFail, StatusCode: 429}
	}
	return out
}

func okOutcomes(n int) []RequestHealthOutcomeDTO {
	out := make([]RequestHealthOutcomeDTO, n)
	for i := range out {
		out[i] = RequestHealthOutcomeDTO{Slot: RequestHealthSlotOK}
	}
	return out
}
