package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"unicode/utf8"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type ingressRejectSettingRepo struct {
	service.SettingRepository
	getValueCalls int
}

func (r *ingressRejectSettingRepo) GetValue(context.Context, string) (string, error) {
	r.getValueCalls++
	return "", service.ErrSettingNotFound
}

func (r *ingressRejectSettingRepo) GetMultiple(context.Context, []string) (map[string]string, error) {
	r.getValueCalls++
	return map[string]string{}, nil
}

func (r *ingressRejectSettingRepo) Set(context.Context, string, string) error {
	return nil
}

type ingressRejectOpsRepo struct {
	service.OpsRepository
	insertCalls int
}

func (r *ingressRejectOpsRepo) InsertErrorLog(context.Context, *service.OpsInsertErrorLogInput) (int64, error) {
	r.insertCalls++
	return 0, nil
}

func (r *ingressRejectOpsRepo) BatchInsertErrorLogs(context.Context, []*service.OpsInsertErrorLogInput) (int64, error) {
	r.insertCalls++
	return 0, nil
}

func TestOpsErrorLogQueueByteBudget(t *testing.T) {
	previousBytes := opsErrorLogQueueBytes.Load()
	previousLen := opsErrorLogQueueLen.Load()
	opsErrorLogQueueBytes.Store(0)
	opsErrorLogQueueLen.Store(0)
	t.Cleanup(func() {
		opsErrorLogQueueBytes.Store(previousBytes)
		opsErrorLogQueueLen.Store(previousLen)
	})

	if !reserveOpsErrorLogQueueBytes(opsErrorLogMaxQueueBytes - 1) {
		t.Fatal("first reservation within byte budget should succeed")
	}
	if reserveOpsErrorLogQueueBytes(2) {
		t.Fatal("reservation beyond byte budget should be rejected")
	}
	if got := OpsErrorLogQueueBytes(); got != opsErrorLogMaxQueueBytes-1 {
		t.Fatalf("queued bytes = %d, want %d", got, opsErrorLogMaxQueueBytes-1)
	}
	if got := OpsErrorLogQueueLength(); got != 1 {
		t.Fatalf("queue length = %d, want 1", got)
	}
}

func TestEstimateOpsErrorLogJobBytesIncludesVariablePayloads(t *testing.T) {
	base := estimateOpsErrorLogJobBytes(&service.OpsInsertErrorLogInput{})
	message := "upstream message"
	detail := "upstream detail"
	events := `[{"error":"x"}]`
	entry := &service.OpsInsertErrorLogInput{
		ErrorBody:            strings.Repeat("x", 1024),
		ErrorMessage:         "client error",
		UserAgent:            "test-agent",
		UpstreamErrorMessage: &message,
		UpstreamErrorDetail:  &detail,
		UpstreamErrorsJSON:   &events,
	}
	if got := estimateOpsErrorLogJobBytes(entry); got <= base+1024 {
		t.Fatalf("estimated bytes = %d, expected variable payloads above %d", got, base+1024)
	}
}

func resetOpsErrorLoggerStateForTest(t *testing.T) {
	t.Helper()

	opsErrorLogMu.Lock()
	ch := opsErrorLogQueue
	opsErrorLogQueue = nil
	opsErrorLogStopping = true
	opsErrorLogMu.Unlock()

	if ch != nil {
		close(ch)
	}
	opsErrorLogWorkersWg.Wait()

	opsErrorLogOnce = sync.Once{}
	opsErrorLogStopOnce = sync.Once{}
	opsErrorLogWorkersWg = sync.WaitGroup{}
	opsErrorLogMu = sync.RWMutex{}
	opsErrorLogStopping = false

	opsErrorLogQueueLen.Store(0)
	opsErrorLogEnqueued.Store(0)
	opsErrorLogDropped.Store(0)
	opsErrorLogProcessed.Store(0)
	opsErrorLogSanitized.Store(0)
	opsErrorLogLastDropLogAt.Store(0)

	opsErrorLogShutdownCh = make(chan struct{})
	opsErrorLogShutdownOnce = sync.Once{}
	opsErrorLogDrained.Store(false)
}

