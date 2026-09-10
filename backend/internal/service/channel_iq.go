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
	GroupIDs        []int64   `json:"group_ids"`
	AutoEnabled     bool      `json:"auto_enabled"`
	IntervalSeconds int       `json:"interval_seconds"`
	Prompt          string    `json:"prompt"`
	Model           string    `json:"model"`
	UpdatedAt       time.Time `json:"updated_at,omitempty"`
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

type ChannelIQOverview struct {
	Settings ChannelIQSettings `json:"settings"`
	Items    []ChannelIQItem   `json:"items"`
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
	accounts, err := s.listWatchAccounts(ctx, settings.GroupIDs)
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
	return &ChannelIQOverview{Settings: *settings, Items: items}, nil
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
	accounts, err := s.listWatchAccounts(ctx, settings.GroupIDs)
	if err != nil {
		return err
	}
	ids := make([]int64, 0, len(accounts))
	for _, account := range accounts {
		ids = append(ids, account.ID)
	}
	return s.startBatch(ctx, ids, settings)
}

func (s *ChannelIQService) RunOne(ctx context.Context, accountID int64) error {
	settings, err := s.normalizedSettings(ctx)
	if err != nil {
		return err
	}
	accounts, err := s.listWatchAccounts(ctx, settings.GroupIDs)
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

func (s *ChannelIQService) listWatchAccounts(ctx context.Context, groupIDs []int64) ([]Account, error) {
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
	seen := map[int64]struct{}{}
	groupIDs := make([]int64, 0, len(in.GroupIDs))
	for _, id := range in.GroupIDs {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		groupIDs = append(groupIDs, id)
	}
	interval := in.IntervalSeconds
	if interval < channelIQMinIntervalSec {
		interval = channelIQDefaultInterval
	}
	if interval > channelIQMaxIntervalSec {
		interval = channelIQMaxIntervalSec
	}
	prompt := strings.TrimSpace(in.Prompt)
	if prompt == "" {
		prompt = channelIQDefaultPrompt
	}
	model := strings.TrimSpace(in.Model)
	if model == "" {
		model = channelIQDefaultModel
	}
	return &ChannelIQSettings{
		GroupIDs:        groupIDs,
		AutoEnabled:     in.AutoEnabled,
		IntervalSeconds: interval,
		Prompt:          prompt,
		Model:           model,
		UpdatedAt:       in.UpdatedAt,
	}
}

func channelIQTimePtr(t time.Time) *time.Time {
	return &t
}
