package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type poolInspectAccountStub struct {
	mu       sync.Mutex
	accounts []Account
	bound    map[int64][]int64
	paused   map[int64]time.Time
	errors   map[int64]string
	extras   map[int64]map[string]any
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

func (s *poolInspectAccountStub) SetError(_ context.Context, id int64, errorMsg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.errors == nil {
		s.errors = map[int64]string{}
	}
	s.errors[id] = errorMsg
	for i := range s.accounts {
		if s.accounts[i].ID == id {
			s.accounts[i].Status = StatusError
			s.accounts[i].Schedulable = false
			s.accounts[i].ErrorMessage = errorMsg
		}
	}
	return nil
}

func (s *poolInspectAccountStub) UpdateExtra(_ context.Context, id int64, updates map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.extras == nil {
		s.extras = map[int64]map[string]any{}
	}
	if s.extras[id] == nil {
		s.extras[id] = map[string]any{}
	}
	for key, value := range updates {
		s.extras[id][key] = value
	}
	for i := range s.accounts {
		if s.accounts[i].ID != id {
			continue
		}
		if s.accounts[i].Extra == nil {
			s.accounts[i].Extra = map[string]any{}
		}
		for key, value := range updates {
			s.accounts[i].Extra[key] = value
		}
	}
	return nil
}

func (s *poolInspectAccountStub) SetTempUnschedulable(_ context.Context, id int64, until time.Time, _ string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.paused == nil {
		s.paused = map[int64]time.Time{}
	}
	s.paused[id] = until
	for i := range s.accounts {
		if s.accounts[i].ID == id {
			copied := until
			s.accounts[i].TempUnschedulableUntil = &copied
		}
	}
	return nil
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

type poolInspectGroupStub struct {
	groups map[int64]Group
}

func (s *poolInspectGroupStub) GetByIDLite(_ context.Context, id int64) (*Group, error) {
	if s == nil || s.groups == nil {
		return nil, errors.New("missing")
	}
	group, ok := s.groups[id]
	if !ok {
		return nil, errors.New("missing")
	}
	copied := group
	return &copied, nil
}

type poolInspectQuizStub struct {
	text string
	err  error
	calls int
}

func (s *poolInspectQuizStub) Ask(context.Context, *Account, string, string) (string, error) {
	s.calls++
	return s.text, s.err
}

type poolInspectLockStub struct {
	mu   sync.Mutex
	keys map[string]time.Time
}

func (s *poolInspectLockStub) Acquire(context.Context, string, string, time.Duration) (bool, error) {
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

func iqInspectConfig(correct, incorrect int64) *AccountPoolAutoInspectConfig {
	cfg := defaultAccountPoolAutoInspectConfig()
	cfg.Enabled = true
	cfg.CorrectGroupID = correct
	cfg.IncorrectGroupID = incorrect
	cfg.PauseMinutes = 1
	return cfg
}

func TestAccountPoolAutoInspectMovesCorrectAnswerAndPauses(t *testing.T) {
	accounts := &poolInspectAccountStub{accounts: []Account{{
		ID: 12, Name: "codex-1", Platform: PlatformOpenAI, Type: AccountTypeOAuth,
		Status: StatusActive, Schedulable: true, GroupIDs: []int64{1, 9},
	}}}
	quiz := &poolInspectQuizStub{text: "答案是 21。"}
	svc := NewAccountPoolAutoInspectService(
		&poolInspectSettingsStub{},
		accounts,
		&poolInspectGroupStub{groups: map[int64]Group{
			3: {ID: 3, Platform: PlatformOpenAI},
			9: {ID: 9, Platform: PlatformOpenAI},
		}},
		quiz,
		&poolInspectLockStub{},
	)
	_, err := svc.UpdateConfig(context.Background(), iqInspectConfig(3, 9))
	require.NoError(t, err)

	status := svc.RunOnce(context.Background(), true)
	require.Empty(t, status.LastError)
	require.Contains(t, status.LastResult, "correct=1")
	require.Contains(t, status.LastResult, "moved=1")
	require.Equal(t, []int64{1, 3}, accounts.bound[12])
	require.WithinDuration(t, time.Now().Add(time.Minute), accounts.paused[12], 5*time.Second)
	require.Equal(t, 1, quiz.calls)
}

func TestAccountPoolAutoInspectMovesIncorrectAnswer(t *testing.T) {
	accounts := &poolInspectAccountStub{accounts: []Account{{
		ID: 13, Name: "codex-2", Platform: PlatformOpenAI, Type: AccountTypeOAuth,
		Status: StatusActive, Schedulable: true, GroupIDs: []int64{3},
	}}}
	svc := NewAccountPoolAutoInspectService(
		&poolInspectSettingsStub{},
		accounts,
		&poolInspectGroupStub{groups: map[int64]Group{
			3: {ID: 3, Platform: PlatformOpenAI},
			9: {ID: 9, Platform: PlatformOpenAI},
		}},
		&poolInspectQuizStub{text: "不知道"},
		nil,
	)
	_, err := svc.UpdateConfig(context.Background(), iqInspectConfig(3, 9))
	require.NoError(t, err)

	status := svc.RunOnce(context.Background(), true)
	require.Contains(t, status.LastResult, "incorrect=1")
	require.Equal(t, []int64{9}, accounts.bound[13])
	require.NotZero(t, accounts.paused[13])
}

func TestAccountPoolAutoInspectUntestableDoesNotMove(t *testing.T) {
	accounts := &poolInspectAccountStub{accounts: []Account{{
		ID: 14, Name: "codex-3", Platform: PlatformOpenAI, Type: AccountTypeOAuth,
		Status: StatusActive, Schedulable: true, GroupIDs: []int64{3},
	}}}
	svc := NewAccountPoolAutoInspectService(
		&poolInspectSettingsStub{},
		accounts,
		nil,
		&poolInspectQuizStub{err: errors.New("timeout")},
		nil,
	)
	_, err := svc.UpdateConfig(context.Background(), iqInspectConfig(3, 9))
	require.NoError(t, err)

	status := svc.RunOnce(context.Background(), true)
	require.Contains(t, status.LastResult, "untestable=1")
	require.Contains(t, status.LastResult, "moved=0")
	require.Contains(t, status.LastResult, "disabled=0")
	require.Empty(t, accounts.bound)
	require.Empty(t, accounts.paused)
	require.Empty(t, accounts.errors)
}

func TestAccountPoolAutoInspectDisablesFirstImportOnIncorrect(t *testing.T) {
	accounts := &poolInspectAccountStub{accounts: []Account{{
		ID: 15, Name: "fresh", Platform: PlatformOpenAI, Type: AccountTypeOAuth,
		Status: StatusActive, Schedulable: true, GroupIDs: []int64{3},
		Extra: map[string]any{"imported_at": "2026-09-29T00:00:00Z"},
	}}}
	cfg := iqInspectConfig(3, 9)
	cfg.DisableFirstImportOnIncorrect = true
	svc := NewAccountPoolAutoInspectService(
		&poolInspectSettingsStub{},
		accounts,
		&poolInspectGroupStub{groups: map[int64]Group{
			3: {ID: 3, Platform: PlatformOpenAI},
			9: {ID: 9, Platform: PlatformOpenAI},
		}},
		&poolInspectQuizStub{text: "不知道"},
		nil,
	)
	_, err := svc.UpdateConfig(context.Background(), cfg)
	require.NoError(t, err)

	status := svc.RunOnce(context.Background(), true)
	require.Contains(t, status.LastResult, "incorrect=1")
	require.Contains(t, status.LastResult, "disabled=1")
	require.Contains(t, status.LastResult, "moved=0")
	require.Equal(t, accountPoolIQFirstImportError, accounts.errors[15])
	require.Equal(t, StatusError, accounts.accounts[0].Status)
	require.False(t, accounts.accounts[0].Schedulable)
	require.Empty(t, accounts.bound)
	require.NotEmpty(t, accounts.accounts[0].GetExtraString(accountPoolAutoInspectCheckedAtExtra))

	status = svc.RunOnce(context.Background(), true)
	require.Contains(t, status.LastResult, "asked=0")
}

func TestAccountPoolAutoInspectKeepsCheckedImportOnIncorrect(t *testing.T) {
	accounts := &poolInspectAccountStub{accounts: []Account{{
		ID: 16, Name: "checked", Platform: PlatformOpenAI, Type: AccountTypeOAuth,
		Status: StatusActive, Schedulable: true, GroupIDs: []int64{3},
		Extra: map[string]any{
			"imported_at":                        "2026-09-29T00:00:00Z",
			accountPoolAutoInspectCheckedAtExtra: "2026-09-29T01:00:00Z",
		},
	}}}
	cfg := iqInspectConfig(3, 9)
	cfg.DisableFirstImportOnIncorrect = true
	svc := NewAccountPoolAutoInspectService(
		&poolInspectSettingsStub{},
		accounts,
		&poolInspectGroupStub{groups: map[int64]Group{
			3: {ID: 3, Platform: PlatformOpenAI},
			9: {ID: 9, Platform: PlatformOpenAI},
		}},
		&poolInspectQuizStub{text: "不知道"},
		nil,
	)
	_, err := svc.UpdateConfig(context.Background(), cfg)
	require.NoError(t, err)

	status := svc.RunOnce(context.Background(), true)
	require.Contains(t, status.LastResult, "disabled=0")
	require.Contains(t, status.LastResult, "moved=1")
	require.Equal(t, []int64{9}, accounts.bound[16])
	require.Empty(t, accounts.errors)
}

func TestAccountPoolAutoInspectAlreadyOnTargetDoesNotPause(t *testing.T) {
	accounts := &poolInspectAccountStub{accounts: []Account{{
		ID: 15, Name: "codex-4", Platform: PlatformOpenAI, Type: AccountTypeOAuth,
		Status: StatusActive, Schedulable: true, GroupIDs: []int64{1, 3},
	}}}
	svc := NewAccountPoolAutoInspectService(
		&poolInspectSettingsStub{},
		accounts,
		nil,
		&poolInspectQuizStub{text: "21"},
		nil,
	)
	_, err := svc.UpdateConfig(context.Background(), iqInspectConfig(3, 9))
	require.NoError(t, err)

	status := svc.RunOnce(context.Background(), true)
	require.Contains(t, status.LastResult, "correct=1")
	require.Contains(t, status.LastResult, "moved=0")
	require.Empty(t, accounts.paused)
}

func TestAccountPoolAutoInspectSkipsWhilePaused(t *testing.T) {
	until := time.Now().Add(time.Minute)
	quiz := &poolInspectQuizStub{text: "21"}
	accounts := &poolInspectAccountStub{accounts: []Account{{
		ID: 16, Name: "cooling", Platform: PlatformOpenAI, Type: AccountTypeOAuth,
		Status: StatusActive, Schedulable: true, GroupIDs: []int64{3},
		TempUnschedulableUntil: &until,
	}}}
	svc := NewAccountPoolAutoInspectService(&poolInspectSettingsStub{}, accounts, nil, quiz, nil)
	_, err := svc.UpdateConfig(context.Background(), iqInspectConfig(3, 9))
	require.NoError(t, err)

	status := svc.RunOnce(context.Background(), true)
	require.Contains(t, status.LastResult, "asked=0")
	require.Zero(t, quiz.calls)
}

func TestAccountPoolAutoInspectTimerRespectsDue(t *testing.T) {
	accounts := &poolInspectAccountStub{accounts: []Account{{
		ID: 17, Name: "codex-5", Platform: PlatformOpenAI, Type: AccountTypeOAuth,
		Status: StatusActive, Schedulable: true, GroupIDs: []int64{1},
	}}}
	quiz := &poolInspectQuizStub{text: "21"}
	svc := NewAccountPoolAutoInspectService(&poolInspectSettingsStub{}, accounts, nil, quiz, &poolInspectLockStub{})
	_, err := svc.UpdateConfig(context.Background(), iqInspectConfig(3, 9))
	require.NoError(t, err)

	first := svc.RunOnce(context.Background(), false)
	require.Contains(t, first.LastResult, "asked=1")
	second := svc.RunOnce(context.Background(), false)
	require.Contains(t, second.LastResult, "asked=0")
	require.Equal(t, 1, quiz.calls)
}