func TestAttachOpsRequestBodyToEntry_SanitizeAndTrim(t *testing.T) {
	resetOpsErrorLoggerStateForTest(t)
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	raw := []byte(`{"access_token":"secret-token","messages":[{"role":"user","content":"hello"}]}`)
	setOpsRequestContext(c, "claude-3", false, raw)

	entry := &service.OpsInsertErrorLogInput{}
	attachOpsRequestBodyToEntry(c, entry)

	require.NotNil(t, entry.RequestBodyBytes)
	require.Equal(t, len(raw), *entry.RequestBodyBytes)
	require.NotNil(t, entry.RequestBodyJSON)
	require.NotContains(t, *entry.RequestBodyJSON, "secret-token")
	require.Contains(t, *entry.RequestBodyJSON, "[REDACTED]")
	require.Equal(t, int64(1), OpsErrorLogSanitizedTotal())
}

func TestAttachOpsRequestBodyToEntry_InvalidJSONKeepsSize(t *testing.T) {
	resetOpsErrorLoggerStateForTest(t)
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	raw := []byte("not-json")
	setOpsRequestContext(c, "claude-3", false, raw)

	entry := &service.OpsInsertErrorLogInput{}
	attachOpsRequestBodyToEntry(c, entry)

	require.Nil(t, entry.RequestBodyJSON)
	require.NotNil(t, entry.RequestBodyBytes)
	require.Equal(t, len(raw), *entry.RequestBodyBytes)
	require.False(t, entry.RequestBodyTruncated)
	require.Equal(t, int64(1), OpsErrorLogSanitizedTotal())
}

func TestEnqueueOpsErrorLog_QueueFullDrop(t *testing.T) {
	resetOpsErrorLoggerStateForTest(t)

	// 禁止 enqueueOpsErrorLog 触发 workers，使用测试队列验证满队列降级。
	opsErrorLogOnce.Do(func() {})

	opsErrorLogMu.Lock()
	opsErrorLogQueue = make(chan opsErrorLogJob, 1)
	opsErrorLogMu.Unlock()

	ops := service.NewOpsService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	entry := &service.OpsInsertErrorLogInput{ErrorPhase: "upstream", ErrorType: "upstream_error"}

	enqueueOpsErrorLog(ops, entry)
	enqueueOpsErrorLog(ops, entry)

	require.Equal(t, int64(1), OpsErrorLogEnqueuedTotal())
	require.Equal(t, int64(1), OpsErrorLogDroppedTotal())
	require.Equal(t, int64(1), OpsErrorLogQueueLength())
}

func TestAttachOpsRequestBodyToEntry_EarlyReturnBranches(t *testing.T) {
	resetOpsErrorLoggerStateForTest(t)
	gin.SetMode(gin.TestMode)

	entry := &service.OpsInsertErrorLogInput{}
	attachOpsRequestBodyToEntry(nil, entry)
	attachOpsRequestBodyToEntry(&gin.Context{}, nil)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	// 无请求体 key
	attachOpsRequestBodyToEntry(c, entry)
	require.Nil(t, entry.RequestBodyJSON)
	require.Nil(t, entry.RequestBodyBytes)
	require.False(t, entry.RequestBodyTruncated)

	// 错误类型
	c.Set(opsRequestBodyKey, "not-bytes")
	attachOpsRequestBodyToEntry(c, entry)
	require.Nil(t, entry.RequestBodyJSON)
	require.Nil(t, entry.RequestBodyBytes)

	// 空 bytes
	c.Set(opsRequestBodyKey, []byte{})
	attachOpsRequestBodyToEntry(c, entry)
	require.Nil(t, entry.RequestBodyJSON)
	require.Nil(t, entry.RequestBodyBytes)

	require.Equal(t, int64(0), OpsErrorLogSanitizedTotal())
}

