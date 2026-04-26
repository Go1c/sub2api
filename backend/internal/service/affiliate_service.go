package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

var (
	ErrAffiliateProfileNotFound = infraerrors.NotFound("AFFILIATE_PROFILE_NOT_FOUND", "affiliate profile not found")
	ErrAffiliateCodeInvalid     = infraerrors.BadRequest("AFFILIATE_CODE_INVALID", "invalid affiliate code")
	ErrAffiliateCodeTaken       = infraerrors.Conflict("AFFILIATE_CODE_TAKEN", "affiliate code already in use")
	ErrAffiliateAlreadyBound    = infraerrors.Conflict("AFFILIATE_ALREADY_BOUND", "affiliate inviter already bound")
	ErrAffiliateQuotaEmpty      = infraerrors.BadRequest("AFFILIATE_QUOTA_EMPTY", "no affiliate quota available to transfer")
)

const (
	affiliateInviteesLimit = 100
	// AffiliateCodeMinLength / AffiliateCodeMaxLength bound both system-generated
	// 12-char codes and admin-customized codes (e.g. "VIP2026").
	AffiliateCodeMinLength = 4
	AffiliateCodeMaxLength = 32
)

// affiliateCodeValidChar accepts uppercase letters, digits, underscore and dash.
// All input passes through strings.ToUpper before validation, so lowercase from
// users is normalized — admins may supply mixed case in their UI.
var affiliateCodeValidChar = func() [256]bool {
	var tbl [256]bool
	for c := byte('A'); c <= 'Z'; c++ {
		tbl[c] = true
	}
	for c := byte('0'); c <= '9'; c++ {
		tbl[c] = true
	}
	tbl['_'] = true
	tbl['-'] = true
	return tbl
}()

// isValidAffiliateCodeFormat validates code format for both binding (user input)
// and admin updates. Caller is expected to upper-case the input first.
func isValidAffiliateCodeFormat(code string) bool {
	if len(code) < AffiliateCodeMinLength || len(code) > AffiliateCodeMaxLength {
		return false
	}
	for i := 0; i < len(code); i++ {
		if !affiliateCodeValidChar[code[i]] {
			return false
		}
	}
	return true
}

