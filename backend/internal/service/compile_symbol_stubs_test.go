//go:build unit

package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

// Compile stubs for test-only references dropped by the 0.2.17 merge.

const (
	PlatformKiro                                           = "kiro"
	SettingKeyAffiliateAdminRechargeEnabled                = "affiliate_admin_recharge_enabled"
	SettingKeyCyberPolicyUserAllowlist                     = "cyber_policy_user_allowlist"
	SettingKeyCompactHomeEnabled                           = "compact_home_enabled"
	SettingKeySubscriptionEnabled                          = "subscription_enabled"
	SettingKeyOpenAIImagesOAuthUnavailableCooldownSettings = "openai_images_oauth_unavailable_cooldown_settings"
	SettingKeyAccountSchedulingThresholds                  = "account_scheduling_thresholds"
	testCodexFingerprintSeed                               = "11111111-1111-4111-8111-111111111111"
	openAIImagesOAuthUnavailableMaxCooldownMinutes         = 24 * 60
	openAIImagesOAuthUnavailableDefaultCooldownMinutes     = 30
	deepSeekChatReasoningPlaceholderText                   = "reasoning"
)

type schedulingThresholdSF struct{}

func (schedulingThresholdSF) Forget(string) {}
func (schedulingThresholdSF) Do(string, func() (any, error)) (any, error, bool) {
	return nil, nil, false
}

var accountSchedulingThresholdsSF schedulingThresholdSF
var accountSchedulingThresholdsCache atomic.Value
var accountSchedulingThresholdsCacheTTL = time.Minute

type cachedAccountSchedulingThresholds struct {
	thresholds map[string]int
	expiresAt  int64
}

var codexCanonicalUAMu sync.RWMutex
var codexCanonicalUAResolver func() string

type ErrInvalidOpenAIServiceTier struct{ Tier string }

func (e *ErrInvalidOpenAIServiceTier) Error() string {
	if e == nil {
		return "invalid openai service tier"
	}
	return "invalid openai service tier: " + e.Tier
}

type codexModelsVisibilityAccountRepo struct {
	AccountRepository
	byGroup map[int64][]Account
}

func (r codexModelsVisibilityAccountRepo) ListSchedulableByGroupID(_ context.Context, groupID int64) ([]Account, error) {
	return append([]Account(nil), r.byGroup[groupID]...), nil
}

func (r codexModelsVisibilityAccountRepo) ListModelAvailabilityCandidates(_ context.Context, groupID *int64, _ []string, _ bool) ([]Account, error) {
	if groupID == nil {
		return nil, nil
	}
	return append([]Account(nil), r.byGroup[*groupID]...), nil
}

type countingCodexModelsAccountRepo struct {
	AccountRepository
	accounts        []Account
	err             error
	availabilityErr error
	groupID         *int64
	platforms       []string
	includeGrouped  bool
	calls           atomic.Int32
}

func (r *countingCodexModelsAccountRepo) ListSchedulableByGroupID(_ context.Context, _ int64) ([]Account, error) {
	r.calls.Add(1)
	if r.err != nil {
		return nil, r.err
	}
	return append([]Account(nil), r.accounts...), nil
}

func (r *countingCodexModelsAccountRepo) ListModelAvailabilityCandidates(_ context.Context, groupID *int64, platforms []string, includeGrouped bool) ([]Account, error) {
	if groupID != nil {
		value := *groupID
		r.groupID = &value
	}
	r.platforms = append([]string(nil), platforms...)
	r.includeGrouped = includeGrouped
	if r.availabilityErr != nil {
		return nil, r.availabilityErr
	}
	return append([]Account(nil), r.accounts...), nil
}

type OpenAIImagesOAuthUnavailableCooldownSettings struct {
	CooldownMinutes int
}
type UpstreamModelCatalog struct {
	Models   []string
	Metadata map[string]UpstreamModelMetadata
	Warnings []string
}

type refundPendingAuditDetail struct{}

