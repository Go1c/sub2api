//go:build unit

package service

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestResolveOpenAITextStreamIdleTimeout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest("POST", "/v1/responses", nil)

	require.Equal(t, time.Duration(0), resolveOpenAITextStreamIdleTimeout(c, 0))
	require.Equal(t, 180*time.Second, resolveOpenAITextStreamIdleTimeout(c, 180))

	MarkOpenAINativeCompactionV2(c)
	require.Equal(t, time.Duration(0), resolveOpenAITextStreamIdleTimeout(c, 0),
		"配置关闭时 compact 也不强行打开超时")
	require.Equal(t, openAICompactStreamIdleTimeout, resolveOpenAITextStreamIdleTimeout(c, 180))
	require.Equal(t, openAICompactStreamIdleTimeout, resolveOpenAITextStreamIdleTimeout(c, 900))
}
