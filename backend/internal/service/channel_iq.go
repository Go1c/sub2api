package service

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	ChannelIQStatusIdle      = "idle"
	ChannelIQStatusRunning   = "running"
	ChannelIQStatusSuccess   = "success"
	ChannelIQStatusFailed    = "failed"
	channelIQConcurrency     = 8
	channelIQMinIntervalSec  = 300
	channelIQMaxIntervalSec  = 86400
	channelIQDefaultInterval = 1800
	channelIQDefaultPrompt   = "Generate an SVG of a pelican riding a bicycle"
	channelIQDefaultModel    = "gpt-6-astra"
)

var ErrChannelIQBusy = errors.New("channel iq detection already running")

type ChannelIQSettings struct {
	GroupIDs           []int64   `json:"group_ids"`
	ExcludedAccountIDs []int64   `json:"excluded_account_ids"`
	AutoEnabled        bool      `json:"auto_enabled"`
	IntervalSeconds    int       `json:"interval_seconds"`
	Prompt             string    `json:"prompt"`
	Model              string    `json:"model"`
	UpdatedAt          time.Time `json:"updated_at,omitempty"`
}

type ChannelIQResult struct {
	AccountID       int64      `json:"account_id"`
	Status          string     `json:"status"`
	TestCount       int        `json:"test_count"`
	Model           string     `json:"model"`
	ReasoningEffort string     `json:"reasoning_effort"`
	DurationMs      int64      `json:"duration_ms"`
	TotalTokens     int64      `json:"total_tokens"`
	SVG             string     `json:"svg"`
	Error           string     `json:"error,omitempty"`
	LastRunAt       *time.Time `json:"last_run_at,omitempty"`
}

type ChannelIQItem struct {
	AccountID       int64      `json:"account_id"`
	Name            string     `json:"name"`
	Status          string     `json:"status"`
	TestCount       int        `json:"test_count"`
	Model           string     `json:"model"`
	ReasoningEffort string     `json:"reasoning_effort"`
	DurationMs      int64      `json:"duration_ms"`
	TotalTokens     int64      `json:"total_tokens"`
	SVG             string     `json:"svg"`
	Error           string     `json:"error,omitempty"`
	LastRunAt       *time.Time `json:"last_run_at,omitempty"`
}

type ChannelIQExcludedAccount struct {
	AccountID int64  `json:"account_id"`
	Name      string `json:"name"`
}

type ChannelIQOverview struct {
	Settings ChannelIQSettings          `json:"settings"`
	Items    []ChannelIQItem            `json:"items"`
	Excluded []ChannelIQExcludedAccount `json:"excluded"`
}

type IQTestRunResult struct {
	Success  bool
	Text     string
	SVG      string
	Tokens   int64
	Duration time.Duration
	Error    string
}

type ChannelIQStore interface {
	GetSettings(ctx context.Context) (*ChannelIQSettings, error)
	SaveSettings(ctx context.Context, settings *ChannelIQSettings) error
	GetResults(ctx context.Context, accountIDs []int64) (map[int64]*ChannelIQResult, error)
	MarkRunning(ctx context.Context, accountID int64) error
	SaveResult(ctx context.Context, result *ChannelIQResult) error
	DeleteResult(ctx context.Context, accountID int64) error
}

type channelIQAccountLister interface {
	ListByGroup(ctx context.Context, groupID int64) ([]Account, error)
}

type channelIQTester interface {
	RunIQTestBackground(ctx context.Context, accountID int64, modelID, prompt string) (*IQTestRunResult, error)
}

type ChannelIQService struct {
	store      ChannelIQStore
	accounts   channelIQAccountLister
	tester     channelIQTester
	batchMu    sync.Mutex
	running    bool
	lastAutoAt time.Time
}

func NewChannelIQService(store ChannelIQStore, accounts channelIQAccountLister, tester channelIQTester) *ChannelIQService {
	return &ChannelIQService{store: store, accounts: accounts, tester: tester}
}

func ProvideChannelIQService(store ChannelIQStore, accountRepo AccountRepository, tester *AccountTestService) *ChannelIQService {
	return NewChannelIQService(store, accountRepo, tester)
}

