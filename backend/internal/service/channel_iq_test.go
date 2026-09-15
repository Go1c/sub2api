//go:build unit

package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type memoryChannelIQStore struct {
	mu       sync.Mutex
	settings *ChannelIQSettings
	results  map[int64]*ChannelIQResult
}

func newMemoryChannelIQStore() *memoryChannelIQStore {
	return &memoryChannelIQStore{results: map[int64]*ChannelIQResult{}}
}

func (s *memoryChannelIQStore) GetSettings(context.Context) (*ChannelIQSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.settings == nil {
		return &ChannelIQSettings{}, nil
	}
	copySettings := *s.settings
	copySettings.GroupIDs = append([]int64(nil), s.settings.GroupIDs...)
	copySettings.ExcludedAccountIDs = append([]int64(nil), s.settings.ExcludedAccountIDs...)
	return &copySettings, nil
}

func (s *memoryChannelIQStore) SaveSettings(_ context.Context, settings *ChannelIQSettings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	copySettings := *settings
	copySettings.GroupIDs = append([]int64(nil), settings.GroupIDs...)
	copySettings.ExcludedAccountIDs = append([]int64(nil), settings.ExcludedAccountIDs...)
	s.settings = &copySettings
	return nil
}

func (s *memoryChannelIQStore) GetResults(_ context.Context, accountIDs []int64) (map[int64]*ChannelIQResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[int64]*ChannelIQResult{}
	for _, id := range accountIDs {
		if result, ok := s.results[id]; ok {
			copyResult := *result
			out[id] = &copyResult
		}
	}
	return out, nil
}

func (s *memoryChannelIQStore) MarkRunning(_ context.Context, accountID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	current := s.results[accountID]
	if current == nil {
		current = &ChannelIQResult{AccountID: accountID}
	}
	current.Status = ChannelIQStatusRunning
	current.Error = ""
	s.results[accountID] = current
	return nil
}

func (s *memoryChannelIQStore) SaveResult(_ context.Context, result *ChannelIQResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	current := s.results[result.AccountID]
	count := 1
	if current != nil {
		count = current.TestCount + 1
	}
	copyResult := *result
	copyResult.TestCount = count
	s.results[result.AccountID] = &copyResult
	return nil
}

func (s *memoryChannelIQStore) DeleteResult(_ context.Context, accountID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.results, accountID)
	return nil
}

type stubChannelIQAccounts struct {
	byGroup map[int64][]Account
}

func (s stubChannelIQAccounts) ListByGroup(_ context.Context, groupID int64) ([]Account, error) {
	return s.byGroup[groupID], nil
}

type stubChannelIQTester struct {
	mu      sync.Mutex
	calls   []int64
	results map[int64]*IQTestRunResult
	block   <-chan struct{}
}

func (s *stubChannelIQTester) RunIQTestBackground(_ context.Context, accountID int64, _, _ string) (*IQTestRunResult, error) {
	if s.block != nil {
		<-s.block
	}
	s.mu.Lock()
	s.calls = append(s.calls, accountID)
	result := s.results[accountID]
	s.mu.Unlock()
	if result == nil {
		return &IQTestRunResult{Success: true, SVG: `<svg xmlns="http://www.w3.org/2000/svg"></svg>`, Duration: 10 * time.Millisecond, Tokens: 100}, nil
	}
	return result, nil
}

func TestExtractCompleteSVG(t *testing.T) {
	got := extractCompleteSVG("here is the drawing\n<svg xmlns=\"http://www.w3.org/2000/svg\"><circle/></svg>\n")
	require.Contains(t, got, "<svg")
	require.Contains(t, got, "</svg>")
	require.Empty(t, extractCompleteSVG("no drawing"))
}

func TestChannelIQService_ListFiltersOpenAIAccountsInSelectedGroups(t *testing.T) {
	store := newMemoryChannelIQStore()
	svc := NewChannelIQService(store, stubChannelIQAccounts{byGroup: map[int64][]Account{
		1: {
			{ID: 2, Name: "codex-b", Platform: PlatformOpenAI},
			{ID: 3, Name: "claude", Platform: "anthropic"},
			{ID: 1, Name: "codex-a", Platform: PlatformOpenAI},
		},
	}}, &stubChannelIQTester{})
	_, err := svc.SaveSettings(context.Background(), ChannelIQSettings{GroupIDs: []int64{1}})
	require.NoError(t, err)

	out, err := svc.GetOverview(context.Background())
	require.NoError(t, err)
	require.Len(t, out.Items, 2)
	require.Equal(t, "codex-a", out.Items[0].Name)
	require.Equal(t, "codex-b", out.Items[1].Name)
	require.Equal(t, ChannelIQStatusIdle, out.Items[0].Status)
}

