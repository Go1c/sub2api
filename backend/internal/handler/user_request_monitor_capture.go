package handler

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *GatewayHandler) captureClientRequest(c *gin.Context, model string, body []byte) {
	if h == nil {
		return
	}
	captureClientRequest(h.userRequestMonitorService, c, model, body)
}

func (h *OpenAIGatewayHandler) captureClientRequest(c *gin.Context, model string, body []byte) {
	if h == nil {
		return
	}
	captureClientRequest(h.userRequestMonitorService, c, model, body)
}

func captureClientRequest(svc *service.OpsUserRequestMonitorService, c *gin.Context, model string, body []byte) {
	if svc == nil || c == nil || len(body) == 0 {
		return
	}
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		return
	}
	apiKey, _ := middleware2.GetAPIKeyFromContext(c)
	var apiKeyID *int64
	var groupID *int64
	if apiKey != nil {
		apiKeyID = int64Ptr(apiKey.ID)
		groupID = cloneHandlerInt64Ptr(apiKey.GroupID)
	}
	svc.CaptureClientRequestIfEnabled(c.Request.Context(), &service.OpsCaptureClientRequestInput{
		UserID:          subject.UserID,
		APIKeyID:        apiKeyID,
		GroupID:         groupID,
		RequestID:       requestMonitorRequestID(c),
		Model:           model,
		InboundEndpoint: GetInboundEndpoint(c),
		Method:          c.Request.Method,
		ContentType:     c.GetHeader("Content-Type"),
		Body:            body,
	})
}

func requestMonitorRequestID(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}
	if v, ok := c.Request.Context().Value(ctxkey.ClientRequestID).(string); ok {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	if v, ok := c.Request.Context().Value(ctxkey.RequestID).(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func int64Ptr(v int64) *int64 {
	return &v
}

func cloneHandlerInt64Ptr(v *int64) *int64 {
	if v == nil {
		return nil
	}
	out := *v
	return &out
}
