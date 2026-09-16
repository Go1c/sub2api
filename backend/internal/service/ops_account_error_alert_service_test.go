package service

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestBuildOpsAccountErrorAlertMessage_SimplifiedAccountTable(t *testing.T) {
	start := time.Date(2026, 7, 2, 15, 32, 0, 0, time.UTC)
	end := start.Add(10 * time.Minute)
	items := []*OpsAccountErrorAlertCandidate{
		{
			AccountID:    12,
			AccountName:  "CC max-0.85",
			StatusCode:   502,
			ErrorCount:   78,
			LatestAt:     end.Add(-10 * time.Second),
			ErrorMessage: "Upstream request failed",
		},
		{
			AccountID:    18,
			AccountName:  "Claude main-01",
			StatusCode:   529,
			ErrorCount:   19,
			LatestAt:     end.Add(-27 * time.Second),
			ErrorMessage: "Overloaded",
		},
	}
	topUsers := []*OpsAccountErrorAlertTopUser{
		{
			UserEmail:  "heavy@example.com",
			ErrorCount: 31,
		},
		{
			UserEmail:  "ops-user@example.com",
			ErrorCount: 12,
		},
		{
			UserEmail:  "",
			ErrorCount: 9,
		},
	}

	msg := buildOpsAccountErrorAlertMessage(start, end, 5, 60, items, topUsers)

	required := []string{
		"[账号异常] 最近 10 分钟有 2 个账号异常",
		"触发条件：单账号异常 >= 5 次",
		"账号",
		"错误",
		"次数",
		"最近时间",
		"CC max-0.85",
		"502",
		"78",
		"Claude main-01",
		"529",
		"主要错误信息：",
		"CC max-0.85：Upstream request failed",
		"影响用户邮箱 Top 2：",
		"heavy@example.com",
		"31",
		"ops-user@example.com",
		"12",
		"降噪：同账号同错误 60 分钟内不重复推送。",
	}
	for _, want := range required {
		if !strings.Contains(msg, want) {
			t.Fatalf("message missing %q:\n%s", want, msg)
		}
	}
	for _, forbidden := range []string{"平台", "分组", "建议", "P0", "P1", "P2"} {
		if strings.Contains(msg, forbidden) {
			t.Fatalf("message should not contain %q:\n%s", forbidden, msg)
		}
	}
	if strings.Contains(msg, "User #") {
		t.Fatalf("message should only use user emails, got:\n%s", msg)
	}
}

func TestUpdateOpsAccountErrorAlertConfig_ValidatesTelegramAndBounds(t *testing.T) {
	repo := newRuntimeSettingRepoStub()
	svc := &OpsService{settingRepo: repo}

	_, err := svc.UpdateOpsAccountErrorAlertConfig(context.Background(), &OpsAccountErrorAlertConfig{
		Enabled:             true,
		IntervalMinutes:     10,
		WindowMinutes:       10,
		MinErrorCount:       5,
		CooldownMinutes:     60,
		MaxAccountsPerAlert: 10,
		MaxUsersPerAlert:    3,
		TelegramBotToken:    "",
		TelegramChatID:      "-100123456",
	})
	if err == nil || !strings.Contains(err.Error(), "telegram_bot_token") {
		t.Fatalf("expected telegram_bot_token validation error, got %v", err)
	}

	updated, err := svc.UpdateOpsAccountErrorAlertConfig(context.Background(), &OpsAccountErrorAlertConfig{
		Enabled:             true,
		IntervalMinutes:     10,
		WindowMinutes:       10,
		MinErrorCount:       5,
		CooldownMinutes:     60,
		MaxAccountsPerAlert: 10,
		MaxUsersPerAlert:    5,
		TelegramBotToken:    " 123456:abcdef ",
		TelegramChatID:      " -100123456 ",
	})
	if err != nil {
		t.Fatalf("UpdateOpsAccountErrorAlertConfig() error = %v", err)
	}
	if updated.TelegramBotToken != "123456:abcdef" {
		t.Fatalf("TelegramBotToken = %q", updated.TelegramBotToken)
	}
	if updated.TelegramChatID != "-100123456" {
		t.Fatalf("TelegramChatID = %q", updated.TelegramChatID)
	}
	if updated.MaxUsersPerAlert != 5 {
		t.Fatalf("MaxUsersPerAlert = %d, want 5", updated.MaxUsersPerAlert)
	}
	if _, ok := repo.values[SettingKeyOpsAccountErrorAlertConfig]; !ok {
		t.Fatalf("expected config persisted under %s", SettingKeyOpsAccountErrorAlertConfig)
	}
}

