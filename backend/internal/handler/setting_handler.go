package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// SettingHandler 公开设置处理器（无需认证）
type SettingHandler struct {
	settingService *service.SettingService
	version        string
}

// NewSettingHandler 创建公开设置处理器
func NewSettingHandler(settingService *service.SettingService, version string) *SettingHandler {
	return &SettingHandler{
		settingService: settingService,
		version:        version,
	}
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
		PurchaseSubscriptionEnabled:           settings.PurchaseSubscriptionEnabled,
		PurchaseSubscriptionURL:               settings.PurchaseSubscriptionURL,
		TableDefaultPageSize:                  settings.TableDefaultPageSize,
		TablePageSizeOptions:                  settings.TablePageSizeOptions,
		CustomMenuItems:                       dto.ParseUserVisibleMenuItems(settings.CustomMenuItems),
		CustomEndpoints:                       dto.ParseCustomEndpoints(settings.CustomEndpoints),
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
		BalanceLowNotifyEnabled:               settings.BalanceLowNotifyEnabled,
		AccountQuotaNotifyEnabled:             settings.AccountQuotaNotifyEnabled,
		BalanceLowNotifyThreshold:             settings.BalanceLowNotifyThreshold,
		BalanceLowNotifyRechargeURL:           settings.BalanceLowNotifyRechargeURL,

		ChannelMonitorEnabled:                settings.ChannelMonitorEnabled,
		ChannelMonitorDefaultIntervalSeconds: settings.ChannelMonitorDefaultIntervalSeconds,

		AvailableChannelsEnabled: settings.AvailableChannelsEnabled,

		AffiliateEnabled: settings.AffiliateEnabled,

		RiskControlEnabled:                settings.RiskControlEnabled,
		SiteMessagesEnabled:               settings.SiteMessagesEnabled,
		SiteMessagesDefaultRecipientEmail: settings.SiteMessagesDefaultRecipientEmail,
	})
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

// GetPublicModelPricing 获取首页公开模型定价
// GET /api/v1/pricing/public
func (h *SettingHandler) GetPublicModelPricing(c *gin.Context) {
	pricing, err := h.settingService.GetPublicModelPricing(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, pricing.EnabledRows())
}
