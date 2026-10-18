package service

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
)

const subscriptionDayDuration = 24 * time.Hour

const subscriptionDayDuration = 24 * time.Hour

// UserSubscription 表示用户订阅。
//
// 订阅额度池模型（详见 doc/plan/2026-05-28-subscription-credit-pool-plan.md）：
//
// 可消费    : status='active' AND ExhaustedAt == nil AND ExpiresAt > now
// 等过期    : status='active' AND ExhaustedAt != nil（总额度已耗尽，等待 ExpiresAt 到达）
// 已过期    : status='expired'
// 暂停      : status='suspended'
//
// 月度窗口字段（MonthlyWindowStart / MonthlyUsageUSD）已从数据库移除，
// 但 Go struct 暂时保留为内存字段以兼容现有代码；Task 4 后将完全清理。
type UserSubscription struct {
	ID      int64
	UserID  int64
	GroupID *int64 // 额度池订阅可空（不绑定具体分组）

	// 额度池字段（新模型）
	PlanID          *int64
	PlanName        string
	PlanProductName string
	ScopeType       string
	ScopeConfig     map[string]any
	QuotaLimitUSD   float64
	QuotaUsedUSD    float64
	DailyLimitUSD   *float64
	WeeklyLimitUSD  *float64

	// 生命周期
	StartsAt  time.Time
	ExpiresAt time.Time
	Status    string

	// 耗尽时间戳（总额度耗尽时刻；nil 表示订阅当前仍可消费）
	ExhaustedAt           *time.Time
	ExpiredCreditLoggedAt *time.Time

	// 窗口状态
	DailyWindowStart   *time.Time
	WeeklyWindowStart  *time.Time
	MonthlyWindowStart *time.Time // 兼容字段；DB 已移除，Task 4 将彻底清理

	DailyUsageUSD   float64
	WeeklyUsageUSD  float64
	MonthlyUsageUSD float64 // 兼容字段；DB 已移除

	// WeeklyLimitUserResetAt 用户手动重置周限时刻。
	// nil = 本订阅周期内尚未手动重置；非 nil = 已使用过一次机会。
	WeeklyLimitUserResetAt *time.Time

	// 管理后台展示字段：近 30 天窗口重置和过期销毁造成的浪费金额。
	Recent30dWastedUSD float64

	AssignedBy *int64
	AssignedAt time.Time
	Notes      string

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time

	User           *User
	Group          *Group
	AssignedByUser *User
}

// IsUsable 当前订阅是否可消费（新模型的可消费判定）。
func (s *UserSubscription) IsUsable() bool {
	if s == nil {
		return false
	}
	return s.Status == SubscriptionStatusActive &&
		s.ExhaustedAt == nil &&
		time.Now().Before(s.ExpiresAt)
}

func (s *UserSubscription) IsActive() bool {
	return s.Status == SubscriptionStatusActive && time.Now().Before(s.ExpiresAt)
}

func (s *UserSubscription) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

func (s *UserSubscription) IsExhausted() bool {
	return s.ExhaustedAt != nil
}

// IsCreditPoolExhausted 总额度池是否已耗尽（订阅应作废、回落到余额闸门）。
//
// 仅在以下情况判为耗尽：
//   - ExhaustedAt 已显式置位；或
//   - 配置了正的总额度池（QuotaLimitUSD > 0）且剩余 <= 0。
//
// QuotaLimitUSD == 0 表示未配置总池（纯日/周窗口订阅），剩余额度恒为 0，
// 不应据此误判为耗尽——否则这类订阅会被错误地绕过、回落到余额闸门。
func (s *UserSubscription) IsCreditPoolExhausted() bool {
	if s.IsExhausted() {
		return true
	}
	return s.QuotaLimitUSD > 0 && s.QuotaRemainingUSD() <= 0
}

func (s *UserSubscription) DaysRemaining() int {
	return s.daysRemainingAt(time.Now())
}

func (s *UserSubscription) daysRemainingAt(now time.Time) int {
	remaining := s.ExpiresAt.Sub(now)
	if remaining <= 0 {
		return 0
	}

	days := int(remaining / subscriptionDayDuration)
	if remaining%subscriptionDayDuration != 0 {
		days++
	}
	return days
}