func (refundPendingAuditDetail) applyTo(*RefundPlan) {}

type responsesStreamOutputItems struct{}

func (r *responsesStreamOutputItems) Observe([]byte)                         {}
func (r *responsesStreamOutputItems) BuildOutput() ([]json.RawMessage, bool) { return nil, false }
func (r *responsesStreamOutputItems) HasItems() bool                         { return false }

type openAIResponsesCompatibilityOptions struct{}
type openAIWSContextWindowBoundary struct {
	Changed                   bool
	PreviousResponseIDRemoved bool
	WindowID                  string
}

type CompositeModelOwnership struct {
	TargetPlatform string
	Matched        bool
}

// ApplyAccountSchedulingThreshold from internal/service/ratelimit_service.go
func (s *RateLimitService) ApplyAccountSchedulingThreshold(ctx context.Context, account *Account) bool {
	panic("compile stub")
}

// BuildCodexModelsManifestForGroup from internal/service/openai_codex_models_service.go
func (s *GatewayService) BuildCodexModelsManifestForGroup(
	ctx context.Context,
	group *Group,
	platformOverride string,
	modelIDs []string,
) ([]byte, error) {
	panic("compile stub")
}

// CacheCreditsSnapshot from internal/service/openai_quota_service.go
func (s *OpenAIQuotaService) CacheCreditsSnapshot(ctx context.Context, accountID int64, usage *OpenAIQuotaUsage) error {
	panic("compile stub")
}

// CodexModelsManifestETag from internal/service/openai_codex_models_service.go
func CodexModelsManifestETag(body []byte) string {
	panic("compile stub")
}

// ExtractGrokVideoBillingFromStatusBody from internal/service/grok_media.go
func ExtractGrokVideoBillingFromStatusBody(statusBody []byte, pending *GrokVideoPendingBilling, requestID string) *OpenAIForwardResult {
	panic("compile stub")
}

// GetAccountSchedulingThresholds from internal/service/setting_features.go
func (s *SettingService) GetAccountSchedulingThresholds(ctx context.Context) map[string]int {
	panic("compile stub")
}

// GetOpenAICodexCanonicalUserAgent from internal/service/setting_gateway_runtime.go
func (s *SettingService) GetOpenAICodexCanonicalUserAgent(ctx context.Context) string {
	panic("compile stub")
}

// GetOpenAIImagesOAuthUnavailableCooldownSettings from internal/service/setting_features.go
func (s *SettingService) GetOpenAIImagesOAuthUnavailableCooldownSettings(ctx context.Context) (*OpenAIImagesOAuthUnavailableCooldownSettings, error) {
	panic("compile stub")
}

// HandleOpenAIImageCapabilityLoss from internal/service/ratelimit_service.go
func (s *RateLimitService) HandleOpenAIImageCapabilityLoss(ctx context.Context, account *Account, statusCode int, responseBody []byte) bool {
	panic("compile stub")
}

// IsForwardableOpenAIResponsesRequestPath from internal/service/openai_gateway_request_body.go
func IsForwardableOpenAIResponsesRequestPath(c *gin.Context) bool {
	panic("compile stub")
}

// IsGrokVideoStatusBillable from internal/service/grok_media.go
func IsGrokVideoStatusBillable(statusBody []byte) bool {
	panic("compile stub")
}

// IsOpenAINativeCompactionV2 from internal/service/openai_compact_body_signal.go
func IsOpenAINativeCompactionV2(c *gin.Context) bool {
	panic("compile stub")
}

// IsOpsRequestScopedError from internal/service/ops_upstream_context.go
func IsOpsRequestScopedError(c *gin.Context) bool {
	panic("compile stub")
}

// MergeGroupConfiguredCodexModels from internal/service/openai_codex_models_service.go
func (s *OpenAIGatewayService) MergeGroupConfiguredCodexModels(
	ctx context.Context,
	group *Group,
	manifest *OpenAIModelsResponse,
	ifNoneMatch string,
) error {
	panic("compile stub")
}

