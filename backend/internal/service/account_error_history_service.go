package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"regexp"
	"strings"
	"time"
)

// accountErrorRecordTimeout 异步写入的兜底超时，防 goroutine 泄漏。
const accountErrorRecordTimeout = 5 * time.Second

// accountErrorMessageMaxLen 入库 message 截断上限，避免超长上游响应撑大表。
const accountErrorMessageMaxLen = 2000

// accountErrorFingerprintNormalizeMaxLen 归一化文本参与指纹计算的截断长度。
const accountErrorFingerprintNormalizeMaxLen = 256

// AccountErrorHistoryService 账号错误历史服务：统一计算指纹、装配快照、异步 best-effort 写入。
// 这是非关键监控数据——写入不阻塞调用方，出错只 warn 不上抛，丢几条可接受。
type AccountErrorHistoryService struct {
	repo AccountErrorHistoryRepository
}

// NewAccountErrorHistoryService 创建服务实例。
func NewAccountErrorHistoryService(repo AccountErrorHistoryRepository) *AccountErrorHistoryService {
	return &AccountErrorHistoryService{repo: repo}
}

// RecordAccountError 异步、非阻塞地记录一条账号错误（best-effort）。
// 调用方只管传上下文，不应等待或处理返回——本方法不返回错误。
// 真实请求（gateway 源）传满 UserID/UserEmail/Model/UpstreamStatusCode，其余来源留空。
func (s *AccountErrorHistoryService) RecordAccountError(ctx context.Context, event AccountErrorEvent) {
	if s == nil || s.repo == nil {
		return
	}
	if event.AccountID == 0 {
		return
	}

	row := &AccountErrorRow{
		AccountID:          event.AccountID,
		UserID:             event.UserID,
		UserEmail:          event.UserEmail,
		Model:              event.Model,
		UpstreamStatusCode: event.UpstreamStatusCode,
		Source:             normalizeAccountErrorSource(event.Source),
		Message:            truncateRunes(event.Message, accountErrorMessageMaxLen),
		Fingerprint:        computeAccountErrorFingerprint(event.Message),
	}

	// 脱离调用方 ctx 的取消（主请求结束不应中断监控写入），但带超时防泄漏。
	detached := context.WithoutCancel(ctx)
	go func() {
		writeCtx, cancel := context.WithTimeout(detached, accountErrorRecordTimeout)
		defer cancel()
		if err := s.repo.Record(writeCtx, row); err != nil {
			slog.Warn("record account error history failed (best-effort)",
				"account_id", row.AccountID, "source", row.Source, "error", err)
		}
	}()
}

// ListRecentAccountErrors 透传 repo.ListRecent，供 handler 懒加载查询。
func (s *AccountErrorHistoryService) ListRecentAccountErrors(ctx context.Context, accountID int64, limit int) ([]*AccountErrorHistoryEntry, error) {
	return s.repo.ListRecent(ctx, accountID, limit)
}

// normalizeAccountErrorSource 把来源兜底到合法枚举值，未知归到 schedule（与 enum/CHECK 对齐）。
func normalizeAccountErrorSource(source string) string {
	switch source {
	case AccountErrorSourceGateway, AccountErrorSourceTest, AccountErrorSourceRateLimit,
		AccountErrorSourceRefresh, AccountErrorSourceSchedule, AccountErrorSourceAdmin:
		return source
	default:
		return AccountErrorSourceSchedule
	}
}

// accountErrorVolatilePattern 匹配错误文本里的易变片段（数字串、十六进制 id、UUID 等），
// 归一化时统一替换为占位符，让「同类错误、仅数字/时间/请求 id 不同」折叠到同一指纹。
var accountErrorVolatilePattern = regexp.MustCompile(
	`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}` + // UUID
		`|0x[0-9a-fA-F]+` + // 十六进制
		`|\d+`, // 任意数字串（含时间戳、计数、毫秒等）
)

// computeAccountErrorFingerprint 对 message 归一化后取 sha256 前 16 字节 hex 作为指纹。
func computeAccountErrorFingerprint(message string) string {
	normalized := normalizeAccountErrorMessage(message)
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:8]) // 8 字节 = 16 hex 字符
}

// normalizeAccountErrorMessage 归一化错误文本：小写、压缩空白、抹掉易变片段、截断。
// 独立函数便于单测。
func normalizeAccountErrorMessage(message string) string {
	s := strings.ToLower(strings.TrimSpace(message))
	s = accountErrorVolatilePattern.ReplaceAllString(s, "#")
	s = strings.Join(strings.Fields(s), " ") // 压缩连续空白
	return truncateRunes(s, accountErrorFingerprintNormalizeMaxLen)
}

// truncateRunes 按 rune 截断，避免切坏多字节字符。
func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}