// QuotaRemainingUSD 剩余总额度（不考虑窗口）。
func (s *UserSubscription) QuotaRemainingUSD() float64 {
	rem := s.QuotaLimitUSD - s.QuotaUsedUSD
	if rem < 0 {
		return 0
	}
	return rem
}

func (s *UserSubscription) IsWindowActivated() bool {
	return s.DailyWindowStart != nil || s.WeeklyWindowStart != nil || s.MonthlyWindowStart != nil
}

func (s *UserSubscription) HasOneTimeDailyQuota() bool {
	if s == nil || s.StartsAt.IsZero() || s.ExpiresAt.IsZero() {
		return false
	}
	return !s.ExpiresAt.After(s.StartsAt.AddDate(0, 0, 1))
}

func (s *UserSubscription) NeedsDailyReset() bool {
	return s.NeedsDailyResetAt(time.Now())
}

func (s *UserSubscription) NeedsDailyResetAt(now time.Time) bool {
	_, ok := s.automaticDailyWindowStartAt(now)
	return ok
}

func (s *UserSubscription) NeedsWeeklyReset() bool {
	return s.NeedsWeeklyResetAt(time.Now())
}

func (s *UserSubscription) NeedsWeeklyResetAt(now time.Time) bool {
	if s.WeeklyWindowStart == nil {
		return false
	}
	return !now.Before(s.WeeklyWindowStart.Add(7 * 24 * time.Hour))
}

func (s *UserSubscription) NeedsMonthlyReset() bool {
	return s.NeedsMonthlyResetAt(time.Now())
}

func (s *UserSubscription) NeedsMonthlyResetAt(now time.Time) bool {
	if s.MonthlyWindowStart == nil {
		return false
	}
	return !now.Before(s.MonthlyWindowStart.Add(30 * 24 * time.Hour))
}

func (s *UserSubscription) canAutomaticallyResetDailyAt(now time.Time) bool {
	_, ok := s.automaticDailyWindowStartAt(now)
	return ok
}

// automaticDailyWindowStartAt 计算日窗口按“配置时区日历日”对齐后的当前窗口起点。
// 日额度固定在每天 0 点刷新（与周/月的期限对齐滚动窗口语义不同），因此只要持久化
// 的窗口起点落在更早的日历日，就允许推进到今天 0 点。手动重置、激活等写入的任何
// 非 0 点锚点都会在下一个 0 点被拉回日历日边界，不会永久漂移刷新时刻。
func (s *UserSubscription) automaticDailyWindowStartAt(now time.Time) (time.Time, bool) {
	if s.DailyWindowStart == nil {
		return time.Time{}, false
	}
	if s.HasOneTimeDailyQuota() {
		return time.Time{}, false
	}
	today := timezone.StartOfDay(now)
	if !today.After(timezone.StartOfDay(*s.DailyWindowStart)) {
		return time.Time{}, false
	}
	return today, true
}

func (s *UserSubscription) canAutomaticallyResetWeeklyAt(now time.Time) bool {
	_, ok := s.automaticWindowStartAt(s.WeeklyWindowStart, 7*24*time.Hour, now)
	return ok
}

func (s *UserSubscription) canAutomaticallyResetMonthlyAt(now time.Time) bool {
	_, ok := s.automaticWindowStartAt(s.MonthlyWindowStart, 30*24*time.Hour, now)
	return ok
}

// windowResetAnchor 返回周/月窗口实际推进所依据的锚点。
// 早期订阅把首个窗口初始化在开通日零点；只有这个初始值是无歧义的，之后出现的
// 零点锚点可能来自手动重置，必须保持权威。
// 自动推进（automaticWindowStartAt）与对外展示的重置时间（WeeklyResetTime/
// MonthlyResetTime）必须共用这一修正，否则仪表盘显示的重置时间会早于窗口实际
// 滚动的时间。
// 日窗口按日历日对齐（automaticDailyWindowStartAt），不走这里。
func (s *UserSubscription) windowResetAnchor(previous time.Time) time.Time {
	legacyAnchor := startOfDay(s.StartsAt)
	if legacyAnchor.Before(s.StartsAt) && previous.Equal(legacyAnchor) {
		return s.StartsAt
	}
	return previous
}