type AffiliateSummary struct {
	UserID               int64     `json:"user_id"`
	AffCode              string    `json:"aff_code"`
	AffCodeCustom        bool      `json:"aff_code_custom"`
	AffRebateRatePercent *float64  `json:"aff_rebate_rate_percent,omitempty"`
	InviterID            *int64    `json:"inviter_id,omitempty"`
	AffCount             int       `json:"aff_count"`
	AffQuota             float64   `json:"aff_quota"`
	AffFrozenQuota       float64   `json:"aff_frozen_quota"`
	AffHistoryQuota      float64   `json:"aff_history_quota"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type AffiliateInvitee struct {
	UserID      int64      `json:"user_id"`
	Email       string     `json:"email"`
	Username    string     `json:"username"`
	CreatedAt   *time.Time `json:"created_at,omitempty"`
	TotalRebate float64    `json:"total_rebate"`
}

type AffiliateInviteLog struct {
	ID              int64     `json:"id"`
	InviterID       *int64    `json:"inviter_id,omitempty"`
	InviterEmail    string    `json:"inviter_email,omitempty"`
	InviterUsername string    `json:"inviter_username,omitempty"`
	InviteeID       *int64    `json:"invitee_id,omitempty"`
	InviteeEmail    string    `json:"invitee_email,omitempty"`
	InviteeUsername string    `json:"invitee_username,omitempty"`
	AffiliateCode   string    `json:"affiliate_code,omitempty"`
	Success         bool      `json:"success"`
	FailureReason   string    `json:"failure_reason,omitempty"`
	FailureMessage  string    `json:"failure_message,omitempty"`
	BonusAmount     float64   `json:"bonus_amount"`
	FingerprintHash string    `json:"fingerprint_hash,omitempty"`
	IPAddress       string    `json:"ip_address,omitempty"`
	UserAgent       string    `json:"user_agent,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

type AffiliateInviteLogFilter struct {
	AccountID        int64
	InviterID        int64
	InviteeID        int64
	IncludeSensitive bool
	Page             int
	PageSize         int
}

type AffiliateDetail struct {
	UserID          int64   `json:"user_id"`
	AffCode         string  `json:"aff_code"`
	InviterID       *int64  `json:"inviter_id,omitempty"`
	AffCount        int     `json:"aff_count"`
	AffQuota        float64 `json:"aff_quota"`
	AffFrozenQuota  float64 `json:"aff_frozen_quota"`
	AffHistoryQuota float64 `json:"aff_history_quota"`
	// EffectiveRebateRatePercent 是当前用户作为邀请人时实际生效的返利比例：
	// 优先用户自己的专属比例（aff_rebate_rate_percent），否则回退到全局比例。
	// 用于在用户的 /affiliate 页面直观展示「分享后能拿到多少」。
	EffectiveRebateRatePercent float64            `json:"effective_rebate_rate_percent"`
	Invitees                   []AffiliateInvitee `json:"invitees"`
}

type AffiliateRepository interface {
	EnsureUserAffiliate(ctx context.Context, userID int64) (*AffiliateSummary, error)
	GetAffiliateByCode(ctx context.Context, code string) (*AffiliateSummary, error)
	BindInviter(ctx context.Context, userID, inviterID int64) (bool, error)
	BindInviterWithSignupBonus(ctx context.Context, req AffiliateSignupBonusRequest) (*AffiliateSignupBonusResult, error)
	RecordInviteLog(ctx context.Context, entry AffiliateInviteLogEntry) error
	ListInviteLogs(ctx context.Context, filter AffiliateInviteLogFilter) ([]AffiliateInviteLog, int64, error)
	AccrueQuota(ctx context.Context, inviterID, inviteeUserID int64, amount float64, freezeHours int) (bool, error)
	GetAccruedRebateFromInvitee(ctx context.Context, inviterID, inviteeUserID int64) (float64, error)
	ThawFrozenQuota(ctx context.Context, userID int64) (float64, error)
	TransferQuotaToBalance(ctx context.Context, userID int64) (float64, float64, error)
	ListInvitees(ctx context.Context, inviterID int64, limit int) ([]AffiliateInvitee, error)

	// 管理端：用户级专属配置
	UpdateUserAffCode(ctx context.Context, userID int64, newCode string) error
	ResetUserAffCode(ctx context.Context, userID int64) (string, error)
	SetUserRebateRate(ctx context.Context, userID int64, ratePercent *float64) error
	BatchSetUserRebateRate(ctx context.Context, userIDs []int64, ratePercent *float64) error
	ListUsersWithCustomSettings(ctx context.Context, filter AffiliateAdminFilter) ([]AffiliateAdminEntry, int64, error)
}

type AffiliateSignupBonusRequest struct {
	UserID          int64
	InviterID       int64
	AffiliateCode   string
	Amount          float64
	InviterTotalCap float64
	DailyTotalCap   float64
	FingerprintHash string
	IPAddress       string
	UserAgent       string
}

type AffiliateSignupBonusResult struct {
	Bound         bool
	AwardedAmount float64
	FailureReason string
}

type AffiliateInviteLogEntry struct {
	InviterID       *int64
	InviteeID       *int64
	AffiliateCode   string
	Success         bool
	FailureReason   string
	BonusAmount     float64
	FingerprintHash string
	IPAddress       string
	UserAgent       string
}

type AffiliateSignupRequestMeta struct {
	FingerprintHash string
	IPAddress       string
	UserAgent       string
}

type affiliateSignupMetaContextKey struct{}

func NewAffiliateSignupRequestMeta(rawFingerprint, ipAddress, userAgent string) AffiliateSignupRequestMeta {
	raw := strings.TrimSpace(rawFingerprint)
	if raw == "" {
		raw = strings.TrimSpace(ipAddress) + "|" + strings.TrimSpace(userAgent)
	}
	hash := ""
	if raw != "" {
		sum := sha256.Sum256([]byte(raw))
		hash = hex.EncodeToString(sum[:])
	}
	return AffiliateSignupRequestMeta{
		FingerprintHash: hash,
		IPAddress:       truncateString(strings.TrimSpace(ipAddress), 64),
		UserAgent:       truncateString(strings.TrimSpace(userAgent), 1024),
	}
}

func ContextWithAffiliateSignupRequestMeta(ctx context.Context, meta AffiliateSignupRequestMeta) context.Context {
	return context.WithValue(ctx, affiliateSignupMetaContextKey{}, meta)
}

func affiliateSignupRequestMetaFromContext(ctx context.Context) AffiliateSignupRequestMeta {
	if ctx == nil {
		return AffiliateSignupRequestMeta{}
	}
	if meta, ok := ctx.Value(affiliateSignupMetaContextKey{}).(AffiliateSignupRequestMeta); ok {
		return meta
	}
	return AffiliateSignupRequestMeta{}
}

// AffiliateAdminFilter 列表筛选条件
type AffiliateAdminFilter struct {
	Search   string
	Page     int
	PageSize int
}

// AffiliateAdminEntry 专属用户列表条目
type AffiliateAdminEntry struct {
	UserID               int64    `json:"user_id"`
	Email                string   `json:"email"`
	Username             string   `json:"username"`
	AffCode              string   `json:"aff_code"`
	AffCodeCustom        bool     `json:"aff_code_custom"`
	AffRebateRatePercent *float64 `json:"aff_rebate_rate_percent,omitempty"`
	AffCount             int      `json:"aff_count"`
}

type AffiliateService struct {
	repo                 AffiliateRepository
	settingService       *SettingService
	authCacheInvalidator APIKeyAuthCacheInvalidator
	billingCacheService  *BillingCacheService
}

func NewAffiliateService(repo AffiliateRepository, settingService *SettingService, authCacheInvalidator APIKeyAuthCacheInvalidator, billingCacheService *BillingCacheService) *AffiliateService {
	return &AffiliateService{
		repo:                 repo,
		settingService:       settingService,
		authCacheInvalidator: authCacheInvalidator,
		billingCacheService:  billingCacheService,
	}
}

// IsEnabled reports whether the affiliate (邀请返利) feature is turned on.
func (s *AffiliateService) IsEnabled(ctx context.Context) bool {
	if s == nil || s.settingService == nil {
		return AffiliateEnabledDefault
	}
	return s.settingService.IsAffiliateEnabled(ctx)
}

func (s *AffiliateService) EnsureUserAffiliate(ctx context.Context, userID int64) (*AffiliateSummary, error) {
	if userID <= 0 {
		return nil, infraerrors.BadRequest("INVALID_USER", "invalid user")
	}
	if s == nil || s.repo == nil {
		return nil, infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "affiliate service unavailable")
	}
	return s.repo.EnsureUserAffiliate(ctx, userID)
}

