//go:build unit

package admin

import (
	"bytes"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 回归分组平台枚举:kimi/zhipu/deepseek 必须能通过 Create/Update 的 binding 校验
// （历史 bug:调度/路由链路已支持 CN 平台分组,但 oneof 白名单漏加三平台,导致
// 平台分组无法创建、CN 账号"无可用分组"）;非法值仍须被拒。
func bindGroupPlatformJSON(t *testing.T, target any, body string) error {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	return c.ShouldBindJSON(target)
}

func TestGroupPlatformBinding_AllowedPlatforms(t *testing.T) {
	allowed := []string{
		"anthropic", "openai", "gemini", "antigravity", "grok",
		"kimi", "zhipu", "deepseek", "composite",
	}
	for _, platform := range allowed {
		t.Run("create_"+platform, func(t *testing.T) {
			var req CreateGroupRequest
			body := fmt.Sprintf(`{"name":"g","platform":%q}`, platform)
			require.NoError(t, bindGroupPlatformJSON(t, &req, body),
				"platform %q 应通过 CreateGroupRequest 校验", platform)
			require.Equal(t, platform, req.Platform)
		})
		t.Run("update_"+platform, func(t *testing.T) {
			var req UpdateGroupRequest
			body := fmt.Sprintf(`{"platform":%q}`, platform)
			require.NoError(t, bindGroupPlatformJSON(t, &req, body),
				"platform %q 应通过 UpdateGroupRequest 校验", platform)
			require.Equal(t, platform, req.Platform)
		})
	}
}

func TestUpdateGroupBinding_ReadsModelPricingFieldNames(t *testing.T) {
	var req UpdateGroupRequest
	body := `{
		"rate_multiplier": 0.55,
		"model_pricing": [{
			"platform": "deepseek",
			"models": ["deepseek-v4.1-flash", "deepseek-flash"],
			"billing_mode": "token",
			"input_price": 0.000002,
			"output_price": 0.000008,
			"cache_read_price": 0.00000004
		}]
	}`
	require.NoError(t, bindGroupPlatformJSON(t, &req, body))
	require.NotNil(t, req.RateMultiplier)
	require.InDelta(t, 0.55, *req.RateMultiplier, 1e-9)
	require.NotNil(t, req.ModelPricing)
	require.Len(t, *req.ModelPricing, 1)
	entry := (*req.ModelPricing)[0]
	require.Equal(t, []string{"deepseek-v4.1-flash", "deepseek-flash"}, entry.Models)
	require.Equal(t, "token", string(entry.BillingMode))
	require.NotNil(t, entry.InputPrice)
	require.InDelta(t, 2e-6, *entry.InputPrice, 1e-15)
	require.NotNil(t, entry.OutputPrice)
	require.InDelta(t, 8e-6, *entry.OutputPrice, 1e-15)
	require.NotNil(t, entry.CacheReadPrice)
	require.InDelta(t, 4e-8, *entry.CacheReadPrice, 1e-18)
}

func TestGroupPlatformBinding_RejectsInvalidPlatforms(t *testing.T) {
	invalid := []string{
		"moonshot", // 厂商别名,不是平台标识
		"Kimi",     // 大小写敏感
		"openai ",  // 尾随空格
		"glm",
		"bogus",
	}
	for _, platform := range invalid {
		t.Run("create_"+platform, func(t *testing.T) {
			var req CreateGroupRequest
			body := fmt.Sprintf(`{"name":"g","platform":%q}`, platform)
			require.Error(t, bindGroupPlatformJSON(t, &req, body),
				"platform %q 应被 CreateGroupRequest 拒绝", platform)
		})
		t.Run("update_"+platform, func(t *testing.T) {
			var req UpdateGroupRequest
			body := fmt.Sprintf(`{"platform":%q}`, platform)
			require.Error(t, bindGroupPlatformJSON(t, &req, body),
				"platform %q 应被 UpdateGroupRequest 拒绝", platform)
		})
	}
}