// MigrateGrokDefaultTextModel from internal/service/setting_gateway_runtime.go
func (s *SettingService) MigrateGrokDefaultTextModel(ctx context.Context) error {
	panic("compile stub")
}

// ResolveGrokMediaVideoRequestAccount from internal/service/grok_media.go
func (s *OpenAIGatewayService) ResolveGrokMediaVideoRequestAccount(
	ctx context.Context,
	groupID *int64,
	requestID string,
	userID, apiKeyID int64,
) (int64, error) {
	panic("compile stub")
}

// SelectGrokMediaVideoRequestAccount from internal/service/grok_media.go
func (s *OpenAIGatewayService) SelectGrokMediaVideoRequestAccount(
	ctx context.Context, groupID *int64, sessionHash string, accountID int64, requestedModel string,
) (*AccountSelectionResult, OpenAIAccountScheduleDecision, error) {
	panic("compile stub")
}

// SetCodexCanonicalUserAgentResolver from internal/service/openai_codex_identity.go
func SetCodexCanonicalUserAgentResolver(resolver func() string) {
	panic("compile stub")
}

// SetOpenAIImagesOAuthUnavailableCooldownSettings from internal/service/setting_features.go
func (s *SettingService) SetOpenAIImagesOAuthUnavailableCooldownSettings(ctx context.Context, settings *OpenAIImagesOAuthUnavailableCooldownSettings) error {
	panic("compile stub")
}

// SetUpstreamModelMetadataSnapshot from internal/service/upstream_models.go
func (a *Account) SetUpstreamModelMetadataSnapshot(snapshot UpstreamModelMetadataSnapshot) {
	panic("compile stub")
}

// SyncUpstreamModelCatalog from internal/service/upstream_models.go
func (s *AccountTestService) SyncUpstreamModelCatalog(ctx context.Context, account *Account) (*UpstreamModelCatalog, error) {
	panic("compile stub")
}

// ValidateAccountGroupBindings from internal/service/admin_account.go
func (s *adminServiceImpl) ValidateAccountGroupBindings(ctx context.Context, groupIDs []int64) error {
	panic("compile stub")
}

// ValidateOpenAIServiceTierField from internal/service/openai_gateway_request_body.go
func ValidateOpenAIServiceTierField(body []byte) (string, error) {
	panic("compile stub")
}

// anthropicMessagesCredentialAllowed from internal/service/gateway_upstream_request.go
func anthropicMessagesCredentialAllowed(account *Account) bool {
	panic("compile stub")
}

// applySyncedAPIKeyCodexModelMetadata from internal/service/openai_codex_models_service.go
func applySyncedAPIKeyCodexModelMetadata(body []byte, account *Account, overwriteLocalDefaults bool) ([]byte, error) {
	panic("compile stub")
}

// billableModelWithFallback from internal/service/gateway_usage_billing.go
func (s *GatewayService) billableModelWithFallback(ctx context.Context, apiKey *APIKey, billingModel string, fallbacks ...string) string {
	panic("compile stub")
}

// buildCodexCLIUserAgent from internal/service/openai_codex_identity.go
func buildCodexCLIUserAgent(version string) string {
	panic("compile stub")
}

// buildCodexModelsManifestForAccounts from internal/service/openai_codex_models_service.go
func buildCodexModelsManifestForAccounts(
	effectivePlatform string,
	modelIDs []string,
	accounts []Account,
	group *Group,
	compositeRoutes []CompositeModelRoute,
	compositeRoutesAvailable bool,
) ([]byte, error) {
	panic("compile stub")
}

// buildOpenAIResponseFailedSSE from internal/service/openai_gateway_response_handling.go
func buildOpenAIResponseFailedSSE(responseID, model string, createdAt int64, source []byte, fallbackMessage string) string {
	panic("compile stub")
}