func TestChannelIQService_RunAllRecordsSuccess(t *testing.T) {
	store := newMemoryChannelIQStore()
	tester := &stubChannelIQTester{results: map[int64]*IQTestRunResult{
		11: {Success: true, SVG: `<svg xmlns="http://www.w3.org/2000/svg"></svg>`, Tokens: 4971, Duration: 150 * time.Millisecond},
	}}
	svc := NewChannelIQService(store, stubChannelIQAccounts{byGroup: map[int64][]Account{
		9: {{ID: 11, Name: "astra", Platform: PlatformOpenAI}},
	}}, tester)
	_, err := svc.SaveSettings(context.Background(), ChannelIQSettings{GroupIDs: []int64{9}})
	require.NoError(t, err)
	require.NoError(t, svc.RunAll(context.Background()))

	require.Eventually(t, func() bool {
		out, err := svc.GetOverview(context.Background())
		return err == nil && len(out.Items) == 1 && out.Items[0].Status == ChannelIQStatusSuccess && out.Items[0].TestCount == 1
	}, 2*time.Second, 20*time.Millisecond)

	out, err := svc.GetOverview(context.Background())
	require.NoError(t, err)
	require.Equal(t, int64(4971), out.Items[0].TotalTokens)
	require.Contains(t, out.Items[0].SVG, "<svg")
}

func TestChannelIQService_SaveSettingsNormalizesInterval(t *testing.T) {
	store := newMemoryChannelIQStore()
	svc := NewChannelIQService(store, stubChannelIQAccounts{}, &stubChannelIQTester{})
	saved, err := svc.SaveSettings(context.Background(), ChannelIQSettings{IntervalSeconds: 10, Prompt: "  ", Model: ""})
	require.NoError(t, err)
	require.Equal(t, channelIQDefaultInterval, saved.IntervalSeconds)
	require.Equal(t, channelIQDefaultPrompt, saved.Prompt)
	require.Equal(t, channelIQDefaultModel, saved.Model)
}

func TestChannelIQService_RunOneRejectsAccountOutsideList(t *testing.T) {
	store := newMemoryChannelIQStore()
	svc := NewChannelIQService(store, stubChannelIQAccounts{byGroup: map[int64][]Account{
		1: {{ID: 1, Name: "a", Platform: PlatformOpenAI}},
	}}, &stubChannelIQTester{})
	_, err := svc.SaveSettings(context.Background(), ChannelIQSettings{GroupIDs: []int64{1}})
	require.NoError(t, err)
	require.Error(t, svc.RunOne(context.Background(), 99))
}

func TestChannelIQService_RunAllBusyWhileRunning(t *testing.T) {
	store := newMemoryChannelIQStore()
	block := make(chan struct{})
	tester := &stubChannelIQTester{
		block: block,
		results: map[int64]*IQTestRunResult{
			1: {Success: true, SVG: `<svg xmlns="http://www.w3.org/2000/svg"></svg>`},
		},
	}
	svc := NewChannelIQService(store, stubChannelIQAccounts{byGroup: map[int64][]Account{
		1: {{ID: 1, Name: "a", Platform: PlatformOpenAI}},
	}}, tester)
	_, err := svc.SaveSettings(context.Background(), ChannelIQSettings{GroupIDs: []int64{1}})
	require.NoError(t, err)
	require.NoError(t, svc.RunAll(context.Background()))
	require.ErrorIs(t, svc.RunAll(context.Background()), ErrChannelIQBusy)
	close(block)
	require.Eventually(t, func() bool {
		out, err := svc.GetOverview(context.Background())
		return err == nil && len(out.Items) == 1 && out.Items[0].Status == ChannelIQStatusSuccess
	}, 2*time.Second, 20*time.Millisecond)
}

