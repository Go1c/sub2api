package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type poolInspectAccountStub struct {
	mu          sync.Mutex
	accounts    []Account
	bound       map[int64][]int64
	credentials map[int64]map[string]any
}

func (s *poolInspectAccountStub) ListAllWithFilters(context.Context, string, string, string, string, int64, string) ([]Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Account, len(s.accounts))
	copy(out, s.accounts)
	return out, nil
}

func (s *poolInspectAccountStub) BindGroups(_ context.Context, accountID int64, groupIDs []int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.bound == nil {
		s.bound = map[int64][]int64{}
	}
	copied := append([]int64(nil), groupIDs...)
	s.bound[accountID] = copied
	for i := range s.accounts {
		if s.accounts[i].ID == accountID {
			s.accounts[i].GroupIDs = copied
		}
	}
	return nil
}

func (s *poolInspectAccountStub) Update(_ context.Context, account *Account) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if account == nil {
		return nil
	}
	if s.credentials == nil {
		s.credentials = map[int64]map[string]any{}
	}
	s.credentials[account.ID] = account.Credentials
	for i := range s.accounts {
		if s.accounts[i].ID == account.ID {
			s.accounts[i] = *account
		}
	}
	return nil
}

func (s *poolInspectAccountStub) UpdateExtra(_ context.Context, accountID int64, updates map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.accounts {
		if s.accounts[i].ID != accountID {
			continue
		}
		extra := map[string]any{}
		for k, v := range s.accounts[i].Extra {
			extra[k] = v
		}
		for k, v := range updates {
			extra[k] = v
		}
		s.accounts[i].Extra = extra
	}
	return nil
}

type poolInspectHealthStub struct {
	items []AccountRequestHealthDTO
}

func (s *poolInspectHealthStub) ListForAccounts(context.Context, []int64, int) ([]AccountRequestHealthDTO, error) {
	return s.items, nil
}

type poolInspectSettingsStub struct {
	raw string
}

func (s *poolInspectSettingsStub) GetValue(context.Context, string) (string, error) {
	if s.raw == "" {
		return "", ErrSettingNotFound
	}
	return s.raw, nil
}

func (s *poolInspectSettingsStub) Set(_ context.Context, _ string, value string) error {
	s.raw = value
	return nil
}

type poolInspectSenderStub struct {
	mu       sync.Mutex
	messages []string
}

func (s *poolInspectSenderStub) SendMessage(_ context.Context, _, _, text string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = append(s.messages, text)
	return nil
}

type poolInspectFallbackStub struct {
	cfg *OpsAccountErrorAlertConfig
}

func (s *poolInspectFallbackStub) GetOpsAccountErrorAlertConfig(context.Context) (*OpsAccountErrorAlertConfig, error) {
	return s.cfg, nil
}

type poolInspectLockStub struct {
	mu     sync.Mutex
	keys   map[string]time.Time
	reject bool
}

func (s *poolInspectLockStub) Acquire(context.Context, string, string, time.Duration) (bool, error) {
	if s != nil && s.reject {
		return false, nil
	}
	return true, nil
}

func (s *poolInspectLockStub) Release(context.Context, string, string) error { return nil }

func (s *poolInspectLockStub) Exists(_ context.Context, key string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	until, ok := s.keys[key]
	return ok && time.Now().Before(until), nil
}

func (s *poolInspectLockStub) SetCooldown(_ context.Context, key string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.keys == nil {
		s.keys = map[string]time.Time{}
	}
	s.keys[key] = time.Now().Add(ttl)
	return nil
}

func (s *poolInspectLockStub) GetInt(context.Context, string) (int64, error) { return 0, nil }

func (s *poolInspectLockStub) Incr(context.Context, string, time.Duration) (int64, error) {
	return 1, nil
}

