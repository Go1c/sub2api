package telemetry

import (
	"context"
	"errors"
	"time"
)

const (
	EventSignupPageView       = "signup_page_view"
	EventLoginPageView        = "login_page_view"
	EventVerifyCodeRequest    = "verify_code_request"
	EventVerifyCodeSuccess    = "verify_code_success"
	EventVerifyCodeFailure    = "verify_code_failure"
	EventAuthRegisterSubmit   = "auth_register_submit"
	EventAuthRegisterSuccess  = "auth_register_success"
	EventAuthRegisterFailure  = "auth_register_failure"
	EventAuthLoginSubmit      = "auth_login_submit"
	EventAuthLogin2FARequired = "auth_login_2fa_required"
	EventAuthLoginSuccess     = "auth_login_success"
	EventAuthLoginFailure     = "auth_login_failure"
	EventDownloadDialogOpen   = "download_dialog_open"
	EventDownloadStart        = "download_start"
	EventAppFirstLaunch       = "app_first_launch"
	EventAppLoginSuccess      = "app_login_success"
	EventCodexSetupSuccess    = "codex_setup_success"
	EventClaudeSetupSuccess   = "claude_setup_success"
	EventAppReady             = "app_ready"

	IngestSourceClient = "client"
	IngestSourceServer = "server"

	ClientSourceUnknown = "unknown"

	AuthorityFirstPartyIngest             = "first_party_ingest"
	MeasureEventCountAndUniqueAnonymousID = "event_count_and_unique_anonymous_id"
	maxStatsRange                         = 90 * 24 * time.Hour
	dedupWindowMillis                     = int64(120000)
	maxRouteLen                           = 128
)

var (
	ErrUnknownEvent    = errors.New("unknown telemetry event")
	ErrDuplicateEvent  = errors.New("duplicate telemetry event")
	ErrInvalidStats    = errors.New("invalid telemetry stats query")
	ErrStatsRangeOrder = errors.New("from must be less than or equal to to")
	ErrStatsRangeCap   = errors.New("stats range cannot exceed 90 days")
)

var allowedEvents = map[string]struct{}{
	EventSignupPageView:       {},
	EventLoginPageView:        {},
	EventVerifyCodeRequest:    {},
	EventVerifyCodeSuccess:    {},
	EventVerifyCodeFailure:    {},
	EventAuthRegisterSubmit:   {},
	EventAuthRegisterSuccess:  {},
	EventAuthRegisterFailure:  {},
	EventAuthLoginSubmit:      {},
	EventAuthLogin2FARequired: {},
	EventAuthLoginSuccess:     {},
	EventAuthLoginFailure:     {},
	EventDownloadDialogOpen:   {},
	EventDownloadStart:        {},
	EventAppFirstLaunch:       {},
	EventAppLoginSuccess:      {},
	EventCodexSetupSuccess:    {},
	EventClaudeSetupSuccess:   {},
	EventAppReady:             {},
}

var lifetimeDedupEvents = map[string]struct{}{
	EventAuthRegisterSuccess: {},
	EventAppFirstLaunch:      {},
}

var windowedDedupEvents = map[string]struct{}{
	EventAuthLoginSuccess:   {},
	EventAppLoginSuccess:    {},
	EventCodexSetupSuccess:  {},
	EventClaudeSetupSuccess: {},
	EventAppReady:           {},
}

var inheritFirstTouchEvents = map[string]struct{}{
	EventAppFirstLaunch:     {},
	EventAppLoginSuccess:    {},
	EventCodexSetupSuccess:  {},
	EventClaudeSetupSuccess: {},
	EventAppReady:           {},
}

type ingestRequest struct {
	Event string            `json:"event"`
	TS    int64             `json:"ts"`
	Props map[string]string `json:"props"`
}

type sanitizedProps struct {
	ClientSource       string
	Route              string
	AuthMethod         string
	Platform           string
	Destination        string
	ErrorCode          string
	AttributionID      string
	FirstTouchSource   string
	FirstTouchMedium   string
	FirstTouchCampaign string
	LastTouchSource    string
	LastTouchMedium    string
	LastTouchCampaign  string
}

type Event struct {
	ID                 int64
	Event              string
	OccurredAt         time.Time
	ClientSource       string
	Route              string
	AuthMethod         string
	Platform           string
	Destination        string
	ErrorCode          string
	AttributionID      string
	FirstTouchSource   string
	FirstTouchMedium   string
	FirstTouchCampaign string
	LastTouchSource    string
	LastTouchMedium    string
	LastTouchCampaign  string
	UserID             *int64
	DedupKey           string
	IngestSource       string
}

type AccountAttribution struct {
	UserID             int64
	FirstTouchSource   string
	FirstTouchMedium   string
	FirstTouchCampaign string
	FirstAttributionID string
	LastTouchSource    string
	LastTouchMedium    string
	LastTouchCampaign  string
	LastAttributionID  string
	FirstTouchAt       time.Time
	LastTouchAt        time.Time
}

type StatsQuery struct {
	From         int64
	To           int64
	ClientSource string
	Campaign     string
	Event        string
}

type StatsRange struct {
	From int64 `json:"from"`
	To   int64 `json:"to"`
}

type StatsRow struct {
	Event                string `json:"event"`
	EventCount           int64  `json:"event_count"`
	UniqueAttributionIDs int64  `json:"unique_attribution_ids"`
	Measure              string `json:"measure"`
}

type StatsResult struct {
	Authority string            `json:"authority"`
	Range     StatsRange        `json:"range"`
	Filters   map[string]string `json:"filters,omitempty"`
	Rows      []StatsRow        `json:"rows"`
}

type AccessTokenIdentifier interface {
	IdentifyAccessToken(token string) (int64, error)
}

type Repository interface {
	InsertEvent(ctx context.Context, event Event) error
	FindSuccessMatch(ctx context.Context, event Event) (*Event, error)
	PatchEventObservability(ctx context.Context, id int64, incoming Event) error
	GetAttribution(ctx context.Context, userID int64) (*AccountAttribution, error)
	UpsertAttribution(ctx context.Context, attr AccountAttribution) error
	Aggregate(ctx context.Context, query StatsQuery) ([]StatsRow, error)
}