func (s *AffiliateService) GetAffiliateDetail(ctx context.Context, userID int64) (*AffiliateDetail, error) {
	// Lazy thaw: move any matured frozen quota to available before reading.
	if s != nil && s.repo != nil {
		// best-effort: thaw failure is non-fatal
		_, _ = s.repo.ThawFrozenQuota(ctx, userID)
	}

	summary, err := s.EnsureUserAffiliate(ctx, userID)
	if err != nil {
		return nil, err
	}
	invitees, err := s.listInvitees(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &AffiliateDetail{
		UserID:                     summary.UserID,
		AffCode:                    summary.AffCode,
		InviterID:                  summary.InviterID,
		AffCount:                   summary.AffCount,
		AffQuota:                   summary.AffQuota,
		AffFrozenQuota:             summary.AffFrozenQuota,
		AffHistoryQuota:            summary.AffHistoryQuota,
		EffectiveRebateRatePercent: s.resolveRebateRatePercent(ctx, summary),
		Invitees:                   invitees,
	}, nil
}

func (s *AffiliateService) BindInviterByCode(ctx context.Context, userID int64, rawCode string) error {
	code := strings.ToUpper(strings.TrimSpace(rawCode))
	if code == "" {
		return nil
	}
	if s == nil || s.repo == nil {
		return infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "affiliate service unavailable")
	}
	meta := affiliateSignupRequestMetaFromContext(ctx)
	inviteeID := userID
	// 总开关关闭时，注册阶段静默忽略 aff 参数（不报错，避免阻断注册流程）
	if !s.IsEnabled(ctx) {
		s.recordInviteLog(ctx, AffiliateInviteLogEntry{
			InviteeID:       &inviteeID,
			AffiliateCode:   code,
			Success:         false,
			FailureReason:   "affiliate_disabled",
			FingerprintHash: meta.FingerprintHash,
			IPAddress:       meta.IPAddress,
			UserAgent:       meta.UserAgent,
		})
		return nil
	}
	if !isValidAffiliateCodeFormat(code) {
		s.recordInviteLog(ctx, AffiliateInviteLogEntry{
			InviteeID:       &inviteeID,
			AffiliateCode:   code,
			Success:         false,
			FailureReason:   "invalid_code",
			FingerprintHash: meta.FingerprintHash,
			IPAddress:       meta.IPAddress,
			UserAgent:       meta.UserAgent,
		})
		return ErrAffiliateCodeInvalid
	}

	selfSummary, err := s.repo.EnsureUserAffiliate(ctx, userID)
	if err != nil {
		return err
	}
	if selfSummary.InviterID != nil {
		s.recordInviteLog(ctx, AffiliateInviteLogEntry{
			InviterID:       selfSummary.InviterID,
			InviteeID:       &inviteeID,
			AffiliateCode:   code,
			Success:         false,
			FailureReason:   "already_bound",
			FingerprintHash: meta.FingerprintHash,
			IPAddress:       meta.IPAddress,
			UserAgent:       meta.UserAgent,
		})
		return nil
	}

	inviterSummary, err := s.repo.GetAffiliateByCode(ctx, code)
	if err != nil {
		if errors.Is(err, ErrAffiliateProfileNotFound) {
			s.recordInviteLog(ctx, AffiliateInviteLogEntry{
				InviteeID:       &inviteeID,
				AffiliateCode:   code,
				Success:         false,
				FailureReason:   "invalid_code",
				FingerprintHash: meta.FingerprintHash,
				IPAddress:       meta.IPAddress,
				UserAgent:       meta.UserAgent,
			})
			return ErrAffiliateCodeInvalid
		}
		return err
	}
	if inviterSummary == nil || inviterSummary.UserID <= 0 || inviterSummary.UserID == userID {
		reason := "invalid_code"
		if inviterSummary != nil && inviterSummary.UserID == userID {
			reason = "self_invite"
		}
		s.recordInviteLog(ctx, AffiliateInviteLogEntry{
			InviteeID:       &inviteeID,
			AffiliateCode:   code,
			Success:         false,
			FailureReason:   reason,
			FingerprintHash: meta.FingerprintHash,
			IPAddress:       meta.IPAddress,
			UserAgent:       meta.UserAgent,
		})
		return ErrAffiliateCodeInvalid
	}

	bonusAmount, inviterTotalCap, dailyTotalCap := s.resolveSignupBonusSettings(ctx)
	result, err := s.repo.BindInviterWithSignupBonus(ctx, AffiliateSignupBonusRequest{
		UserID:          userID,
		InviterID:       inviterSummary.UserID,
		AffiliateCode:   code,
		Amount:          bonusAmount,
		InviterTotalCap: inviterTotalCap,
		DailyTotalCap:   dailyTotalCap,
		FingerprintHash: meta.FingerprintHash,
		IPAddress:       meta.IPAddress,
		UserAgent:       meta.UserAgent,
	})
	if err != nil {
		return err
	}
	if result == nil || !result.Bound {
		return ErrAffiliateAlreadyBound
	}
	return nil
}