// buildOpenAIWSHTTPBridgeFailedEvent from internal/service/openai_ws_http_bridge.go
func buildOpenAIWSHTTPBridgeFailedEvent(responseID, model string, createdAt int64, source []byte, fallbackMessage string) []byte {
	panic("compile stub")
}

// combineOpenAIWSReplayItems from internal/service/openai_ws_forwarder_payload.go
func combineOpenAIWSReplayItems(history, delta []json.RawMessage) []json.RawMessage {
	panic("compile stub")
}

// completeAPIKeyCodexModelsManifestMetadata from internal/service/openai_codex_models_service.go
func completeAPIKeyCodexModelsManifestMetadata(body []byte, completeAll bool, account *Account) ([]byte, error) {
	panic("compile stub")
}

// compositeBillableModel from internal/service/gateway_usage_billing.go
func (s *GatewayService) compositeBillableModel(ctx context.Context, apiKey *APIKey, billingModel, concreteBillingModel string) string {
	panic("compile stub")
}

// configuredUpstreamModelsForCapabilitySync from internal/service/upstream_models.go
func configuredUpstreamModelsForCapabilitySync(account *Account) []string {
	panic("compile stub")
}

// convertOpenAIModelListToCodexManifestForAccount from internal/service/openai_codex_models_service.go
func convertOpenAIModelListToCodexManifestForAccount(body []byte, account *Account) []byte {
	panic("compile stub")
}

// coolOpenAIImagesOAuthTool from internal/service/openai_images_responses.go
func (s *OpenAIGatewayService) coolOpenAIImagesOAuthTool(ctx context.Context, account *Account) {
	panic("compile stub")
}

// createPendingRefundOrderWithDetailForTest from internal/service/payment_refund_test.go
func createPendingRefundOrderWithDetailForTest(t *testing.T, ctx context.Context, client *dbent.Client, suffix, pendingDetail string) *dbent.PaymentOrder {
	panic("compile stub")
}

// decodeCodexManifestModels from internal/service/openai_codex_models_service_test.go
func decodeCodexManifestModels(t *testing.T, body []byte) []map[string]any {
	panic("compile stub")
}

// deepseekPeakMultiplierAt from internal/service/billing_service.go
func deepseekPeakMultiplierAt(now time.Time) float64 {
	panic("compile stub")
}

// defaultAccountSchedulingThresholds from internal/service/setting_update.go
func defaultAccountSchedulingThresholds() map[string]int {
	panic("compile stub")
}

// effortsFromManifestModel from internal/service/openai_codex_models_service_test.go
func effortsFromManifestModel(t *testing.T, model map[string]any) []string {
	panic("compile stub")
}

// ensureDeepSeekChatReasoningPlaceholders from internal/service/openai_gateway_responses_chat_fallback.go
func ensureDeepSeekChatReasoningPlaceholders(account *Account, body []byte) []byte {
	panic("compile stub")
}

// expireCodexModelsManifestCache from internal/service/openai_codex_models_service_test.go
func expireCodexModelsManifestCache(s *OpenAIGatewayService, age time.Duration) {
	panic("compile stub")
}

// extractContentModerationKeywordText from internal/service/content_moderation_input.go
func extractContentModerationKeywordText(protocol string, body []byte) string {
	panic("compile stub")
}

// extractUpstreamModelCatalog from internal/service/upstream_models.go
func extractUpstreamModelCatalog(body []byte, grok bool) ([]string, map[string]UpstreamModelMetadata, error) {
	panic("compile stub")
}

// filterOpenAIResponsesNoneReasoningEffortForAccount from internal/service/openai_gateway_request_body.go
func filterOpenAIResponsesNoneReasoningEffortForAccount(account *Account, body []byte) ([]byte, error) {
	panic("compile stub")
}

// finalizeRefundFailed from internal/service/payment_refund.go
func (s *PaymentService) finalizeRefundFailed(ctx context.Context, p *RefundPlan, gErr error) (*RefundResult, error) {
	panic("compile stub")
}

