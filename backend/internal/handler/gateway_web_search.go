package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/websearch"
	"github.com/Wei-Shaw/sub2api/internal/pkg/xai"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func (h *GatewayHandler) WebSearch(c *gin.Context) {
	isXSearch := c.GetBool("grok_x_search_endpoint")
	var req grokStandaloneSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"type":    "invalid_request_error",
			"message": err.Error(),
		}})
		return
	}
	query := strings.TrimSpace(req.Query)
	if query == "" {
		query = strings.TrimSpace(req.Input)
	}
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"type":    "invalid_request_error",
			"message": "query is required",
		}})
		return
	}
	req.Query = query
	maxResults := 0
	if req.MaxResults != nil {
		maxResults = *req.MaxResults
	}
	maxResults = normalizeGrokWebSearchMaxResults(maxResults)
	searchModel := resolveGrokStandaloneSearchModel()
	searchLabel := "web_search"
	if isXSearch {
		searchLabel = "x_search"
	}

	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok || apiKey == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{
			"type":    "authentication_error",
			"message": "API key required",
		}})
		return
	}

	if apiKey.Group == nil || apiKey.Group.Platform != "grok" {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"type":    "invalid_request_error",
			"message": searchLabel + " is only supported for grok groups",
		}})
		return
	}

	// Billing eligibility (same as other requests)
	subscription, _ := middleware2.GetSubscriptionFromContext(c)
	if err := h.billingCacheService.CheckBillingEligibility(c.Request.Context(), apiKey.User, apiKey, apiKey.Group, subscription, service.QuotaPlatform(c.Request.Context(), apiKey)); err != nil {
		status, code, message, retryAfter := billingErrorDetails(err)
		if retryAfter > 0 {
			c.Header("Retry-After", strconv.Itoa(retryAfter))
		}
		c.JSON(status, gin.H{"error": gin.H{"type": code, "message": message}})
		return
	}

	// 余额模式在途预留（与计费同口径估算；计费任务扣减余额缓存后才释放）。
	inflightDone, inflightErr := reserveInflightBalance(c, h.billingCacheService, h.gatewayService, apiKey, subscription, service.InflightEstimateRequest{Model: "grok-" + strings.ReplaceAll(searchLabel, "_", "-"), Kind: service.InflightEstimatePerRequest, SearchCalls: 1})
	if inflightErr != nil {
		status, code, message, retryAfter := billingErrorDetails(inflightErr)
		if retryAfter > 0 {
			c.Header("Retry-After", strconv.Itoa(retryAfter))
		}
		c.JSON(status, gin.H{"error": gin.H{"type": code, "message": message}})
		return
	}
	defer inflightDone()

	subject, _ := middleware2.GetAuthSubjectFromContext(c)
	reqLog := requestLogger(c, "handler.gateway.web_search")
	// Audit user search query before upstream Grok web_search traffic.
	auditBody, _ := json.Marshal(map[string]any{
		"messages": []map[string]any{{
			"role": "user", "content": req.Query,
		}},
	})
	if decision := h.checkSecurityAudit(c, reqLog, apiKey, subject, service.ContentModerationProtocolOpenAIChat, searchModel, auditBody); decision != nil && !decision.AllowNextStage {
		status := decision.HTTPStatus
		if status == 0 {
			status = http.StatusForbidden
		}
		code := decision.ErrorCode
		if code == "" {
			code = "content_policy_violation"
		}
		msg := decision.ClientMessage
		if msg == "" {
			msg = "Request blocked by content policy"
		}
		c.JSON(status, gin.H{"error": gin.H{"type": code, "message": msg}})
		return
	}

	// Use exactly the same scheduling as other requests (SelectAccountWithLoadAwareness handles load, rate limit, sticky, etc.)
	groupID := apiKey.GroupID
	if groupID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"type":    "invalid_request_error",
			"message": "group required",
		}})
		return
	}

	failedAccounts := make(map[int64]struct{})
	var account *service.Account
	var accountReleaseFunc func()
	var nativeResp *websearch.SearchResponse
	var providerName string
	var err error

	// Acquire + release holder for the whole handler (including failover retries).
	defer func() {
		if accountReleaseFunc != nil {
			accountReleaseFunc()
		}
	}()

	// First attempt + up to 3 failover accounts (max 4 total).
	for attempt := 0; attempt < 4; attempt++ {
		selected, selectErr := h.gatewayService.SelectAccountWithLoadAwareness(
			c.Request.Context(), groupID, "", searchModel, failedAccounts, "", 0,
		)
		if selectErr != nil {
			if attempt == 0 {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{
					"type":    "scheduling_error",
					"message": selectErr.Error(),
				}})
				return
			}
			break
		}
		if selected == nil || selected.Account == nil {
			if attempt == 0 {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{
					"type":    "scheduling_error",
					"message": "No available accounts",
				}})
				return
			}
			break
		}

		release, acquireOK, acquireErr := h.acquireWebSearchAccountSlot(c, selected)
		if !acquireOK {
			// First hop: surface concurrency errors; later hops try another account.
			if attempt == 0 && acquireErr != nil {
				h.handleConcurrencyError(c, acquireErr, "account", false)
				return
			}
			failedAccounts[selected.Account.ID] = struct{}{}
			continue
		}
		account = selected.Account
		accountReleaseFunc = release

		if isXSearch {
			nativeResp, providerName, err = h.doGrokNativeXSearch(c.Request.Context(), c, account, req, searchModel, maxResults)
		} else {
			nativeResp, providerName, err = h.doGrokNativeWebSearch(c.Request.Context(), c, account, req.Query, maxResults, searchModel)
		}
		if err == nil {
			break
		}
		var failoverErr *service.UpstreamFailoverError
		if !errors.As(err, &failoverErr) || !failoverErr.ShouldRetryNextAccount() {
			break
		}
		failedAccounts[account.ID] = struct{}{}
		if accountReleaseFunc != nil {
			accountReleaseFunc()
			accountReleaseFunc = nil
		}
		account = nil
	}
	if err != nil || nativeResp == nil {
		msg := "web search failed"
		if err != nil {
			msg = err.Error()
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"type": "web_search_error", "message": msg}})
		return
	}
	if account == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{
			"type":    "scheduling_error",
			"message": "No available accounts",
		}})
		return
	}

	userAgent := c.GetHeader("User-Agent")
	clientIP := ip.GetClientIP(c)
	inboundEndpoint := GetInboundEndpoint(c)
	upstreamEndpoint := GetUpstreamEndpoint(c, account.Platform)
	requestPayloadHash := service.HashUsageRequestPayload([]byte(req.Query))
	quotaPlatform := service.QuotaPlatform(c.Request.Context(), apiKey)
	// Request IDs are billing idempotency keys, so they must be unique per invocation.
	// Query/IP/UA hashes would collapse repeated identical searches into one charge.
	searchRequestID := searchLabel + ":" + uuid.NewString()
	if apiKey.Group != nil {
		if p := apiKey.Group.GetSearchPricePer1k(); p != nil && *p == 0 {
			logger.L().With(
				zap.String("component", "handler.gateway.web_search"),
				zap.Int64("group_id", apiKey.Group.ID),
			).Info("gateway.web_search.search_price_per_1k_explicit_free")
		}
	}
	h.submitMandatoryUsageRecordTask(c.Request.Context(), func(ctx context.Context) {
		if err := h.gatewayService.RecordUsage(ctx, &service.RecordUsageInput{
			Result: &service.ForwardResult{
				RequestID:   searchRequestID,
				Model:       "grok-" + strings.ReplaceAll(searchLabel, "_", "-"),
				SearchCount: 1,
				Duration:    0,
			},
			APIKey:             apiKey,
			User:               apiKey.User,
			Account:            account,
			Subscription:       subscription,
			InboundEndpoint:    inboundEndpoint,
			UpstreamEndpoint:   upstreamEndpoint,
			UserAgent:          userAgent,
			IPAddress:          clientIP,
			RequestPayloadHash: requestPayloadHash,
			APIKeyService:      h.apiKeyService,
			QuotaPlatform:      quotaPlatform,
		}); err != nil {
			logger.L().With(
				zap.String("component", "handler.gateway.web_search"),
				zap.Int64("user_id", apiKey.User.ID),
				zap.Int64("api_key_id", apiKey.ID),
				zap.Int64("account_id", account.ID),
			).Error("gateway.web_search.record_usage_failed", zap.Error(err))
		}
	})

	c.JSON(http.StatusOK, gin.H{
		"query":       req.Query,
		"results":     nativeResp.Results,
		"provider":    providerName,
		"max_results": maxResults,
	})
}