func (s *AffiliateService) resolveSignupBonusSettings(ctx context.Context) (amount, inviterTotalCap, dailyTotalCap float64) {
	if s == nil || s.settingService == nil || !s.settingService.IsAffiliateSignupBonusEnabled(ctx) {
		return 0, 0, 0
	}
	amount = roundTo(s.settingService.GetAffiliateSignupBonusAmount(ctx), 8)
	if amount <= 0 {
		return 0, 0, 0
	}
	inviterTotalCap = roundTo(s.settingService.GetAffiliateSignupBonusTotalCap(ctx), 8)
	dailyTotalCap = roundTo(s.settingService.GetAffiliateSignupBonusDailyCap(ctx), 8)
	return amount, inviterTotalCap, dailyTotalCap
}

func (s *AffiliateService) recordInviteLog(ctx context.Context, entry AffiliateInviteLogEntry) {
	if s == nil || s.repo == nil {
		return
	}
	if err := s.repo.RecordInviteLog(ctx, entry); err != nil {
		logger.LegacyPrintf("service.affiliate", "[Affiliate] Failed to record invite log: %v", err)
	}
}

func (s *AffiliateService) AccrueInviteRebate(ctx context.Context, inviteeUserID int64, baseRechargeAmount float64) (float64, error) {
	if s == nil || s.repo == nil {
		return 0, nil
	}
	if inviteeUserID <= 0 || baseRechargeAmount <= 0 || math.IsNaN(baseRechargeAmount) || math.IsInf(baseRechargeAmount, 0) {
		return 0, nil
	}
	// 总开关关闭时，新充值不再产生返利
	if !s.IsEnabled(ctx) {
		return 0, nil
	}

	inviteeSummary, err := s.repo.EnsureUserAffiliate(ctx, inviteeUserID)
	if err != nil {
		return 0, err
	}
	if inviteeSummary.InviterID == nil || *inviteeSummary.InviterID <= 0 {
		return 0, nil
	}

	// 加载邀请人 profile，优先使用专属比例（覆盖全局）
	inviterSummary, err := s.repo.EnsureUserAffiliate(ctx, *inviteeSummary.InviterID)
	if err != nil {
		return 0, err
	}
	// 有效期检查：超过返利有效期后不再产生返利
	if s.settingService != nil {
		if durationDays := s.settingService.GetAffiliateRebateDurationDays(ctx); durationDays > 0 {
			if time.Now().After(inviteeSummary.CreatedAt.AddDate(0, 0, durationDays)) {
				return 0, nil
			}
		}
	}

	rebateRatePercent := s.resolveRebateRatePercent(ctx, inviterSummary)
	rebate := roundTo(baseRechargeAmount*(rebateRatePercent/100), 8)
	if rebate <= 0 {
		return 0, nil
	}

	// 单人上限检查：精确截断到剩余额度
	if s.settingService != nil {
		if perInviteeCap := s.settingService.GetAffiliateRebatePerInviteeCap(ctx); perInviteeCap > 0 {
			existing, err := s.repo.GetAccruedRebateFromInvitee(ctx, *inviteeSummary.InviterID, inviteeUserID)
			if err != nil {
				return 0, err
			}
			if existing >= perInviteeCap {
				return 0, nil
			}
			if remaining := perInviteeCap - existing; rebate > remaining {
				rebate = roundTo(remaining, 8)
			}
		}
	}

	var freezeHours int
	if s.settingService != nil {
		freezeHours = s.settingService.GetAffiliateRebateFreezeHours(ctx)
	}

	applied, err := s.repo.AccrueQuota(ctx, *inviteeSummary.InviterID, inviteeUserID, rebate, freezeHours)
	if err != nil {
		return 0, err
	}
	if !applied {
		return 0, nil
	}
	return rebate, nil
}