func TestChannelIQService_AutoTickHonorsEnabledAndInterval(t *testing.T) {
	store := newMemoryChannelIQStore()
	tester := &stubChannelIQTester{}
	svc := NewChannelIQService(store, stubChannelIQAccounts{byGroup: map[int64][]Account{
		1: {{ID: 1, Name: "a", Platform: PlatformOpenAI}},
	}}, tester)
	_, err := svc.SaveSettings(context.Background(), ChannelIQSettings{GroupIDs: []int64{1}, AutoEnabled: false, IntervalSeconds: 1800})
	require.NoError(t, err)
	require.NoError(t, svc.AutoTick(context.Background()))
	require.Empty(t, tester.calls)

	_, err = svc.SaveSettings(context.Background(), ChannelIQSettings{GroupIDs: []int64{1}, AutoEnabled: true, IntervalSeconds: 1800})
	require.NoError(t, err)
	svc.lastAutoAt = time.Now()
	require.NoError(t, svc.AutoTick(context.Background()))
	require.Empty(t, tester.calls)

	svc.lastAutoAt = time.Now().Add(-time.Hour)
	require.NoError(t, svc.AutoTick(context.Background()))
	require.Eventually(t, func() bool {
		tester.mu.Lock()
		defer tester.mu.Unlock()
		return len(tester.calls) == 1
	}, 2*time.Second, 20*time.Millisecond)
}

func TestChannelIQService_ExcludeAccountRemovesFromListAndDeletesSVG(t *testing.T) {
	store := newMemoryChannelIQStore()
	store.results[11] = &ChannelIQResult{
		AccountID: 11,
		Status:    ChannelIQStatusSuccess,
		SVG:       `<svg xmlns="http://www.w3.org/2000/svg"><circle/></svg>`,
		TestCount: 2,
	}
	svc := NewChannelIQService(store, stubChannelIQAccounts{byGroup: map[int64][]Account{
		9: {
			{ID: 11, Name: "keep-out", Platform: PlatformOpenAI},
			{ID: 12, Name: "keep-in", Platform: PlatformOpenAI},
		},
	}}, &stubChannelIQTester{})
	_, err := svc.SaveSettings(context.Background(), ChannelIQSettings{GroupIDs: []int64{9}})
	require.NoError(t, err)

	require.NoError(t, svc.ExcludeAccount(context.Background(), 11))

	out, err := svc.GetOverview(context.Background())
	require.NoError(t, err)
	require.Len(t, out.Items, 1)
	require.Equal(t, int64(12), out.Items[0].AccountID)
	require.Len(t, out.Excluded, 1)
	require.Equal(t, int64(11), out.Excluded[0].AccountID)
	require.Equal(t, "keep-out", out.Excluded[0].Name)
	_, stillStored := store.results[11]
	require.False(t, stillStored)
	require.Error(t, svc.RunOne(context.Background(), 11))
}

func TestChannelIQService_RestoreAccountReturnsToIdleList(t *testing.T) {
	store := newMemoryChannelIQStore()
	svc := NewChannelIQService(store, stubChannelIQAccounts{byGroup: map[int64][]Account{
		1: {{ID: 5, Name: "astra", Platform: PlatformOpenAI}},
	}}, &stubChannelIQTester{})
	_, err := svc.SaveSettings(context.Background(), ChannelIQSettings{GroupIDs: []int64{1}})
	require.NoError(t, err)
	require.NoError(t, svc.ExcludeAccount(context.Background(), 5))
	require.NoError(t, svc.RestoreAccount(context.Background(), 5))

	out, err := svc.GetOverview(context.Background())
	require.NoError(t, err)
	require.Len(t, out.Items, 1)
	require.Equal(t, int64(5), out.Items[0].AccountID)
	require.Equal(t, ChannelIQStatusIdle, out.Items[0].Status)
	require.Empty(t, out.Items[0].SVG)
	require.Empty(t, out.Excluded)
}

func TestChannelIQService_RunAllSkipsExcludedAccounts(t *testing.T) {
	store := newMemoryChannelIQStore()
	tester := &stubChannelIQTester{}
	svc := NewChannelIQService(store, stubChannelIQAccounts{byGroup: map[int64][]Account{
		1: {
			{ID: 1, Name: "a", Platform: PlatformOpenAI},
			{ID: 2, Name: "b", Platform: PlatformOpenAI},
		},
	}}, tester)
	_, err := svc.SaveSettings(context.Background(), ChannelIQSettings{GroupIDs: []int64{1}})
	require.NoError(t, err)
	require.NoError(t, svc.ExcludeAccount(context.Background(), 2))
	require.NoError(t, svc.RunAll(context.Background()))

	require.Eventually(t, func() bool {
		tester.mu.Lock()
		defer tester.mu.Unlock()
		return len(tester.calls) == 1 && tester.calls[0] == 1
	}, 2*time.Second, 20*time.Millisecond)
}