// floorClaudeCLIUserAgentVersion from internal/service/identity_service.go
func floorClaudeCLIUserAgentVersion(ua string) (string, bool) {
	panic("compile stub")
}

// formatOpsReportInteger from internal/service/ops_scheduled_report_service.go
func formatOpsReportInteger(value int64) string {
	panic("compile stub")
}

// grokStickyAffinitySeed from internal/service/openai_gateway_scheduling.go
func grokStickyAffinitySeed(sessionID string, body []byte) string {
	panic("compile stub")
}

// grokTokenRefreshWindowWithJitter from internal/service/grok_token_refresher.go
func grokTokenRefreshWindowWithJitter(accountID int64, refreshWindow time.Duration) time.Duration {
	panic("compile stub")
}

// handleOpenAIStreamTerminalAccountSideEffects from internal/service/openai_gateway_passthrough.go
func (s *OpenAIGatewayService) handleOpenAIStreamTerminalAccountSideEffects(
	c *gin.Context,
	account *Account,
	payload []byte,
	message string,
	headers http.Header,
	canonicalModel ...string,
) (int, bool) {
	panic("compile stub")
}

// handleOpenAIWSFailureAccountSideEffects from internal/service/openai_ws_forwarder_support.go
func (s *OpenAIGatewayService) handleOpenAIWSFailureAccountSideEffects(ctx context.Context, account *Account, canonicalModel string, headers http.Header, payload []byte) bool {
	panic("compile stub")
}

// hasResolvableTokenPricing from internal/service/gateway_usage_billing.go
func (s *GatewayService) hasResolvableTokenPricing(ctx context.Context, model string, apiKey *APIKey) bool {
	panic("compile stub")
}

// isCNProviderQuotaExhausted403 from internal/service/ratelimit_cn_providers.go
func isCNProviderQuotaExhausted403(account *Account, responseBody []byte, upstreamMsg string) bool {
	panic("compile stub")
}

// isOpenAIHTTPUpstreamAccessStateError from internal/service/openai_gateway_upstream_errors.go
func isOpenAIHTTPUpstreamAccessStateError(_ int, _ string, body []byte) bool {
	panic("compile stub")
}

// isOpenAIImageCapabilityLossError from internal/service/ratelimit_service.go
func isOpenAIImageCapabilityLossError(statusCode int, body []byte) bool {
	panic("compile stub")
}

// isOpenAIUpstreamAccessStateError from internal/service/openai_gateway_upstream_errors.go
func isOpenAIUpstreamAccessStateError(_ string, body []byte) bool {
	panic("compile stub")
}

// lastMessageBody from internal/service/notification_email_service_test.go
func (s *notificationEmailTestSMTPServer) lastMessageBody(t *testing.T) string {
	panic("compile stub")
}

// latestRefundPendingDetail from internal/service/payment_refund.go
func (s *PaymentService) latestRefundPendingDetail(ctx context.Context, oid int64) (refundPendingAuditDetail, error) {
	panic("compile stub")
}

// markOpenAIWSClientVisibleFailure from internal/service/openai_ws_forwarder_support.go
func markOpenAIWSClientVisibleFailure(c *gin.Context, eventType string, payload []byte) {
	panic("compile stub")
}

// mergeHostedImageGenToolUsage from internal/service/openai_gateway_response_handling.go
func mergeHostedImageGenToolUsage(imageGen gjson.Result, usage *OpenAIUsage) {
	panic("compile stub")
}

// modelRoutingAppliesToPlatform from internal/service/gateway_scheduling.go
func modelRoutingAppliesToPlatform(targetPlatform, groupPlatform string) bool {
	panic("compile stub")
}

// newCodexCatalogMappedAccount from internal/service/openai_codex_models_service_test.go
func newCodexCatalogMappedAccount(
	id int64,
	target string,
	displayName string,
	levels []string,
	modalities []string,
	contextWindow int64,
	schedulable bool,
	extraMapping map[string]any,
) Account {
	panic("compile stub")
}