// resolveRebateRatePercent returns the inviter's exclusive rate when set,
// otherwise the global setting value (clamped to [Min, Max]).
func (s *AffiliateService) resolveRebateRatePercent(ctx context.Context, inviter *AffiliateSummary) float64 {
	if inviter != nil && inviter.AffRebateRatePercent != nil {
		v := *inviter.AffRebateRatePercent
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return s.globalRebateRatePercent(ctx)
		}
		return clampAffiliateRebateRate(v)
	}
	return s.globalRebateRatePercent(ctx)
}

// globalRebateRatePercent reads the system-wide rebate rate via SettingService,
// returning the documented default when SettingService is unavailable.
func (s *AffiliateService) globalRebateRatePercent(ctx context.Context) float64 {
	if s == nil || s.settingService == nil {
		return AffiliateRebateRateDefault
	}
	return s.settingService.GetAffiliateRebateRatePercent(ctx)
}

func (s *AffiliateService) TransferAffiliateQuota(ctx context.Context, userID int64) (float64, float64, error) {
	if s == nil || s.repo == nil {
		return 0, 0, infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "affiliate service unavailable")
	}

	transferred, balance, err := s.repo.TransferQuotaToBalance(ctx, userID)
	if err != nil {
		return 0, 0, err
	}
	if transferred > 0 {
		s.invalidateAffiliateCaches(ctx, userID)
	}
	return transferred, balance, nil
}