// acquireWebSearchAccountSlot resolves an immediate slot or WaitPlan wait.
// On failure returns (nil, false, err); err is non-nil for concurrency acquire
// failures so the first hop can map them to HTTP. Wait-queue full returns
// (nil, false, nil) so failover can try another account.
func (h *GatewayHandler) acquireWebSearchAccountSlot(
	c *gin.Context,
	selected *service.AccountSelectionResult,
) (release func(), ok bool, acquireErr error) {
	if selected == nil || selected.Account == nil {
		return nil, false, nil
	}
	if selected.Acquired {
		return selected.ReleaseFunc, true, nil
	}
	if selected.WaitPlan == nil || h.concurrencyHelper == nil {
		return nil, false, nil
	}
	account := selected.Account
	accountWaitCounted := false
	canWait, waitErr := h.concurrencyHelper.IncrementAccountWaitCount(c.Request.Context(), account.ID, selected.WaitPlan.MaxWaiting)
	if waitErr != nil {
		logger.L().Warn("gateway.web_search.account_wait_counter_increment_failed",
			zap.Int64("account_id", account.ID),
			zap.Error(waitErr),
		)
		// Best-effort wait without counter (same as first-hop legacy path).
	} else if !canWait {
		return nil, false, nil
	} else {
		accountWaitCounted = true
	}
	releaseWait := func() {
		if accountWaitCounted {
			h.concurrencyHelper.DecrementAccountWaitCount(c.Request.Context(), account.ID)
			accountWaitCounted = false
		}
	}
	streamStarted := false
	slotRelease, err := h.concurrencyHelper.AcquireAccountSlotWithWaitTimeout(
		c,
		account.ID,
		selected.WaitPlan.MaxConcurrency,
		selected.WaitPlan.Timeout,
		false,
		&streamStarted,
	)
	releaseWait()
	if err != nil {
		return nil, false, err
	}
	return slotRelease, true, nil
}