// newCodexModelsOAuthCacheServer from internal/service/openai_codex_models_service_test.go
func newCodexModelsOAuthCacheServer(t *testing.T, body string) (*httptest.Server, *atomic.Int32) {
	panic("compile stub")
}

// newOpenAIAccountFailoverError from internal/service/openai_gateway_upstream_errors.go
func (s *OpenAIGatewayService) newOpenAIAccountFailoverError(
	account *Account,
	statusCode int,
	responseHeaders http.Header,
	responseBody []byte,
	upstreamMsg string,
	shouldDisable bool,
	retryableOnSameAccount bool,
) *UpstreamFailoverError {
	panic("compile stub")
}

// newOpenAIWSDownstreamWriteContext from internal/service/openai_ws_forwarder_ingress.go
func newOpenAIWSDownstreamWriteContext(controlCtx context.Context, hooks *OpenAIWSIngressHooks, timeout time.Duration) (context.Context, context.CancelFunc) {
	panic("compile stub")
}

// newOpenAIWSRateLimitFailoverError from internal/service/openai_ws_forwarder_support.go
func (s *OpenAIGatewayService) newOpenAIWSRateLimitFailoverError(account *Account, headers http.Header, responseBody []byte, message string) *UpstreamFailoverError {
	panic("compile stub")
}

// newResponsesStreamOutputItems from internal/service/openai_gateway_response_handling.go
func newResponsesStreamOutputItems() *responsesStreamOutputItems {
	panic("compile stub")
}

// nonStreamingTerminalFailureFailover from internal/service/openai_gateway_passthrough.go
func (s *OpenAIGatewayService) nonStreamingTerminalFailureFailover(
	c *gin.Context,
	resp *http.Response,
	account *Account,
	passthrough bool,
	terminalType string,
	payload []byte,
	message string,
	canonicalModel ...string,
) *UpstreamFailoverError {
	panic("compile stub")
}

// normalizeCacheControlTTLOrder from internal/service/gateway_claude_oauth_body.go
func normalizeCacheControlTTLOrder(body []byte) []byte {
	panic("compile stub")
}

// normalizeOpenAIAPIKeyStoreFalseReasoningReplay from internal/service/openai_gateway_request_body.go
func normalizeOpenAIAPIKeyStoreFalseReasoningReplay(body []byte, knownStoreFalse bool) ([]byte, bool, error) {
	panic("compile stub")
}

// normalizeOpenAIAPIKeyStoreFalseReasoningReplayDecoded from internal/service/openai_gateway_request_body.go
func normalizeOpenAIAPIKeyStoreFalseReasoningReplayDecoded(body []byte, knownStoreFalse bool) ([]byte, bool, error) {
	panic("compile stub")
}

// normalizeOpenAIOAuthResponsesCompatibilityBody from internal/service/openai_gateway_request_body.go
func normalizeOpenAIOAuthResponsesCompatibilityBody(body []byte) ([]byte, bool, error) {
	panic("compile stub")
}

// normalizeOpenAIResponseFormatSchemasBody from internal/service/openai_gateway_request_body.go
func normalizeOpenAIResponseFormatSchemasBody(body []byte) ([]byte, bool, error) {
	panic("compile stub")
}

// normalizeOpenAIResponsesCompatibilityBodyWithOptions from internal/service/openai_gateway_request_body.go
func normalizeOpenAIResponsesCompatibilityBodyWithOptions(body []byte, account *Account, opts openAIResponsesCompatibilityOptions) ([]byte, bool, error) {
	panic("compile stub")
}

// normalizeOpenAIResponsesReasoningContentReplay from internal/service/openai_gateway_request_body.go
func normalizeOpenAIResponsesReasoningContentReplay(body []byte) ([]byte, bool, error) {
	panic("compile stub")
}

// normalizeOpenAIResponsesReasoningMode from internal/service/openai_gateway_request_body.go
func normalizeOpenAIResponsesReasoningMode(body []byte, model string) ([]byte, bool, error) {
	panic("compile stub")
}

