package service

import "time"

// UserSubscription 表示用户订阅。
//
// 订阅额度池模型（详见 doc/plan/2026-05-28-subscription-credit-pool-plan.md）：
//
//	可消费    : status='active' AND ExhaustedAt == nil AND ExpiresAt > now
//	等过期    : status='active' AND ExhaustedAt != nil（总额度已耗尽，等待 ExpiresAt 到达）
//	已过期    : status='expired'
//	暂停      : status='suspended'
//
// 月度窗口字段（MonthlyWindowStart / MonthlyUsageUSD）已从数据库移除，
// 但 Go struct 暂时保留为内存字段以兼容现有代码；Task 4 后将完全清理。
type UserSubscription struct {
	ID      int64
	UserID  int64
	GroupID *int64 // 额度池订阅可空（不绑定具体分组）

	// 额度池字段（新模型）
	PlanID         *int64
	ScopeType      string
	ScopeConfig    map[string]any
	QuotaLimitUSD  float64
	QuotaUsedUSD   float64
	DailyLimitUSD  *float64
	WeeklyLimitUSD *float64

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

	AssignedBy *int64
	AssignedAt time.Time
	Notes      string

	CreatedAt time.Time
	UpdatedAt time.Time

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

func (s *UserSubscription) DaysRemaining() int {
	if s.IsExpired() {
		return 0
	}
	return int(time.Until(s.ExpiresAt).Hours() / 24)
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

func (s *UserSubscription) NeedsDailyReset() bool {
	if s.DailyWindowStart == nil {
		return false
	}
	return time.Since(*s.DailyWindowStart) >= 24*time.Hour
}

func (s *UserSubscription) NeedsWeeklyReset() bool {
	if s.WeeklyWindowStart == nil {
		return false
	}
	return time.Since(*s.WeeklyWindowStart) >= 7*24*time.Hour
}

func (s *UserSubscription) NeedsMonthlyReset() bool {
	if s.MonthlyWindowStart == nil {
		return false
	}
	return time.Since(*s.MonthlyWindowStart) >= 30*24*time.Hour
}

func (s *UserSubscription) DailyResetTime() *time.Time {
	if s.DailyWindowStart == nil {
		return nil
	}
	t := s.DailyWindowStart.Add(24 * time.Hour)
	return &t
}

func (s *UserSubscription) WeeklyResetTime() *time.Time {
	if s.WeeklyWindowStart == nil {
		return nil
	}
	t := s.WeeklyWindowStart.Add(7 * 24 * time.Hour)
	return &t
}

func (s *UserSubscription) MonthlyResetTime() *time.Time {
	if s.MonthlyWindowStart == nil {
		return nil
	}
	t := s.MonthlyWindowStart.Add(30 * 24 * time.Hour)
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