func TestEnqueueOpsErrorLog_EarlyReturnBranches(t *testing.T) {
	resetOpsErrorLoggerStateForTest(t)

	ops := service.NewOpsService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	entry := &service.OpsInsertErrorLogInput{ErrorPhase: "upstream", ErrorType: "upstream_error"}

	// nil 入参分支
	enqueueOpsErrorLog(nil, entry)
	enqueueOpsErrorLog(ops, nil)
	require.Equal(t, int64(0), OpsErrorLogEnqueuedTotal())

	// shutdown 分支
	close(opsErrorLogShutdownCh)
	enqueueOpsErrorLog(ops, entry)
	require.Equal(t, int64(0), OpsErrorLogEnqueuedTotal())

	// stopping 分支
	resetOpsErrorLoggerStateForTest(t)
	opsErrorLogMu.Lock()
	opsErrorLogStopping = true
	opsErrorLogMu.Unlock()
	enqueueOpsErrorLog(ops, entry)
	require.Equal(t, int64(0), OpsErrorLogEnqueuedTotal())

	// queue nil 分支（防止启动 worker 干扰）
	resetOpsErrorLoggerStateForTest(t)
	opsErrorLogOnce.Do(func() {})
	opsErrorLogMu.Lock()
	opsErrorLogQueue = nil
	opsErrorLogMu.Unlock()
	enqueueOpsErrorLog(ops, entry)
	require.Equal(t, int64(0), OpsErrorLogEnqueuedTotal())
}

func TestOpsCaptureWriterPool_ResetOnRelease(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

	writer := acquireOpsCaptureWriter(c.Writer)
	require.NotNil(t, writer)
	_, err := writer.buf.WriteString("temp-error-body")
	require.NoError(t, err)

	releaseOpsCaptureWriter(writer)

	reused := acquireOpsCaptureWriter(c.Writer)
	defer releaseOpsCaptureWriter(reused)

	require.Zero(t, reused.buf.Len(), "writer should be reset before reuse")
}

func TestOpsCaptureWriterPool_DropsLargeBuffers(t *testing.T) {
	w := &opsCaptureWriter{}
	w.buf.Grow(opsCaptureWriterPoolMaxRetainedCapacity + 1)
	require.False(t, shouldPoolOpsCaptureWriter(w))
}

func TestEnqueueOpsErrorLog_SanitizesAndBoundsBodyBeforeQueue(t *testing.T) {
	setupOpsErrorLogTestQueue(t, 1)
	ops := service.NewOpsService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	secret := strings.Repeat("s", service.OpsErrorLogQueueBodyMaxBytes)
	entry := &service.OpsInsertErrorLogInput{
		ErrorPhase: "request",
		ErrorType:  "api_error",
		ErrorBody:  `{"authorization":"Bearer ` + secret + `","message":"failed"}`,
	}

	enqueueOpsErrorLog(ops, entry)
	job := <-opsErrorLogQueue
	require.LessOrEqual(t, len(job.entry.ErrorBody), service.OpsErrorLogQueueBodyMaxBytes)
	require.NotContains(t, job.entry.ErrorBody, secret)
	require.Equal(t, int64(1), OpsErrorLogSanitizedTotal())
}