// normalizeOpenAIResponsesWebSocketCompatibilityBody from internal/service/openai_gateway_request_body.go
func normalizeOpenAIResponsesWebSocketCompatibilityBody(body []byte, account *Account, responsesLite bool) ([]byte, bool, error) {
	panic("compile stub")
}

// normalizeOpenAIWSContextWindowBoundary from internal/service/openai_ws_forwarder_payload.go
func normalizeOpenAIWSContextWindowBoundary(
	payload []byte,
	previousWindowID string,
) ([]byte, openAIWSContextWindowBoundary, error) {
	panic("compile stub")
}

// normalizeOpenAIWSHandshakeCompatibility from internal/service/openai_ws_pool.go
func normalizeOpenAIWSHandshakeCompatibility(account *Account, headers http.Header) openAIWSHandshakeCompatibilityKey {
	panic("compile stub")
}

// normalizeOpenAIWSRoutingAffinity from internal/service/openai_ws_pool.go
func normalizeOpenAIWSRoutingAffinity(headers http.Header) string {
	panic("compile stub")
}

// openAICompatTerminalResponse from internal/service/openai_gateway_messages.go
func openAICompatTerminalResponse(event *apicompat.ResponsesStreamEvent, payload []byte) *apicompat.ResponsesResponse {
	panic("compile stub")
}

// openAICompatibleAccountEligibilityFailureReasonBeforeProfit from internal/service/openai_gateway_scheduling.go
func openAICompatibleAccountEligibilityFailureReasonBeforeProfit(ctx context.Context, account *Account, platform string, requestedModel string, requireCompact bool, requiredCapability OpenAIEndpointCapability) string {
	panic("compile stub")
}

// openAIConfiguredCodexModelIDsForGroup from internal/service/openai_codex_models_service.go
func openAIConfiguredCodexModelIDsForGroup(accounts []Account, group *Group) []string {
	panic("compile stub")
}

// openAIImagesTextFallbackError from internal/service/openai_images_responses.go
func openAIImagesTextFallbackError(body []byte) *OpenAIImagesUpstreamError {
	panic("compile stub")
}

// openAIImagesTextFallbackErrorForText from internal/service/openai_images_responses.go
func openAIImagesTextFallbackErrorForText(text string) *OpenAIImagesUpstreamError {
	panic("compile stub")
}

// openAIResponsesCompletedEventIsEmpty from internal/service/openai_gateway_response_handling.go
func openAIResponsesCompletedEventIsEmpty(data []byte, usage *OpenAIUsage) bool {
	panic("compile stub")
}

// openAISchedulingResetWindowEnd from internal/service/openai_account_scheduler.go
func openAISchedulingResetWindowEnd(account *Account, now time.Time) (time.Time, bool) {
	panic("compile stub")
}

// openAIStreamCredentialAuthFailure from internal/service/openai_gateway_passthrough.go
func openAIStreamCredentialAuthFailure(payload []byte) bool {
	panic("compile stub")
}

// openAIStreamDataStartsVisibleOutput from internal/service/openai_gateway_passthrough.go
func openAIStreamDataStartsVisibleOutput(data, eventType string) bool {
	panic("compile stub")
}

// openAISubscriptionLongContextExempt from internal/service/openai_gateway_usage.go
func openAISubscriptionLongContextExempt(account *Account, result *OpenAIForwardResult) bool {
	panic("compile stub")
}

// quotaSnapshot from internal/service/grok_quota_service_test.go
func (u *grokHybridUpstream) quotaSnapshot() ([]*http.Request, [][]byte) {
	panic("compile stub")
}

// refundFinalizePlan from internal/service/payment_refund.go
func (s *PaymentService) refundFinalizePlan(o *dbent.PaymentOrder) *RefundPlan {
	panic("compile stub")
}

// renewedSubscriptionTerm from internal/service/subscription_service.go
func renewedSubscriptionTerm(existingSub *UserSubscription, notes string, startsAt, expiresAt time.Time) *UserSubscription {
	panic("compile stub")
}