func (s *ChannelIQService) GetOverview(ctx context.Context) (*ChannelIQOverview, error) {
	settings, err := s.normalizedSettings(ctx)
	if err != nil {
		return nil, err
	}
	accounts, err := s.listWatchAccounts(ctx, settings)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(accounts))
	for _, account := range accounts {
		ids = append(ids, account.ID)
	}
	results, err := s.store.GetResults(ctx, ids)
	if err != nil {
		return nil, err
	}
	items := make([]ChannelIQItem, 0, len(accounts))
	for _, account := range accounts {
		item := ChannelIQItem{
			AccountID:       account.ID,
			Name:            account.Name,
			Status:          ChannelIQStatusIdle,
			ReasoningEffort: accountIQTestReasoningEffort,
			Model:           settings.Model,
		}
		if result := results[account.ID]; result != nil {
			item.Status = result.Status
			item.TestCount = result.TestCount
			item.Model = result.Model
			item.ReasoningEffort = result.ReasoningEffort
			item.DurationMs = result.DurationMs
			item.TotalTokens = result.TotalTokens
			item.SVG = result.SVG
			item.Error = result.Error
			item.LastRunAt = result.LastRunAt
		}
		items = append(items, item)
	}
	excludedAccounts, err := s.listExcludedAccounts(ctx, settings)
	if err != nil {
		return nil, err
	}
	excluded := make([]ChannelIQExcludedAccount, 0, len(excludedAccounts))
	for _, account := range excludedAccounts {
		excluded = append(excluded, ChannelIQExcludedAccount{AccountID: account.ID, Name: account.Name})
	}
	return &ChannelIQOverview{Settings: *settings, Items: items, Excluded: excluded}, nil
}

func (s *ChannelIQService) SaveSettings(ctx context.Context, incoming ChannelIQSettings) (*ChannelIQSettings, error) {
	normalized := normalizeChannelIQSettings(incoming)
	if err := s.store.SaveSettings(ctx, normalized); err != nil {
		return nil, err
	}
	return s.normalizedSettings(ctx)
}

func (s *ChannelIQService) RunAll(ctx context.Context) error {
	settings, err := s.normalizedSettings(ctx)
	if err != nil {
		return err
	}
	accounts, err := s.listWatchAccounts(ctx, settings)
	if err != nil {
		return err
	}
	ids := make([]int64, 0, len(accounts))
	for _, account := range accounts {
		ids = append(ids, account.ID)
	}
	return s.startBatch(ctx, ids, settings)
}

func (s *ChannelIQService) ExcludeAccount(ctx context.Context, accountID int64) error {
	settings, err := s.normalizedSettings(ctx)
	if err != nil {
		return err
	}
	inGroup, err := s.accountInSelectedGroups(ctx, settings, accountID)
	if err != nil {
		return err
	}
	if !inGroup {
		return errors.New("account is not in the channel iq list")
	}
	settings.ExcludedAccountIDs = channelIQAppendID(settings.ExcludedAccountIDs, accountID)
	if err := s.store.SaveSettings(ctx, settings); err != nil {
		return err
	}
	return s.store.DeleteResult(ctx, accountID)
}

func (s *ChannelIQService) RestoreAccount(ctx context.Context, accountID int64) error {
	settings, err := s.normalizedSettings(ctx)
	if err != nil {
		return err
	}
	settings.ExcludedAccountIDs = channelIQRemoveID(settings.ExcludedAccountIDs, accountID)
	return s.store.SaveSettings(ctx, settings)
}

func (s *ChannelIQService) RunOne(ctx context.Context, accountID int64) error {
	settings, err := s.normalizedSettings(ctx)
	if err != nil {
		return err
	}
	accounts, err := s.listWatchAccounts(ctx, settings)
	if err != nil {
		return err
	}
	found := false
	for _, account := range accounts {
		if account.ID == accountID {
			found = true
			break
		}
	}
	if !found {
		return errors.New("account is not in the channel iq list")
	}
	return s.startBatch(ctx, []int64{accountID}, settings)
}

