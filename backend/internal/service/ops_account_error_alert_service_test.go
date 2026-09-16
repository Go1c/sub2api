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