func (s *AffiliateService) ListInviteLogs(ctx context.Context, accountID int64, page, pageSize int) ([]AffiliateInviteLog, int64, error) {
	if s == nil || s.repo == nil {
		return nil, 0, infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "affiliate service unavailable")
	}
	if accountID <= 0 {
		return nil, 0, infraerrors.BadRequest("INVALID_USER", "invalid user")
	}
	items, total, err := s.repo.ListInviteLogs(ctx, AffiliateInviteLogFilter{
		AccountID:        accountID,
		IncludeSensitive: false,
		Page:             page,
		PageSize:         pageSize,
	})
	if err != nil {
		return nil, 0, err
	}
	for i := range items {
		items[i].FailureMessage = affiliateInviteFailureMessage(items[i].FailureReason)
		items[i] = scrubAffiliateInviteLogForUser(items[i])
	}
	return items, total, nil
}

func (s *AffiliateService) AdminListInviteLogs(ctx context.Context, filter AffiliateInviteLogFilter) ([]AffiliateInviteLog, int64, error) {
	if s == nil || s.repo == nil {
		return nil, 0, infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "affiliate service unavailable")
	}
	filter.IncludeSensitive = true
	items, total, err := s.repo.ListInviteLogs(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	for i := range items {
		items[i].FailureMessage = affiliateInviteFailureMessage(items[i].FailureReason)
	}
	return items, total, nil
}

func scrubAffiliateInviteLogForUser(item AffiliateInviteLog) AffiliateInviteLog {
	item.FingerprintHash = ""
	item.IPAddress = ""
	item.UserAgent = ""
	item.InviterEmail = maskEmail(item.InviterEmail)
	item.InviteeEmail = maskEmail(item.InviteeEmail)
	return item
}

func affiliateInviteFailureMessage(reason string) string {
	switch strings.TrimSpace(reason) {
	case "":
		return ""
	case "affiliate_disabled":
		return "邀请返利功能未开启，未绑定邀请关系。"
	case "invalid_code":
		return "邀请码无效，未绑定邀请关系。"
	case "already_bound":
		return "该账号已经绑定过邀请人，不能重复绑定。"
	case "self_invite":
		return "不能使用自己的邀请码。"
	case "fingerprint_reused":
		return "该设备指纹已经领取过注册奖励，本次只绑定邀请关系，不再赠送余额。"
	case "inviter_total_cap_reached":
		return "邀请人的注册赠送累计上限已达到，本次不再赠送余额。"
	case "daily_total_cap_reached":
		return "全站今日注册赠送额度已达到上限，本次不再赠送余额，明日自动恢复。"
	case "cap_reached":
		return "注册赠送额度上限已达到，本次不再赠送余额。"
	default:
		return "邀请处理未完成，请联系管理员查看原因。"
	}
}

func (s *AffiliateService) listInvitees(ctx context.Context, inviterID int64) ([]AffiliateInvitee, error) {
	if s == nil || s.repo == nil {
		return nil, infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "affiliate service unavailable")
	}
	invitees, err := s.repo.ListInvitees(ctx, inviterID, affiliateInviteesLimit)
	if err != nil {
		return nil, err
	}
	for i := range invitees {
		invitees[i].Email = maskEmail(invitees[i].Email)
	}
	return invitees, nil
}

func roundTo(v float64, scale int) float64 {
	factor := math.Pow10(scale)
	return math.Round(v*factor) / factor
}