// replacePaymentProviderFactoryForTest from internal/service/payment_refund_test.go
func replacePaymentProviderFactoryForTest(t *testing.T, prov payment.Provider) func() {
	panic("compile stub")
}

// requireOpenAICodexProbeHeaders from internal/service/openai_codex_identity_test.go
func requireOpenAICodexProbeHeaders(t *testing.T, h http.Header) {
	panic("compile stub")
}

// resolveCompositeModelOwnership from internal/service/gateway_service.go
func (s *GatewayService) resolveCompositeModelOwnership(ctx context.Context, groupID int64, model string) (CompositeModelOwnership, error) {
	panic("compile stub")
}

// resolveModerationProxyURL from internal/service/content_moderation.go
func (s *ContentModerationService) resolveModerationProxyURL(ctx context.Context, proxyID int64) (string, error) {
	panic("compile stub")
}

// resolveOpenAIErrorSchedulingModel from internal/service/openai_gateway_scheduling.go
func resolveOpenAIErrorSchedulingModel(billingModel, upstreamModel string) string {
	panic("compile stub")
}

// resolveOpenAIForwardMappedModels from internal/service/openai_gateway_scheduling.go
func resolveOpenAIForwardMappedModels(account *Account, requestedModel string, requireCompact bool) (billingModel, upstreamModel string) {
	panic("compile stub")
}

// resolvedOpenAIUpstreamServiceTierFromObserver from internal/service/upstream_response_model.go
func resolvedOpenAIUpstreamServiceTierFromObserver(_ *upstreamResponseModelObserver, outboundBodyTier *string) *string {
	panic("compile stub")
}

// runOpenAIResponseFlushTestWithAccount from internal/service/openai_gateway_response_flush_test.go
func runOpenAIResponseFlushTestWithAccount(recorder *openAIResponseFlushRecorder, body io.ReadCloser, gatewayCfg config.GatewayConfig, account *Account) (*openaiStreamingResult, error) {
	panic("compile stub")
}

// shouldCoolOpenAIImagesToolForError from internal/service/openai_images_responses.go
func shouldCoolOpenAIImagesToolForError(upstreamErr *OpenAIImagesUpstreamError) bool {
	panic("compile stub")
}

// shouldRetryOpenAIOAuth429OnSameAccount from internal/service/openai_account_runtime_block_fastpath.go
func (s *OpenAIGatewayService) shouldRetryOpenAIOAuth429OnSameAccount(account *Account, statusCode int, shouldDisable bool) bool {
	panic("compile stub")
}

// startOpenAISSEKeepalive from internal/service/openai_compact_sse_keepalive.go
func startOpenAISSEKeepalive(c *gin.Context, interval time.Duration) func() {
	panic("compile stub")
}

// tempUnschedRuleUsesModelScope from internal/service/ratelimit_service.go
func tempUnschedRuleUsesModelScope(rule TempUnschedulableRule, modelKey string, statusCode int) bool {
	panic("compile stub")
}

// toolAnthropicSSEStream from internal/service/openai_gateway_anthropic_native_pump_test.go
func toolAnthropicSSEStream() string {
	panic("compile stub")
}

// upstreamModelListEndpointUnsupported from internal/service/upstream_models.go
func upstreamModelListEndpointUnsupported(err error) bool {
	panic("compile stub")
}

// upstreamModelRegistryBaseURL from internal/service/upstream_models.go
func upstreamModelRegistryBaseURL(account *Account) string {
	panic("compile stub")
}

// validAirwallexProviderConfig from internal/service/payment_config_providers_test.go
func validAirwallexProviderConfig(t *testing.T) map[string]string {
	panic("compile stub")
}

// validateAndNormalizeAccountSchedulingThresholds from internal/service/setting_update.go
func validateAndNormalizeAccountSchedulingThresholds(input map[string]int) (map[string]int, error) {
	panic("compile stub")
}
