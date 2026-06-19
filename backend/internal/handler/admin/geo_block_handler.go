package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// GetGeoBlockSettings 获取地区拦截（合规）运行时配置。
// GET /api/v1/admin/settings/geo-block
func (h *SettingHandler) GetGeoBlockSettings(c *gin.Context) {
	rt := h.settingService.GetGeoBlockRuntime(c.Request.Context())
	response.Success(c, geoBlockSettingsToDTO(rt))
}

// UpdateGeoBlockSettings 更新地区拦截（合规）运行时配置，保存后即时生效。
// PUT /api/v1/admin/settings/geo-block
func (h *SettingHandler) UpdateGeoBlockSettings(c *gin.Context) {
	var req dto.GeoBlockSettings
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	if err := h.settingService.UpdateGeoBlockRuntime(c.Request.Context(), config.GeoBlockConfig{
		Enabled:   req.Enabled,
		Countries: req.Countries,
		Whitelist: req.Whitelist,
	}); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	rt := h.settingService.GetGeoBlockRuntime(c.Request.Context())
	response.Success(c, geoBlockSettingsToDTO(rt))
}

func geoBlockSettingsToDTO(rt config.GeoBlockConfig) dto.GeoBlockSettings {
	countries := rt.Countries
	if countries == nil {
		countries = []string{}
	}
	whitelist := rt.Whitelist
	if whitelist == nil {
		whitelist = []string{}
	}
	return dto.GeoBlockSettings{
		Enabled:   rt.Enabled,
		Countries: countries,
		Whitelist: whitelist,
	}
}