func (s *ChannelIQService) AutoTick(ctx context.Context) error {
	settings, err := s.normalizedSettings(ctx)
	if err != nil {
		return err
	}
	if !settings.AutoEnabled {
		return nil
	}
	interval := time.Duration(settings.IntervalSeconds) * time.Second
	s.batchMu.Lock()
	due := s.lastAutoAt.IsZero() || time.Since(s.lastAutoAt) >= interval
	s.batchMu.Unlock()
	if !due {
		return nil
	}
	if err := s.RunAll(ctx); err != nil {
		if errors.Is(err, ErrChannelIQBusy) {
			return nil
		}
		return err
	}
	s.batchMu.Lock()
	s.lastAutoAt = time.Now()
	s.batchMu.Unlock()
	return nil
}

func (s *ChannelIQService) startBatch(ctx context.Context, accountIDs []int64, settings *ChannelIQSettings) error {
	if len(accountIDs) == 0 {
		return nil
	}
	s.batchMu.Lock()
	if s.running {
		s.batchMu.Unlock()
		return ErrChannelIQBusy
	}
	s.running = true
	s.lastAutoAt = time.Now()
	s.batchMu.Unlock()

	for _, id := range accountIDs {
		_ = s.store.MarkRunning(context.Background(), id)
	}

	go s.runBatch(context.Background(), accountIDs, *settings)
	return nil
}

func (s *ChannelIQService) runBatch(ctx context.Context, accountIDs []int64, settings ChannelIQSettings) {
	defer func() {
		s.batchMu.Lock()
		s.running = false
		s.batchMu.Unlock()
	}()

	sem := make(chan struct{}, channelIQConcurrency)
	var wg sync.WaitGroup
	for _, id := range accountIDs {
		wg.Add(1)
		sem <- struct{}{}
		go func(accountID int64) {
			defer wg.Done()
			defer func() { <-sem }()
			s.runOne(ctx, accountID, settings)
		}(id)
	}
	wg.Wait()
}

func (s *ChannelIQService) runOne(ctx context.Context, accountID int64, settings ChannelIQSettings) {
	started := time.Now()
	run, err := s.tester.RunIQTestBackground(ctx, accountID, settings.Model, settings.Prompt)
	if s.accountExcluded(ctx, accountID) {
		_ = s.store.DeleteResult(ctx, accountID)
		return
	}
	result := &ChannelIQResult{
		AccountID:       accountID,
		Status:          ChannelIQStatusFailed,
		Model:           settings.Model,
		ReasoningEffort: accountIQTestReasoningEffort,
		DurationMs:      time.Since(started).Milliseconds(),
		LastRunAt:       channelIQTimePtr(started),
	}
	if err != nil {
		result.Error = err.Error()
		_ = s.store.SaveResult(ctx, result)
		return
	}
	if run != nil {
		result.DurationMs = run.Duration.Milliseconds()
		result.TotalTokens = run.Tokens
		result.SVG = run.SVG
		result.Error = run.Error
		if run.Success && strings.TrimSpace(run.SVG) != "" {
			result.Status = ChannelIQStatusSuccess
			result.Error = ""
		} else if run.Success {
			result.Status = ChannelIQStatusFailed
			result.Error = "未能从回复中提取 SVG"
		} else if result.Error == "" {
			result.Error = "iq test failed"
		}
	}
	_ = s.store.SaveResult(ctx, result)
}

func (s *ChannelIQService) accountExcluded(ctx context.Context, accountID int64) bool {
	current, err := s.normalizedSettings(ctx)
	if err != nil || current == nil {
		return false
	}
	return channelIQHasID(current.ExcludedAccountIDs, accountID)
}

func (s *ChannelIQService) normalizedSettings(ctx context.Context) (*ChannelIQSettings, error) {
	settings, err := s.store.GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	if settings == nil {
		settings = &ChannelIQSettings{}
	}
	return normalizeChannelIQSettings(*settings), nil
}

func (s *ChannelIQService) listWatchAccounts(ctx context.Context, settings *ChannelIQSettings) ([]Account, error) {
	accounts, err := s.listGroupOpenAIAccounts(ctx, settings.GroupIDs)
	if err != nil {
		return nil, err
	}
	return channelIQRejectIDs(accounts, settings.ExcludedAccountIDs), nil
}