func TestOpsErrorLoggerMiddleware_DoesNotBreakOuterMiddlewares(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(middleware2.Recovery())
	r.Use(middleware2.RequestLogger())
	r.Use(middleware2.Logger())
	r.GET("/v1/messages", OpsErrorLoggerMiddleware(nil), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/messages", nil)

	require.NotPanics(t, func() {
		r.ServeHTTP(rec, req)
	})
	require.Equal(t, http.StatusNoContent, rec.Code)
}

// setupOpsErrorLogTestQueue 阻止 enqueueOpsErrorLog 启动真实 worker，改用可检查的测试队列。
func setupOpsErrorLogTestQueue(t *testing.T, size int) {
	t.Helper()
	resetOpsErrorLoggerStateForTest(t)
	opsErrorLogOnce.Do(func() {})
	opsErrorLogMu.Lock()
	opsErrorLogQueue = make(chan opsErrorLogJob, size)
	opsErrorLogMu.Unlock()
}

func TestOpsErrorLoggerMiddleware_HardSkipsIngressRejection(t *testing.T) {
	setupOpsErrorLogTestQueue(t, 4)
	gin.SetMode(gin.TestMode)

	settings := &ingressRejectSettingRepo{}
	repo := &ingressRejectOpsRepo{}
	ops := service.NewOpsService(repo, settings, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	// Construction may read unrelated runtime settings; only request-path reads matter here.
	settings.getValueCalls = 0

	router := gin.New()
	router.Use(OpsErrorLoggerMiddleware(ops))
	router.GET("/v1/messages", func(c *gin.Context) {
		middleware2.MarkIngressRejected(c, middleware2.IngressRejectInvalidAPIKey)
		c.JSON(http.StatusUnauthorized, gin.H{"code": "INVALID_API_KEY", "message": "Invalid API key"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/messages", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.JSONEq(t, `{"code":"INVALID_API_KEY","message":"Invalid API key"}`, w.Body.String())
	require.Zero(t, settings.getValueCalls, "ingress rejection must bypass monitoring settings reads")
	require.Zero(t, repo.insertCalls, "ingress rejection must bypass inserts")
	require.Zero(t, OpsErrorLogEnqueuedTotal(), "ingress rejection must not enter the error queue")
}

func TestNormalizeOpsPersistentUserAgentBoundsAndPreservesUTF8(t *testing.T) {
	value := strings.Repeat("a", opsErrorLogMaxUserAgentBytes-1) + "你" + strings.Repeat("b", 32)
	got := normalizeOpsPersistentUserAgent("  " + value + "  ")
	require.LessOrEqual(t, len(got), opsErrorLogMaxUserAgentBytes)
	require.True(t, utf8.ValidString(got))
	require.NotContains(t, got, "b")
}

// 就地(in-band) SSE 错误挂在已固化的 HTTP 200 流上：wire 状态码为 200，
// 常规 status>=400 采集路径不会触发。logOpsStreamError 必须据 MarkOpsStreamError
// 补记一条错误日志，且用 IntendedStatus(429) 分级、StatusCode 仍记 wire 的 200。
func TestLogOpsStreamError_RecordsInBandConcurrencyLimit(t *testing.T) {
	setupOpsErrorLogTestQueue(t, 4)

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	c.Set(opsModelKey, "test-model")

	service.MarkOpsStreamError(c, "rate_limit_error",
		"Concurrency limit exceeded for account, please retry later", http.StatusTooManyRequests)

	ops := service.NewOpsService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	logOpsStreamError(c, ops, http.StatusOK)

	require.Equal(t, int64(1), OpsErrorLogEnqueuedTotal())
	require.Equal(t, int64(1), OpsErrorLogQueueLength())

	job := <-opsErrorLogQueue
	require.NotNil(t, job.entry)
	require.Equal(t, "rate_limit_error", job.entry.ErrorType)
	require.Equal(t, "request", job.entry.ErrorPhase)
	require.True(t, job.entry.IsBusinessLimited)
	require.True(t, job.entry.Stream)
	require.Equal(t, http.StatusOK, job.entry.StatusCode) // wire 状态码保持 200
	require.Equal(t, "P1", job.entry.Severity)            // 用 IntendedStatus 429 分级
	require.Equal(t, "test-model", job.entry.Model)
	require.Equal(t, "Concurrency limit exceeded for account, please retry later", job.entry.ErrorMessage)
}

func TestLogOpsStreamError_UpstreamFailureCountsTowardsSLA(t *testing.T) {
	setupOpsErrorLogTestQueue(t, 4)

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	c.Set(opsModelKey, "gpt-5.6-sol")

	service.MarkOpsStreamFailure(
		c,
		"upstream_error",
		service.OpenAIUpstreamHTTP2StreamErrorCode,
		"Upstream HTTP/2 stream failed",
		http.StatusBadGateway,
	)

	ops := service.NewOpsService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	logOpsStreamError(c, ops, http.StatusOK)

	job := <-opsErrorLogQueue
	require.NotNil(t, job.entry)
	require.Equal(t, http.StatusBadGateway, job.entry.StatusCode)
	require.Equal(t, "upstream_error", job.entry.ErrorType)
	require.Equal(t, "upstream", job.entry.ErrorPhase)
	require.Equal(t, "provider", job.entry.ErrorOwner)
	require.False(t, job.entry.IsBusinessLimited)
	require.Contains(t, job.entry.ErrorBody, service.OpenAIUpstreamHTTP2StreamErrorCode)
}

// 未标记流内错误时 logOpsStreamError 必须是 no-op（不误记正常的 200 流）。
func TestLogOpsStreamError_NoopWhenNotMarked(t *testing.T) {
	setupOpsErrorLogTestQueue(t, 4)

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	ops := service.NewOpsService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	logOpsStreamError(c, ops, http.StatusOK)

	require.Equal(t, int64(0), OpsErrorLogEnqueuedTotal())
}

// 命中 skip_monitoring=true 透传规则时不落库，与其它采集分支一致。
func TestLogOpsStreamError_SkipWhenPassthroughSkipMonitoring(t *testing.T) {
	setupOpsErrorLogTestQueue(t, 4)

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	service.MarkOpsStreamError(c, "upstream_error", "Upstream request failed", http.StatusBadGateway)
	c.Set(service.OpsSkipPassthroughKey, true)

	ops := service.NewOpsService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	logOpsStreamError(c, ops, http.StatusOK)

	require.Equal(t, int64(0), OpsErrorLogEnqueuedTotal())
}

// MarkOpsStreamError 采用「首个标记生效」：后续的通用兜底帧不得覆盖根因错误。
func TestMarkOpsStreamError_FirstWins(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	service.MarkOpsStreamError(c, "rate_limit_error", "Concurrency limit exceeded for account", http.StatusTooManyRequests)
	service.MarkOpsStreamError(c, "upstream_error", "Upstream request failed", http.StatusBadGateway)

	se, ok := service.GetOpsStreamError(c)
	require.True(t, ok)
	require.Equal(t, "rate_limit_error", se.ErrType)
	require.Equal(t, "Concurrency limit exceeded for account", se.Message)
	require.Equal(t, http.StatusTooManyRequests, se.IntendedStatus)
}
func TestIsKnownOpsErrorType(t *testing.T) {
	known := []string{
		"invalid_request_error",
		"authentication_error",
		"rate_limit_error",
		"billing_error",
		"subscription_error",
		"upstream_error",
		"overloaded_error",
		"api_error",
		"not_found_error",
		"forbidden_error",
	}
	for _, k := range known {
		require.True(t, isKnownOpsErrorType(k), "expected known: %s", k)
	}

	unknown := []string{"<nil>", "null", "", "random_error", "some_new_type", "<nil>\u003e"}
	for _, u := range unknown {
		require.False(t, isKnownOpsErrorType(u), "expected unknown: %q", u)
	}
}

func TestClassifyOpsLocalModelConfigurationRejection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalModelConfiguration)

	phase, isBusinessLimited, errorOwner, errorSource := classifyOpsErrorLog(
		c,
		"model_not_found",
		"Model \"gpt-missing\" is not supported by any configured account in this group",
		"",
		http.StatusNotFound,
	)

	require.Equal(t, "routing", phase)
	require.True(t, isBusinessLimited)
	require.Equal(t, "platform", errorOwner)
	require.Equal(t, "gateway", errorSource)
}

func TestClassifyOpsLocalModelConfigurationOverridesStaleUpstreamMarkers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalModelConfiguration)
	c.Set(service.OpsUpstreamStatusCodeKey, http.StatusUnauthorized)
	c.Set(service.OpsUpstreamErrorsKey, []*service.OpsUpstreamErrorEvent{{
		Stage:              string(service.GatewayFailureStageAccountAuth),
		UpstreamStatusCode: http.StatusUnauthorized,
	}})

	phase, limited, owner, source := classifyOpsErrorLog(c, "model_not_found", "unsupported configured model", "", http.StatusNotFound)

	require.Equal(t, "routing", phase)
	require.True(t, limited)
	require.Equal(t, "platform", owner)
	require.Equal(t, "gateway", source)
}

func TestClassifyOpsLocalModelConfigurationRequiresMarkerAndReason(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set(service.OpsClientBusinessLimitedReasonKey, service.OpsClientBusinessLimitedReasonLocalModelConfiguration)
	c.Set(service.OpsUpstreamStatusCodeKey, http.StatusBadGateway)

	phase, limited, owner, source := classifyOpsErrorLog(c, "upstream_error", "provider failed", "", http.StatusBadGateway)

	require.Equal(t, "upstream", phase)
	require.False(t, limited)
	require.Equal(t, "provider", owner)
	require.Equal(t, "upstream_http", source)
}

func TestNormalizeOpsErrorType(t *testing.T) {
	tests := []struct {
		name    string
		errType string
		code    string
		want    string
	}{
		// Known types pass through.
		{"known invalid_request_error", "invalid_request_error", "", "invalid_request_error"},
		{"known rate_limit_error", "rate_limit_error", "", "rate_limit_error"},
		{"known upstream_error", "upstream_error", "", "upstream_error"},

		// Unknown/garbage types are rejected and fall through to code-based or default.
		{"nil literal from upstream", "<nil>", "", "api_error"},
		{"null string", "null", "", "api_error"},
		{"random string", "something_weird", "", "api_error"},

		// Unknown type but known code still maps correctly.
		{"nil with INSUFFICIENT_BALANCE code", "<nil>", "INSUFFICIENT_BALANCE", "billing_error"},
		{"nil with USAGE_LIMIT_EXCEEDED code", "<nil>", "USAGE_LIMIT_EXCEEDED", "subscription_error"},

		// Empty type falls through to code-based mapping.
		{"empty type with balance code", "", "INSUFFICIENT_BALANCE", "billing_error"},
		{"empty type with subscription code", "", "SUBSCRIPTION_NOT_FOUND", "subscription_error"},
		{"empty type no code", "", "", "api_error"},

		// Known type overrides conflicting code-based mapping.
		{"known type overrides conflicting code", "rate_limit_error", "INSUFFICIENT_BALANCE", "rate_limit_error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeOpsErrorType(tt.errType, tt.code)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestSetOpsEndpointContext_SetsContextKeys(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	setOpsEndpointContext(c, "claude-3-5-sonnet-20241022", int16(2)) // stream

	v, ok := c.Get(opsUpstreamModelKey)
	require.True(t, ok)
	vStr, ok := v.(string)
	require.True(t, ok)
	require.Equal(t, "claude-3-5-sonnet-20241022", vStr)

	rt, ok := c.Get(opsRequestTypeKey)
	require.True(t, ok)
	rtVal, ok := rt.(int16)
	require.True(t, ok)
	require.Equal(t, int16(2), rtVal)
}

func TestSetOpsEndpointContext_EmptyModelNotStored(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	setOpsEndpointContext(c, "", int16(1))

	_, ok := c.Get(opsUpstreamModelKey)
	require.False(t, ok, "empty upstream model should not be stored")

	rt, ok := c.Get(opsRequestTypeKey)
	require.True(t, ok)
	rtVal, ok := rt.(int16)
	require.True(t, ok)
	require.Equal(t, int16(1), rtVal)
}

func TestSetOpsEndpointContext_NilContext(t *testing.T) {
	require.NotPanics(t, func() {
		setOpsEndpointContext(nil, "model", int16(1))
	})
}
