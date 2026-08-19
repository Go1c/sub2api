package checkin

import (
	"fmt"
	"time"
)

type milestoneResponse struct {
	Day   int    `json:"day"`
	Bonus string `json:"bonus"`
}

type settingsResponse struct {
	Enabled             bool                `json:"enabled"`
	MinReward           string              `json:"min_reward"`
	MaxReward           string              `json:"max_reward"`
	Timezone            string              `json:"timezone"`
	DailyCap            string              `json:"daily_cap"`
	Milestones          []milestoneResponse `json:"milestones"`
	MaximumSingleReward string              `json:"maximum_single_reward"`
	UpdatedAt           string              `json:"updated_at"`
}

type recordResponse struct {
	ID               int64  `json:"id"`
	UserID           *int64 `json:"user_id,omitempty"`
	UserEmail        string `json:"user_email"`
	Username         string `json:"username"`
	BusinessDate     string `json:"business_date"`
	CheckedAt        string `json:"checked_at"`
	Timezone         string `json:"timezone"`
	StreakDays       int    `json:"streak_days"`
	CycleDay         int    `json:"cycle_day"`
	MilestoneDay     *int   `json:"milestone_day,omitempty"`
	BaseReward       string `json:"base_reward"`
	MilestoneBonus   string `json:"milestone_bonus"`
	ActualReward     string `json:"actual_reward"`
	Status           string `json:"status"`
	BalanceAfter     string `json:"balance_after"`
	ClientIP         string `json:"client_ip,omitempty"`
	UserAgent        string `json:"user_agent,omitempty"`
	AlreadyCheckedIn *bool  `json:"already_checked_in,omitempty"`
}

type nextMilestoneResponse struct {
	Day       int    `json:"day"`
	Bonus     string `json:"bonus"`
	DaysUntil int    `json:"days_until"`
}

type userStatusResponse struct {
	Enabled        bool                   `json:"enabled"`
	CheckedInToday bool                   `json:"checked_in_today"`
	TotalCheckIns  int64                  `json:"total_checkins"`
	TotalReward    string                 `json:"total_reward"`
	CurrentStreak  int                    `json:"current_streak"`
	CycleDay       int                    `json:"cycle_day"`
	NextMilestone  *nextMilestoneResponse `json:"next_milestone"`
	Balance        string                 `json:"balance"`
	TodayRecord    *recordResponse        `json:"today_record"`
	RecentRecords  []recordResponse       `json:"recent_records"`
}

func toSettingsResponse(settings Settings) settingsResponse {
	milestones := make([]milestoneResponse, 0, len(settings.Milestones))
	for _, milestone := range settings.Milestones {
		milestones = append(milestones, milestoneResponse{Day: milestone.Day, Bonus: formatAmount(milestone.Bonus)})
	}
	return settingsResponse{
		Enabled:             settings.Enabled,
		MinReward:           formatAmount(settings.MinReward),
		MaxReward:           formatAmount(settings.MaxReward),
		Timezone:            settings.Timezone,
		DailyCap:            formatAmount(settings.DailyCap),
		Milestones:          milestones,
		MaximumSingleReward: formatAmount(settings.MaximumSingleReward()),
		UpdatedAt:           formatTimestamp(settings.UpdatedAt),
	}
}

func toRecordResponse(record Record, includeClient bool) recordResponse {
	response := recordResponse{
		ID:             record.ID,
		UserEmail:      record.UserEmail,
		Username:       record.Username,
		BusinessDate:   formatBusinessDate(record.BusinessDate),
		CheckedAt:      formatTimestamp(record.CheckedAt),
		Timezone:       record.Timezone,
		StreakDays:     record.StreakDays,
		CycleDay:       record.CycleDay,
		BaseReward:     formatAmount(record.BaseReward),
		MilestoneBonus: formatAmount(record.MilestoneBonus),
		ActualReward:   formatAmount(record.ActualReward),
		Status:         record.Status,
		BalanceAfter:   formatAmount(record.BalanceAfter),
	}
	if record.UserID > 0 {
		userID := record.UserID
		response.UserID = &userID
	}
	if record.MilestoneDay > 0 {
		milestoneDay := record.MilestoneDay
		response.MilestoneDay = &milestoneDay
	}
	if includeClient {
		response.ClientIP = record.ClientIP
		response.UserAgent = record.UserAgent
	}
	return response
}

func toUserStatusResponse(status UserStatus) userStatusResponse {
	response := userStatusResponse{
		Enabled:        status.Enabled,
		CheckedInToday: status.CheckedInToday,
		TotalCheckIns:  status.TotalCheckIns,
		TotalReward:    formatAmount(status.TotalReward),
		CurrentStreak:  status.CurrentStreak,
		CycleDay:       status.CycleDay,
		Balance:        formatAmount(status.Balance),
		RecentRecords:  make([]recordResponse, 0, len(status.RecentRecords)),
	}
	if status.NextMilestone != nil {
		response.NextMilestone = &nextMilestoneResponse{
			Day:       status.NextMilestone.Day,
			Bonus:     formatAmount(status.NextMilestone.Bonus),
			DaysUntil: status.NextMilestone.DaysUntil,
		}
	}
	if status.TodayRecord != nil {
		today := toRecordResponse(*status.TodayRecord, false)
		response.TodayRecord = &today
	}
	for _, record := range status.RecentRecords {
		response.RecentRecords = append(response.RecentRecords, toRecordResponse(record, false))
	}
	return response
}

func formatBusinessDate(value time.Time) string {
	year, month, day := value.Date()
	return fmt.Sprintf("%04d-%02d-%02d", year, month, day)
}

func formatTimestamp(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}
