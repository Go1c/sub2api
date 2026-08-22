package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"html"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

const lumioDesktopConfigCacheControl = "public, max-age=300, stale-if-error=86400"

// SettingHandler 公开设置处理器（无需认证）
type SettingHandler struct {
	settingService           *service.SettingService
	notificationEmailService *service.NotificationEmailService
	version                  string
}

// NewSettingHandler 创建公开设置处理器
func NewSettingHandler(settingService *service.SettingService, version string) *SettingHandler {
	return &SettingHandler{
		settingService: settingService,
		version:        version,
	}
}

// SetNotificationEmailService attaches the public notification email service without
// changing the constructor signature used by existing tests.
func (h *SettingHandler) SetNotificationEmailService(notificationEmailService *service.NotificationEmailService) {
	h.notificationEmailService = notificationEmailService
}

// GetPublicSettings 获取公开设置
// GET /api/v1/settings/public
func (h *SettingHandler) GetPublicSettings(c *gin.Context) {
	settings, err := h.settingService.GetPublicSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.PublicSettings{
		RegistrationEnabled:                   settings.RegistrationEnabled,
		EmailVerifyEnabled:                    settings.EmailVerifyEnabled,
		ForceEmailOnThirdPartySignup:          settings.ForceEmailOnThirdPartySignup,
		RegistrationEmailSuffixWhitelist:      settings.RegistrationEmailSuffixWhitelist,
		PromoCodeEnabled:                      settings.PromoCodeEnabled,
		PasswordResetEnabled:                  settings.PasswordResetEnabled,
		InvitationCodeEnabled:                 settings.InvitationCodeEnabled,
		InvitationRegistrationMode:            settings.InvitationRegistrationMode,
		TotpEnabled:                           settings.TotpEnabled,
		LoginAgreementEnabled:                 settings.LoginAgreementEnabled,
		LoginAgreementMode:                    settings.LoginAgreementMode,
		LoginAgreementUpdatedAt:               settings.LoginAgreementUpdatedAt,
		LoginAgreementRevision:                settings.LoginAgreementRevision,
		LoginAgreementDocuments:               publicLoginAgreementDocumentsToDTO(settings.LoginAgreementDocuments),
		TurnstileEnabled:                      settings.TurnstileEnabled,
		TurnstileSiteKey:                      settings.TurnstileSiteKey,
		SiteName:                              settings.SiteName,
		SiteLogo:                              settings.SiteLogo,
		SiteSubtitle:                          settings.SiteSubtitle,
		APIBaseURL:                            settings.APIBaseURL,
		ContactInfo:                           settings.ContactInfo,
		ContactChannels:                       dto.ParseContactChannels(settings.ContactChannels),
		SupportChatEnabled:                    settings.SupportChatEnabled,
		SupportChatGatewayURL:                 settings.SupportChatGatewayURL,
		SupportChatTitle:                      settings.SupportChatTitle,
		SupportChatWelcomeMessage:             settings.SupportChatWelcomeMessage,
		SupportChatOfficialContactText:        settings.SupportChatOfficialContactText,
		SupportChatOfficialContactURL:         settings.SupportChatOfficialContactURL,
		DocURL:                                settings.DocURL,
		SitePages:                             dto.ParseSitePages(settings.SitePages),
		HomeContent:                           settings.HomeContent,
		HideCcsImportButton:                   settings.HideCcsImportButton,
		FrontendLocales:                       settings.FrontendLocales,
		CCSwitchDefaultModelAnthropic:         settings.CCSwitchDefaultModelAnthropic,
		CCSwitchDefaultModelOpenAI:            settings.CCSwitchDefaultModelOpenAI,
		CCSwitchDefaultModelGemini:            settings.CCSwitchDefaultModelGemini,
		CCSwitchDefaultModelAntigravity:       settings.CCSwitchDefaultModelAntigravity,
		CCSwitchDefaultModelAntigravityGemini: settings.CCSwitchDefaultModelAntigravityGemini,
		UserSubscriptionsVisible:              settings.UserSubscriptionsVisible,
		PurchaseSubscriptionEnabled:           settings.PurchaseSubscriptionEnabled,
		PurchaseSubscriptionURL:               settings.PurchaseSubscriptionURL,
		TableDefaultPageSize:                  settings.TableDefaultPageSize,
		TablePageSizeOptions:                  settings.TablePageSizeOptions,
		CustomMenuItems:                       dto.ParseUserVisibleMenuItems(settings.CustomMenuItems),
		CustomEndpoints:                       dto.ParseCustomEndpoints(settings.CustomEndpoints),
		DingTalkOAuthEnabled:                  settings.DingTalkOAuthEnabled,
		LinuxDoOAuthEnabled:                   settings.LinuxDoOAuthEnabled,
		WeChatOAuthEnabled:                    settings.WeChatOAuthEnabled,
		WeChatOAuthOpenEnabled:                settings.WeChatOAuthOpenEnabled,
		WeChatOAuthMPEnabled:                  settings.WeChatOAuthMPEnabled,
		WeChatOAuthMobileEnabled:              settings.WeChatOAuthMobileEnabled,
		OIDCOAuthEnabled:                      settings.OIDCOAuthEnabled,
		OIDCOAuthProviderName:                 settings.OIDCOAuthProviderName,
		GitHubOAuthEnabled:                    settings.GitHubOAuthEnabled,
		GoogleOAuthEnabled:                    settings.GoogleOAuthEnabled,
		BackendModeEnabled:                    settings.BackendModeEnabled,
		PaymentEnabled:                        settings.PaymentEnabled,
		Version:                               h.version,
		ServerTimezone:                        timezone.Name(),
		ServerUTCOffset:                       timezone.UTCOffset(),
		BalanceLowNotifyEnabled:               settings.BalanceLowNotifyEnabled,
		AccountQuotaNotifyEnabled:             settings.AccountQuotaNotifyEnabled,
		BalanceLowNotifyThreshold:             settings.BalanceLowNotifyThreshold,
		BalanceLowNotifyRechargeURL:           settings.BalanceLowNotifyRechargeURL,

		ChannelMonitorEnabled:                settings.ChannelMonitorEnabled,
		ChannelMonitorDefaultIntervalSeconds: settings.ChannelMonitorDefaultIntervalSeconds,
		ChannelMonitorStatusBanner:           settings.ChannelMonitorStatusBanner,
		ChannelMonitorShowQuota:              settings.ChannelMonitorShowQuota,

		AvailableChannelsEnabled: settings.AvailableChannelsEnabled,

		AffiliateEnabled: settings.AffiliateEnabled,

		RiskControlEnabled:                settings.RiskControlEnabled,
		SiteMessagesEnabled:               settings.SiteMessagesEnabled,
		SiteMessagesDefaultRecipientEmail: settings.SiteMessagesDefaultRecipientEmail,

		AllowUserViewErrorRequests: settings.AllowUserViewErrorRequests,
	})
}