func maskEmail(email string) string {
	email = strings.TrimSpace(email)
	if email == "" {
		return ""
	}
	at := strings.Index(email, "@")
	if at <= 0 || at >= len(email)-1 {
		return "***"
	}

	local := email[:at]
	domain := email[at+1:]
	dot := strings.LastIndex(domain, ".")

	maskedLocal := maskSegment(local)
	if dot <= 0 || dot >= len(domain)-1 {
		return maskedLocal + "@" + maskSegment(domain)
	}

	domainName := domain[:dot]
	tld := domain[dot:]
	return maskedLocal + "@" + maskSegment(domainName) + tld
}

func maskSegment(s string) string {
	r := []rune(s)
	if len(r) == 0 {
		return "***"
	}
	if len(r) == 1 {
		return string(r[0]) + "***"
	}
	return string(r[0]) + "***"
}

func (s *AffiliateService) invalidateAffiliateCaches(ctx context.Context, userID int64) {
	if s.authCacheInvalidator != nil {
		s.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, userID)
	}
	if s.billingCacheService != nil {
		if err := s.billingCacheService.InvalidateUserBalance(ctx, userID); err != nil {
			logger.LegacyPrintf("service.affiliate", "[Affiliate] Failed to invalidate billing cache for user %d: %v", userID, err)
		}
	}
}

// =========================
// Admin: 专属配置管理
// =========================

// validateExclusiveRate ensures a per-user override is finite and within
// [Min, Max]. nil is always valid (means "clear / fall back to global").
func validateExclusiveRate(ratePercent *float64) error {
	if ratePercent == nil {
		return nil
	}
	v := *ratePercent
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return infraerrors.BadRequest("INVALID_RATE", "invalid rebate rate")
	}
	if v < AffiliateRebateRateMin || v > AffiliateRebateRateMax {
		return infraerrors.BadRequest("INVALID_RATE", "rebate rate out of range")
	}
	return nil
}

// AdminUpdateUserAffCode 管理员改写用户的邀请码（专属邀请码）。
func (s *AffiliateService) AdminUpdateUserAffCode(ctx context.Context, userID int64, rawCode string) error {
	if s == nil || s.repo == nil {
		return infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "affiliate service unavailable")
	}
	code := strings.ToUpper(strings.TrimSpace(rawCode))
	if !isValidAffiliateCodeFormat(code) {
		return ErrAffiliateCodeInvalid
	}
	return s.repo.UpdateUserAffCode(ctx, userID, code)
}

// AdminResetUserAffCode 重置用户邀请码为系统随机码。
func (s *AffiliateService) AdminResetUserAffCode(ctx context.Context, userID int64) (string, error) {
	if s == nil || s.repo == nil {
		return "", infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "affiliate service unavailable")
	}
	return s.repo.ResetUserAffCode(ctx, userID)
}

// AdminSetUserRebateRate 设置/清除用户专属返利比例。ratePercent==nil 表示清除。
func (s *AffiliateService) AdminSetUserRebateRate(ctx context.Context, userID int64, ratePercent *float64) error {
	if s == nil || s.repo == nil {
		return infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "affiliate service unavailable")
	}
	if err := validateExclusiveRate(ratePercent); err != nil {
		return err
	}
	return s.repo.SetUserRebateRate(ctx, userID, ratePercent)
}

// AdminBatchSetUserRebateRate 批量设置/清除用户专属返利比例。
func (s *AffiliateService) AdminBatchSetUserRebateRate(ctx context.Context, userIDs []int64, ratePercent *float64) error {
	if s == nil || s.repo == nil {
		return infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "affiliate service unavailable")
	}
	if err := validateExclusiveRate(ratePercent); err != nil {
		return err
	}
	cleaned := make([]int64, 0, len(userIDs))
	for _, uid := range userIDs {
		if uid > 0 {
			cleaned = append(cleaned, uid)
		}
	}
	if len(cleaned) == 0 {
		return nil
	}
	return s.repo.BatchSetUserRebateRate(ctx, cleaned, ratePercent)
}

// AdminListCustomUsers 列出有专属配置的用户。
func (s *AffiliateService) AdminListCustomUsers(ctx context.Context, filter AffiliateAdminFilter) ([]AffiliateAdminEntry, int64, error) {
	if s == nil || s.repo == nil {
		return nil, 0, infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "affiliate service unavailable")
	}
	return s.repo.ListUsersWithCustomSettings(ctx, filter)
}