func TestAccountPoolAutoInspectServiceRunOnceDegradesUnhealthyAccount(t *testing.T) {
	accounts := &poolInspectAccountStub{
		accounts: []Account{{
			ID:          12,
			Name:        "codex-1",
			Status:      StatusActive,
			Schedulable: true,
			GroupIDs:    []int64{1},
			Credentials: map[string]any{
				"access_token": "tok",
				"model_mapping": map[string]any{
					"gpt-5.4":     "gpt-5.4",
					"gpt-6-astra": "gpt-6-astra",
				},
			},
		}},
	}
	settings := &poolInspectSettingsStub{}
	sender := &poolInspectSenderStub{}
	svc := NewAccountPoolAutoInspectService(
		settings,
		accounts,
		&poolInspectHealthStub{items: []AccountRequestHealthDTO{{
			AccountID: 12,
			Mode:      RequestHealthModeIPGroup,
			Lines: []AccountRequestHealthLineDTO{
				{IP: "a", Outcomes: failOutcomes(6)},
				{IP: "b", Outcomes: failOutcomes(6)},
			},
		}}},
		sender,
		&poolInspectFallbackStub{cfg: &OpsAccountErrorAlertConfig{TelegramBotToken: "bot", TelegramChatID: "chat"}},
		&poolInspectLockStub{},
		nil,
	)
	cfg := defaultAccountPoolAutoInspectConfig()
	cfg.Enabled = true
	cfg.AddGroupIDs = []int64{7}
	cfg.RemoveModels = []string{"gpt-6-astra"}
	_, err := svc.UpdateConfig(context.Background(), cfg)
	require.NoError(t, err)

	status := svc.RunOnce(context.Background(), true)
	require.Empty(t, status.LastError)
	require.Contains(t, status.LastResult, "degraded=1")
	require.Equal(t, []int64{1, 7}, accounts.bound[12])
	mapping, _ := accounts.credentials[12]["model_mapping"].(map[string]any)
	_, exists := mapping["gpt-6-astra"]
	require.False(t, exists)
	require.Equal(t, "tok", accounts.credentials[12]["access_token"])
	require.Len(t, sender.messages, 1)
	require.Contains(t, sender.messages[0], "成功率过低")
	require.Contains(t, sender.messages[0], "codex-1")
}

func TestAccountPoolAutoInspectServiceRunOnceNotifiesOAuth401(t *testing.T) {
	until := time.Now().Add(8 * time.Minute)
	accounts := &poolInspectAccountStub{
		accounts: []Account{{
			ID:                      21,
			Name:                    "dead-oauth",
			Status:                  StatusActive,
			Schedulable:             true,
			TempUnschedulableUntil:  &until,
			TempUnschedulableReason: "Authentication failed (401): invalid or expired credentials",
		}},
	}
	sender := &poolInspectSenderStub{}
	svc := NewAccountPoolAutoInspectService(
		&poolInspectSettingsStub{},
		accounts,
		&poolInspectHealthStub{},
		sender,
		&poolInspectFallbackStub{cfg: &OpsAccountErrorAlertConfig{TelegramBotToken: "bot", TelegramChatID: "chat"}},
		&poolInspectLockStub{},
		nil,
	)
	cfg := defaultAccountPoolAutoInspectConfig()
	cfg.Enabled = true
	cfg.NotifyOAuth401 = true
	_, err := svc.UpdateConfig(context.Background(), cfg)
	require.NoError(t, err)

	status := svc.RunOnce(context.Background(), true)
	require.Contains(t, status.LastResult, "oauth401=1")
	require.Len(t, sender.messages, 1)
	require.Contains(t, sender.messages[0], "401 已停止调度")
	require.Contains(t, sender.messages[0], "dead-oauth")

	status = svc.RunOnce(context.Background(), true)
	require.Contains(t, status.LastResult, "oauth401=0")
	require.Len(t, sender.messages, 1)
}

func TestAccountPoolAutoInspectServiceSkipsDisabledUnlessForced(t *testing.T) {
	accounts := &poolInspectAccountStub{accounts: []Account{{ID: 1, Status: StatusActive, Schedulable: true}}}
	svc := NewAccountPoolAutoInspectService(
		&poolInspectSettingsStub{},
		accounts,
		&poolInspectHealthStub{},
		&poolInspectSenderStub{},
		nil,
		&poolInspectLockStub{},
		nil,
	)
	status := svc.RunOnce(context.Background(), false)
	require.Empty(t, status.LastResult)
	require.Nil(t, status.LastRunAt)
}