// GetLumioDesktopConfig returns the public, whitelisted Lumio desktop bootstrap document.
// GET /api/v1/desktop/config
func (h *SettingHandler) GetLumioDesktopConfig(c *gin.Context) {
	config, err := h.settingService.GetLumioDesktopConfig(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	payload, err := json.Marshal(config)
	if err != nil {
		response.InternalError(c, "Failed to serialize desktop configuration")
		return
	}
	digest := sha256.Sum256(payload)
	etag := `"` + hex.EncodeToString(digest[:]) + `"`
	c.Header("Cache-Control", lumioDesktopConfigCacheControl)
	c.Header("ETag", etag)
	if strings.TrimSpace(c.GetHeader("If-None-Match")) == etag {
		c.Status(http.StatusNotModified)
		return
	}
	response.Success(c, config)
}

// UnsubscribeNotificationEmail handles optional notification email opt-outs.
// GET /api/v1/settings/email-unsubscribe?token=...
func (h *SettingHandler) UnsubscribeNotificationEmail(c *gin.Context) {
	if h.notificationEmailService == nil {
		response.InternalError(c, "notification email service is not configured")
		return
	}
	token := strings.TrimSpace(c.Query("token"))
	if token == "" {
		response.BadRequest(c, "token is required")
		return
	}
	result, err := h.notificationEmailService.Unsubscribe(c.Request.Context(), token)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	body := "<!doctype html><html><head><meta charset=\"utf-8\"><title>Unsubscribed</title></head><body style=\"font-family:-apple-system,BlinkMacSystemFont,Segoe UI,sans-serif;padding:32px;\"><h1>Unsubscribed</h1><p>You have unsubscribed <strong>" + html.EscapeString(result.Email) + "</strong> from <strong>" + html.EscapeString(result.Event) + "</strong> emails.</p></body></html>"
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(body))
}

func publicLoginAgreementDocumentsToDTO(items []service.LoginAgreementDocument) []dto.LoginAgreementDocument {
	result := make([]dto.LoginAgreementDocument, 0, len(items))
	for _, item := range items {
		result = append(result, dto.LoginAgreementDocument{
			ID:        item.ID,
			Title:     item.Title,
			ContentMD: item.ContentMD,
		})
	}
	return result
}