func (s *ChannelIQService) listExcludedAccounts(ctx context.Context, settings *ChannelIQSettings) ([]Account, error) {
	accounts, err := s.listGroupOpenAIAccounts(ctx, settings.GroupIDs)
	if err != nil {
		return nil, err
	}
	return channelIQKeepIDs(accounts, settings.ExcludedAccountIDs), nil
}

func (s *ChannelIQService) accountInSelectedGroups(ctx context.Context, settings *ChannelIQSettings, accountID int64) (bool, error) {
	accounts, err := s.listGroupOpenAIAccounts(ctx, settings.GroupIDs)
	if err != nil {
		return false, err
	}
	for _, account := range accounts {
		if account.ID == accountID {
			return true, nil
		}
	}
	return false, nil
}

func (s *ChannelIQService) listGroupOpenAIAccounts(ctx context.Context, groupIDs []int64) ([]Account, error) {
	seen := map[int64]Account{}
	for _, groupID := range groupIDs {
		if groupID <= 0 {
			continue
		}
		accounts, err := s.accounts.ListByGroup(ctx, groupID)
		if err != nil {
			return nil, err
		}
		for _, account := range accounts {
			if !strings.EqualFold(account.Platform, PlatformOpenAI) {
				continue
			}
			if _, ok := seen[account.ID]; ok {
				continue
			}
			seen[account.ID] = account
		}
	}
	out := make([]Account, 0, len(seen))
	for _, account := range seen {
		out = append(out, account)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name == out[j].Name {
			return out[i].ID < out[j].ID
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

func normalizeChannelIQSettings(in ChannelIQSettings) *ChannelIQSettings {
	return &ChannelIQSettings{
		GroupIDs:           channelIQUniquePositiveIDs(in.GroupIDs),
		ExcludedAccountIDs: channelIQUniquePositiveIDs(in.ExcludedAccountIDs),
		AutoEnabled:        in.AutoEnabled,
		IntervalSeconds:    channelIQNormalizeInterval(in.IntervalSeconds),
		Prompt:             channelIQNormalizePrompt(in.Prompt),
		Model:              channelIQNormalizeModel(in.Model),
		UpdatedAt:          in.UpdatedAt,
	}
}

func channelIQNormalizeInterval(interval int) int {
	if interval < channelIQMinIntervalSec {
		return channelIQDefaultInterval
	}
	if interval > channelIQMaxIntervalSec {
		return channelIQMaxIntervalSec
	}
	return interval
}

func channelIQNormalizePrompt(prompt string) string {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return channelIQDefaultPrompt
	}
	return prompt
}

func channelIQNormalizeModel(model string) string {
	model = strings.TrimSpace(model)
	if model == "" {
		return channelIQDefaultModel
	}
	return model
}

func channelIQUniquePositiveIDs(ids []int64) []int64 {
	seen := map[int64]struct{}{}
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func channelIQHasID(ids []int64, id int64) bool {
	for _, existing := range ids {
		if existing == id {
			return true
		}
	}
	return false
}

func channelIQAppendID(ids []int64, id int64) []int64 {
	if channelIQHasID(ids, id) {
		return channelIQUniquePositiveIDs(ids)
	}
	return channelIQUniquePositiveIDs(append(append([]int64(nil), ids...), id))
}

func channelIQRemoveID(ids []int64, id int64) []int64 {
	out := make([]int64, 0, len(ids))
	for _, existing := range ids {
		if existing == id {
			continue
		}
		out = append(out, existing)
	}
	return channelIQUniquePositiveIDs(out)
}

func channelIQRejectIDs(accounts []Account, excluded []int64) []Account {
	skip := map[int64]struct{}{}
	for _, id := range excluded {
		skip[id] = struct{}{}
	}
	out := make([]Account, 0, len(accounts))
	for _, account := range accounts {
		if _, ok := skip[account.ID]; ok {
			continue
		}
		out = append(out, account)
	}
	return out
}

func channelIQKeepIDs(accounts []Account, keep []int64) []Account {
	want := map[int64]struct{}{}
	for _, id := range keep {
		want[id] = struct{}{}
	}
	out := make([]Account, 0, len(keep))
	for _, account := range accounts {
		if _, ok := want[account.ID]; ok {
			out = append(out, account)
		}
	}
	return out
}

func channelIQTimePtr(t time.Time) *time.Time {
	return &t
}