func TestAccountPoolAutoInspectServiceForceRunIgnoresLeaderLock(t *testing.T) {
	parent := int64(99)
	accounts := &poolInspectAccountStub{
		accounts: []Account{
			{
				ID:          12,
				Name:        "codex-1",
				Platform:    PlatformOpenAI,
				Status:      StatusActive,
				Schedulable: true,
				GroupIDs:    []int64{1},
			},
			{
				ID:              13,
				Name:            "shadow",
				Platform:        PlatformOpenAI,
				Status:          StatusActive,
				Schedulable:     true,
				ParentAccountID: &parent,
				GroupIDs:        []int64{1},
			},
		},
	}
	svc := NewAccountPoolAutoInspectService(
		&poolInspectSettingsStub{},
		accounts,
		&poolInspectHealthStub{items: []AccountRequestHealthDTO{
			{AccountID: 12, Lines: []AccountRequestHealthLineDTO{{IP: "a", Outcomes: failOutcomes(6)}}},
			{AccountID: 13, Lines: []AccountRequestHealthLineDTO{{IP: "a", Outcomes: failOutcomes(6)}}},
		}},
		&poolInspectSenderStub{},
		nil,
		&poolInspectLockStub{reject: true},
		nil,
	)
	svc.groups = &poolInspectGroupStub{groups: map[int64]Group{
		7: {ID: 7, Platform: PlatformOpenAI},
		8: {ID: 8, Platform: PlatformAnthropic},
	}}
	cfg := defaultAccountPoolAutoInspectConfig()
	cfg.Enabled = true
	cfg.AddGroupIDs = []int64{7, 8}
	_, err := svc.UpdateConfig(context.Background(), cfg)
	require.NoError(t, err)

	scheduled := svc.RunOnce(context.Background(), false)
	require.Equal(t, "skipped_lock", scheduled.LastResult)
	require.Empty(t, accounts.bound)

	forced := svc.RunOnce(context.Background(), true)
	require.Contains(t, forced.LastResult, "degraded=1")
	require.Equal(t, []int64{1, 7}, accounts.bound[12])
	require.NotContains(t, accounts.bound, int64(13))
}

func TestAccountPoolAutoInspectServiceRejectsUnknownGroup(t *testing.T) {
	svc := NewAccountPoolAutoInspectService(
		&poolInspectSettingsStub{},
		&poolInspectAccountStub{},
		&poolInspectHealthStub{},
		&poolInspectSenderStub{},
		nil,
		&poolInspectLockStub{},
		nil,
	)
	svc.groups = &poolInspectGroupStub{groups: map[int64]Group{7: {ID: 7, Platform: PlatformOpenAI}}}
	cfg := defaultAccountPoolAutoInspectConfig()
	cfg.AddGroupIDs = []int64{99}
	_, err := svc.UpdateConfig(context.Background(), cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "99")
}

type poolInspectGroupStub struct {
	groups map[int64]Group
}

func (s *poolInspectGroupStub) GetByIDLite(_ context.Context, id int64) (*Group, error) {
	group, ok := s.groups[id]
	if !ok {
		return nil, ErrGroupNotFound
	}
	return &group, nil
}

