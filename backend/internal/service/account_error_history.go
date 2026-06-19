package service

import (
	"context"
	"time"
)

// 账号错误历史来源枚举。与 migrations/152 的 CHECK 约束、ent schema 的 enum 严格对齐。
const (
	AccountErrorSourceGateway   = "gateway"   // 网关真实请求上游错误（字段齐全）
	AccountErrorSourceTest      = "test"      // 账号测试失败
	AccountErrorSourceRateLimit = "ratelimit" // 限流相关
	AccountErrorSourceRefresh   = "refresh"   // token 刷新失败
	AccountErrorSourceSchedule  = "schedule"  // 调度循环检测到的异常
	AccountErrorSourceAdmin     = "admin"     // 管理员手动置错
)

// AccountErrorEvent 是埋点处提交给 RecordAccountError 的原始事件。
// 只有 gateway（真实请求）来源会填满 UserID/UserEmail/Model/UpstreamStatusCode，
// 其余来源对应字段留 nil。Fingerprint 由 service 层统一计算，调用方无需填。
type AccountErrorEvent struct {
	AccountID          int64
	UserID             *int64
	UserEmail          *string
	Model              *string
	UpstreamStatusCode *int
	Source             string
	Message            string
}

// AccountErrorRow 是 service 层向 repository 提交的入库行（含已算好的 Fingerprint）。
type AccountErrorRow struct {
	AccountID          int64
	UserID             *int64
	UserEmail          *string
	Model              *string
	UpstreamStatusCode *int
	Source             string
	Message            string
	Fingerprint        string
}

// AccountErrorHistoryEntry 是查询返回行（含 ent 主键 ID 与折叠计数）。
type AccountErrorHistoryEntry struct {
	ID                 int64
	AccountID          int64
	UserID             *int64
	UserEmail          *string
	Model              *string
	UpstreamStatusCode *int
	Source             string
	Message            string
	DupCount           int
	CreatedAt          time.Time
}

// AccountErrorHistoryRepository 账号错误历史数据访问接口。
// 去重折叠 / 写入节流 / 每账号裁剪都收口在 repository 实现里（命中索引的原生 SQL）。
type AccountErrorHistoryRepository interface {
	// Record best-effort 写入一行错误历史：60s 窗口内同维度折叠为一行并 dup_count+1，
	// 命中每账号最小写入间隔节流时直接丢弃，写入成功后裁剪该账号超出上限的旧行。
	Record(ctx context.Context, row *AccountErrorRow) error
	// ListRecent 按 created_at 倒序返回某账号最近 limit 条错误历史。
	ListRecent(ctx context.Context, accountID int64, limit int) ([]*AccountErrorHistoryEntry, error)
}