func TestCollectAccountErrorAlertItems_PrefersRuleHits(t *testing.T) {
	windowEnd := time.Date(2026, 9, 16, 5, 0, 0, 0, time.UTC)
	windowStart := windowEnd.Add(-10 * time.Minute)
	ruleItem := &OpsAccountErrorAlertCandidate{
		AccountID:    7,
		AccountName:  "CPA-Pro20-817",
		StatusCode:   503,
		ErrorCount:   2,
		ErrorMessage: "Our servers are currently overloaded",
	}
	defaultOther := &OpsAccountErrorAlertCandidate{
		AccountID:    8,
		AccountName:  "other",
		StatusCode:   500,
		ErrorCount:   6,
		ErrorMessage: "boom",
	}
	repo := &opsRepoMock{
		ListAccountErrorAlertCandidatesFn: func(_ context.Context, filter *OpsAccountErrorAlertCandidateFilter) ([]*OpsAccountErrorAlertCandidate, error) {
			if filter.AccountID == 7 && filter.Keyword == "overloaded" {
				return []*OpsAccountErrorAlertCandidate{ruleItem}, nil
			}
			if filter.UseAccountKeywords {
				return []*OpsAccountErrorAlertCandidate{ruleItem, defaultOther}, nil
			}
			return nil, nil
		},
		ListAccountIDsWithErrorAlertRulesFn: func(context.Context) ([]int64, error) {
			return []int64{7}, nil
		},
		GetAccountErrorAlertSettingsFn: func(_ context.Context, _ []int64) (map[int64]AccountErrorAlertSettings, error) {
			return map[int64]AccountErrorAlertSettings{
				7: {
					Enabled: true,
					Rules: []AccountErrorAlertRule{{
						Keyword:       "overloaded",
						WindowMinutes: 10,
						MinErrorCount: 1,
						MaxSends:      1,
					}},
				},
			}, nil
		},
	}
	svc := &OpsAccountErrorAlertService{opsRepo: repo}
	got, err := svc.collectAccountErrorAlertItems(context.Background(), &OpsAccountErrorAlertConfig{
		WindowMinutes:       10,
		MinErrorCount:       5,
		MaxAccountsPerAlert: 10,
	}, windowStart, windowEnd)
	if err != nil {
		t.Fatalf("collectAccountErrorAlertItems() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}
	if got[0].AccountID != 7 || got[1].AccountID != 8 {
		t.Fatalf("unexpected order: %+v", got)
	}
}

func TestFilterAccountErrorAlertItems_RespectsMaxSends(t *testing.T) {
	item := &OpsAccountErrorAlertCandidate{
		AccountID:    7,
		AccountName:  "CPA-Pro20-817",
		StatusCode:   503,
		ErrorCount:   3,
		ErrorMessage: "Our servers are currently overloaded",
	}
	repo := &opsRepoMock{
		GetAccountErrorAlertSettingsFn: func(_ context.Context, _ []int64) (map[int64]AccountErrorAlertSettings, error) {
			return map[int64]AccountErrorAlertSettings{
				7: {
					Enabled: true,
					Rules: []AccountErrorAlertRule{{
						Keyword:       "overloaded",
						WindowMinutes: 10,
						MinErrorCount: 1,
						MaxSends:      1,
					}},
				},
			}, nil
		},
	}
	svc := &OpsAccountErrorAlertService{
		opsRepo:    repo,
		sendCounts: map[string]sendWindowCounter{},
	}
	cfg := &OpsAccountErrorAlertConfig{WindowMinutes: 10, CooldownMinutes: 60}
	got, marks := svc.filterAccountErrorAlertItems(context.Background(), cfg, []*OpsAccountErrorAlertCandidate{item})
	if len(got) != 1 {
		t.Fatalf("first filter len = %d, want 1", len(got))
	}
	svc.markSends(context.Background(), marks)
	got, _ = svc.filterAccountErrorAlertItems(context.Background(), cfg, []*OpsAccountErrorAlertCandidate{item})
	if len(got) != 0 {
		t.Fatal("expected max_sends to drop the second alert")
	}
}