// doGrokNativeWebSearch executes web search using the Grok account's native capability
// by calling the responses endpoint with web_search tool, then normalizes sources to unified format.
func (h *GatewayHandler) doGrokNativeWebSearch(ctx context.Context, c *gin.Context, account *service.Account, query string, maxResults int, model string) (*websearch.SearchResponse, string, error) {
	maxResults = normalizeGrokWebSearchMaxResults(maxResults)

	// Build a minimal responses request that triggers Grok web search tool.
	// Ask for structured metadata because xAI action.sources commonly contains URLs only.
	searchBody := map[string]any{
		"model":   xai.ResolveDefaultTextModel(model),
		"input":   buildGrokWebSearchPrompt(query, maxResults),
		"tools":   []map[string]any{{"type": "web_search"}},
		"include": []string{"web_search_call.action.sources"},
		"store":   false,
		"stream":  false,
	}
	bodyBytes, _ := json.Marshal(searchBody)

	respBytes, err := h.gatewayService.DoGrokNativeResponsesJSON(ctx, account, bodyBytes)
	if err != nil {
		return nil, "", err
	}

	// Extract sources from Grok responses output.
	// Prefer web_search_call.action.sources (standardized), fallback to annotations or text links.
	results := extractGrokWebSearchSources(respBytes, maxResults)

	return &websearch.SearchResponse{
		Results: results,
		Query:   query,
	}, "grok-native", nil
}

func (h *GatewayHandler) doGrokNativeXSearch(ctx context.Context, c *gin.Context, account *service.Account, req grokStandaloneSearchRequest, model string, maxResults int) (*websearch.SearchResponse, string, error) {
	maxResults = normalizeGrokWebSearchMaxResults(maxResults)
	bodyBytes, err := buildGrokXSearchResponsesBody(req, model)
	if err != nil {
		return nil, "", err
	}
	respBytes, err := h.gatewayService.DoGrokNativeResponsesJSON(ctx, account, bodyBytes)
	if err != nil {
		return nil, "", err
	}
	results := extractGrokWebSearchSources(respBytes, maxResults)
	return &websearch.SearchResponse{
		Results: results,
		Query:   req.Query,
	}, "grok-native", nil
}

func buildGrokWebSearchPrompt(query string, maxResults int) string {
	return fmt.Sprintf(`Search the web for the user query below. Return ONLY valid JSON with this exact shape: {"results":[{"url":"https://...","title":"page title","snippet":"concise factual summary"}]}. Return at most %d unique results. Every URL must be an actual web_search source. Populate a non-empty title and snippet for every result. Do not wrap the JSON in markdown.

User query:
%s`, normalizeGrokWebSearchMaxResults(maxResults), query)
}