func TestAccountPoolAutoInspectServiceRunOnceCloses429ExemptionOnDegrade(t *testing.T) {
	accounts := &poolInspectAccountStub{accounts: []Account{
		{ID: 31, Name: "codex-exempt", Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, GroupIDs: []int64{1}},
		{ID: 32, Name: "codex-enforced", Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, GroupIDs: []int64{1}, Extra: map[string]any{OAuth429CooldownEnforcedExtraKey: true}},
		{ID: 33, Name: "gemini-1", Platform: PlatformGemini, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, GroupIDs: []int64{1}},
	}}
	allFail := []AccountRequestHealthDTO{
		{AccountID: 31, Mode: RequestHealthModeIPGroup, Lines: []AccountRequestHealthLineDTO{{IP: "a", Outcomes: failOutcomes(6)}}},
		{AccountID: 32, Mode: RequestHealthModeIPGroup, Lines: []AccountRequestHealthLineDTO{{IP: "a", Outcomes: failOutcomes(6)}}},
		{AccountID: 33, Mode: RequestHealthModeIPGroup, Lines: []AccountRequestHealthLineDTO{{IP: "a", Outcomes: failOutcomes(6)}}},
	}
	sender := &poolInspectSenderStub{}
	svc := NewAccountPoolAutoInspectService(
		&poolInspectSettingsStub{},
		accounts,
		&poolInspectHealthStub{items: allFail},
		sender,
		&poolInspectFallbackStub{cfg: &OpsAccountErrorAlertConfig{TelegramBotToken: "bot", TelegramChatID: "chat"}},
		&poolInspectLockStub{},
		nil,
	)
	cfg := defaultAccountPoolAutoInspectConfig()
	cfg.Enabled = true
	cfg.Close429ExemptionOnDegrade = true
	_, err := svc.UpdateConfig(context.Background(), cfg)
	require.NoError(t, err)

	status := svc.RunOnce(context.Background(), true)
	require.Empty(t, status.LastError)
	require.Contains(t, status.LastResult, "degraded=1", "只有豁免中的 OpenAI 账号有可执行动作")
	require.Equal(t, true, accounts.accounts[0].Extra[OAuth429CooldownEnforcedExtraKey], "降级后关闭豁免")
	require.Equal(t, true, accounts.accounts[1].Extra[OAuth429CooldownEnforcedExtraKey], "已 enforced 的账号不重复处理")
	require.Nil(t, accounts.accounts[2].Extra, "非 OpenAI 账号不写豁免键")
	require.Len(t, sender.messages, 1)
	require.Contains(t, sender.messages[0], "关闭 429 豁免")

	// 恢复健康后不自动回开豁免：与分组/模型降级一致，恢复不撤销动作。
	healthy := []AccountRequestHealthDTO{
		{AccountID: 31, Mode: RequestHealthModeIPGroup, Lines: []AccountRequestHealthLineDTO{{IP: "a", Outcomes: okOutcomes(6)}}},
	}
	svc2 := NewAccountPoolAutoInspectService(
		&poolInspectSettingsStub{},
		accounts,
		&poolInspectHealthStub{items: healthy},
		&poolInspectSenderStub{},
		&poolInspectFallbackStub{},
		&poolInspectLockStub{},
		nil,
	)
	svc2.RunOnce(context.Background(), true)
	require.Equal(t, true, accounts.accounts[0].Extra[OAuth429CooldownEnforcedExtraKey], "巡检恢复不自动重新豁免")
}

func TestAccountPoolAutoInspectServiceDegradeKeeps429ExemptionWhenDisabled(t *testing.T) {
	accounts := &poolInspectAccountStub{accounts: []Account{
		{ID: 41, Name: "codex-1", Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, GroupIDs: []int64{1}},
	}}
	svc := NewAccountPoolAutoInspectService(
		&poolInspectSettingsStub{},
		accounts,
		&poolInspectHealthStub{items: []AccountRequestHealthDTO{
			{AccountID: 41, Mode: RequestHealthModeIPGroup, Lines: []AccountRequestHealthLineDTO{{IP: "a", Outcomes: failOutcomes(6)}}},
		}},
		&poolInspectSenderStub{},
		&poolInspectFallbackStub{},
		&poolInspectLockStub{},
		nil,
	)
	cfg := defaultAccountPoolAutoInspectConfig()
	cfg.Enabled = true
	cfg.AddGroupIDs = []int64{7}
	cfg.Close429ExemptionOnDegrade = false
	_, err := svc.UpdateConfig(context.Background(), cfg)
	require.NoError(t, err)

	status := svc.RunOnce(context.Background(), true)
	require.Empty(t, status.LastError)
	require.Contains(t, status.LastResult, "degraded=1")
	require.Equal(t, []int64{1, 7}, accounts.bound[41])
	require.Nil(t, accounts.accounts[0].Extra, "关闭联动时降级不写豁免键")
}

func TestAccountPoolAutoInspectConfigDefaultCloses429Exemption(t *testing.T) {
	require.True(t, defaultAccountPoolAutoInspectConfig().Close429ExemptionOnDegrade, "联动默认开启")
	parsed := parseAccountPoolAutoInspectConfig(`{"enabled":true,"interval_minutes":5,"success_rate_threshold":50,"min_samples":4,"add_group_ids":[],"remove_models":[]}`)
	require.True(t, parsed.Close429ExemptionOnDegrade, "存量配置缺键时按默认联动处理")
}