// automaticWindowStartAt 计算周/月窗口（期限对齐滚动窗口）的当前窗口起点。
// 窗口从锚点按整数个 period 步进，且不越过订阅到期时间，避免最后一个不完整
// 周期重复发放额度（issue #5051）。日窗口不走此函数，见 automaticDailyWindowStartAt。
func (s *UserSubscription) automaticWindowStartAt(previous *time.Time, period time.Duration, now time.Time) (time.Time, bool) {
	if previous == nil {
		return time.Time{}, false
	}

	anchor := s.windowResetAnchor(*previous)
	next := anchor.Add(period)
	if now.Before(next) || !next.Before(s.ExpiresAt) {
		return time.Time{}, false
	}

	periods := now.Sub(anchor) / period
	lastPeriodBeforeExpiry := (s.ExpiresAt.Sub(anchor) - 1) / period
	if periods > lastPeriodBeforeExpiry {
		periods = lastPeriodBeforeExpiry
	}
	return anchor.Add(periods * period), true
}

func (s *UserSubscription) DailyResetTime() *time.Time {
	if s.DailyWindowStart == nil {
		return nil
	}
	if s.HasOneTimeDailyQuota() {
		t := s.ExpiresAt
		return &t
	}
	// 日窗口按日历日对齐：下次刷新固定在窗口起点所在日的次日 0 点。
	t := timezone.StartOfDay(*s.DailyWindowStart).AddDate(0, 0, 1)
	return &t
}

func (s *UserSubscription) WeeklyResetTime() *time.Time {
	if s.WeeklyWindowStart == nil {
		return nil
	}
	t := s.windowResetAnchor(*s.WeeklyWindowStart).Add(7 * 24 * time.Hour)
	return &t
}

func (s *UserSubscription) MonthlyResetTime() *time.Time {
	if s.MonthlyWindowStart == nil {
		return nil
	}
	t := s.windowResetAnchor(*s.MonthlyWindowStart).Add(30 * 24 * time.Hour)
	return &t
}

func (s *UserSubscription) CheckDailyLimit(group *Group, additionalCost float64) bool {
	limit := s.dailyLimitUSD(group)
	if limit == nil {
		return true
	}
	if additionalCost <= 0 {
		return s.DailyUsageUSD < *limit
	}
	return s.DailyUsageUSD+additionalCost <= *limit
}

func (s *UserSubscription) CheckWeeklyLimit(group *Group, additionalCost float64) bool {
	limit := s.weeklyLimitUSD(group)
	if limit == nil {
		return true
	}
	if additionalCost <= 0 {
		return s.WeeklyUsageUSD < *limit
	}
	return s.WeeklyUsageUSD+additionalCost <= *limit
}

func (s *UserSubscription) CheckMonthlyLimit(group *Group, additionalCost float64) bool {
	if group == nil || !group.HasMonthlyLimit() {
		return true
	}
	if additionalCost <= 0 {
		return s.MonthlyUsageUSD < *group.MonthlyLimitUSD
	}
	return s.MonthlyUsageUSD+additionalCost <= *group.MonthlyLimitUSD
}

func (s *UserSubscription) CheckAllLimits(group *Group, additionalCost float64) (daily, weekly, monthly bool) {
	daily = s.CheckDailyLimit(group, additionalCost)
	weekly = s.CheckWeeklyLimit(group, additionalCost)
	monthly = s.CheckMonthlyLimit(group, additionalCost)
	return
}

func (s *UserSubscription) dailyLimitUSD(group *Group) *float64 {
	if s != nil && s.DailyLimitUSD != nil && *s.DailyLimitUSD > 0 {
		return s.DailyLimitUSD
	}
	if group != nil && group.HasDailyLimit() {
		return group.DailyLimitUSD
	}
	return nil
}

func (s *UserSubscription) weeklyLimitUSD(group *Group) *float64 {
	if s != nil && s.WeeklyLimitUSD != nil && *s.WeeklyLimitUSD > 0 {
		return s.WeeklyLimitUSD
	}
	if group != nil && group.HasWeeklyLimit() {
		return group.WeeklyLimitUSD
	}
	return nil
}
