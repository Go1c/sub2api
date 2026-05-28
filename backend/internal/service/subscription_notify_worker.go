package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// 订阅通知 worker 默认参数。
const (
	defaultSubscriptionNotifyInterval = 5 * time.Second
	defaultSubscriptionNotifyBatch    = 50
	subscriptionNotifyEventTimeout    = 30 * time.Second
)

// SubscriptionNotifyPayload 是 scheduler_outbox 中 subscription_notify
// 事件的 payload 反序列化结构。kind 取值参见 domain.SubscriptionCreditLedger* 常量。
//
//	{"user_id":..., "subscription_id":..., "kind":"limit_reached_total|limit_reached_daily|limit_reached_weekly|expired"}
type SubscriptionNotifyPayload struct {
	UserID         int64  `json:"user_id"`
	SubscriptionID int64  `json:"subscription_id"`
	Kind           string `json:"kind"`
}

// SubscriptionNotifyUserReader 仅暴露按 ID 取用户的能力，
// 让 worker 不依赖整套 UserRepository（便于测试 mock 最小化）。
type SubscriptionNotifyUserReader interface {
	GetByID(ctx context.Context, id int64) (*User, error)
}

// SubscriptionNotifyMessenger 负责把通知作为站内消息写入数据库。
// 实现注意：避免走 SiteMessageService.Send 等被 admin / 限频校验包裹的 API，
// 这里走最直接的 repository.Create 路径。
type SubscriptionNotifyMessenger interface {
	SendSubscriptionMessage(ctx context.Context, recipientID int64, subject, content string) error
}

// SubscriptionNotifyEmailer 负责把通知发送到用户邮箱。
type SubscriptionNotifyEmailer interface {
	SendSubscriptionEmail(ctx context.Context, to, subject, body string) error
}

// SubscriptionNotifySettingsReader 提供 setting 读取能力，
// worker 只用 GetValue（与 SettingRepository 兼容）。
type SubscriptionNotifySettingsReader interface {
	GetValue(ctx context.Context, key string) (string, error)
}

// SubscriptionNotifyOutboxRepository 用于按 event_type 拉取 subscription_notify
// outbox 事件。与现有 SchedulerOutboxRepository 互补：scheduler 自己的 worker 拉的是
// account_changed / group_changed 等事件，subscription_notify 走这里。
type SubscriptionNotifyOutboxRepository interface {
	ListSubscriptionNotifyAfter(ctx context.Context, afterID int64, limit int) ([]SchedulerOutboxEvent, error)
	MaxSubscriptionNotifyID(ctx context.Context) (int64, error)
}

// SubscriptionNotifyService 处理单条 subscription_notify outbox event。
//
// 关键约束（来自 plan）：
//   - 失败仅记日志，不重试；
//   - 任一渠道失败不影响另一渠道；
//   - invalid payload 直接吞错返回 nil（避免 worker 卡在坏数据上）。
type SubscriptionNotifyService struct {
	users     SubscriptionNotifyUserReader
	messenger SubscriptionNotifyMessenger
	emailer   SubscriptionNotifyEmailer
	settings  SubscriptionNotifySettingsReader
}

// NewSubscriptionNotifyService 构造通知服务。任一依赖可为 nil，但相应渠道会被跳过。
func NewSubscriptionNotifyService(
	users SubscriptionNotifyUserReader,
	messenger SubscriptionNotifyMessenger,
	emailer SubscriptionNotifyEmailer,
	settings SubscriptionNotifySettingsReader,
) *SubscriptionNotifyService {
	return &SubscriptionNotifyService{
		users:     users,
		messenger: messenger,
		emailer:   emailer,
		settings:  settings,
	}
}

// Handle 处理单个 outbox payload。即使解析失败也返回 nil，
// 否则 worker 会反复重试同一条坏事件而无法推进 watermark。
func (s *SubscriptionNotifyService) Handle(ctx context.Context, payload json.RawMessage) error {
	if s == nil {
		return nil
	}
	var p SubscriptionNotifyPayload
	if len(payload) == 0 {
		slog.Warn("subscription_notify: empty payload, skipping")
		return nil
	}
	if err := json.Unmarshal(payload, &p); err != nil {
		slog.Error("subscription_notify: invalid payload, skipping",
			"error", err,
			"raw", truncateForLog(payload, 200))
		return nil
	}
	if p.UserID <= 0 || p.SubscriptionID <= 0 || strings.TrimSpace(p.Kind) == "" {
		slog.Warn("subscription_notify: payload missing required fields, skipping",
			"user_id", p.UserID,
			"subscription_id", p.SubscriptionID,
			"kind", p.Kind)
		return nil
	}

	subject, body, plainBody := s.buildContent(ctx, p)

	// 站内消息：失败仅记日志，不影响后续邮件渠道。
	if s.messenger != nil {
		if err := s.messenger.SendSubscriptionMessage(ctx, p.UserID, subject, plainBody); err != nil {
			slog.Error("subscription_notify: site message failed",
				"error", err,
				"user_id", p.UserID,
				"subscription_id", p.SubscriptionID,
				"kind", p.Kind)
		}
	}

	// 邮件：需要先把 user_id 解析为 email。失败仅记日志。
	if s.emailer != nil {
		email := s.resolveUserEmail(ctx, p.UserID)
		if email == "" {
			slog.Warn("subscription_notify: user has no email, skipping email channel",
				"user_id", p.UserID,
				"subscription_id", p.SubscriptionID,
				"kind", p.Kind)
		} else if err := s.emailer.SendSubscriptionEmail(ctx, email, subject, body); err != nil {
			slog.Error("subscription_notify: email failed",
				"error", err,
				"user_id", p.UserID,
				"subscription_id", p.SubscriptionID,
				"kind", p.Kind)
		}
	}

	return nil
}

// resolveUserEmail 取用户邮箱；用户查询失败时返回空串（调用方会跳过邮件渠道）。
func (s *SubscriptionNotifyService) resolveUserEmail(ctx context.Context, userID int64) string {
	if s.users == nil {
		return ""
	}
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		slog.Warn("subscription_notify: lookup user failed",
			"error", err,
			"user_id", userID)
		return ""
	}
	if user == nil {
		return ""
	}
	return strings.TrimSpace(user.Email)
}

// buildContent 根据 kind 生成站内消息标题、邮件 HTML 正文、站内消息纯文本正文。
// 同一份文案，emailer 用 HTML，messenger 用纯文本，便于阅读。
func (s *SubscriptionNotifyService) buildContent(ctx context.Context, p SubscriptionNotifyPayload) (subject, htmlBody, plainBody string) {
	link := s.resolveRepurchaseURL(ctx)
	switch p.Kind {
	case subscriptionNotifyKindLimitTotal():
		subject = "订阅额度已用完"
		plainBody = fmt.Sprintf(
			"您当前的订阅额度已用完，订阅 ID #%d。"+
				"\n\n后续请求将自动改用账户余额扣费；如需继续使用订阅服务，请前往以下链接重新订阅：\n%s",
			p.SubscriptionID, link)
		htmlBody = renderSubscriptionNotifyEmail(
			"订阅额度已用完",
			fmt.Sprintf("您当前的订阅额度（订阅 ID <strong>#%d</strong>）已经用完。", p.SubscriptionID),
			"后续请求将自动改用账户余额扣费。如需继续使用订阅服务，欢迎重新订阅。",
			link,
		)
	case subscriptionNotifyKindLimitDaily():
		subject = "订阅已到达日限额"
		plainBody = fmt.Sprintf(
			"您的订阅（订阅 ID #%d）已到达今日的日限额。"+
				"\n\n本日剩余请求将自动改用账户余额扣费，日限会在 UTC 0 点重置。如需调整订阅或额度，请访问：\n%s",
			p.SubscriptionID, link)
		htmlBody = renderSubscriptionNotifyEmail(
			"订阅已到达日限额",
			fmt.Sprintf("您的订阅（订阅 ID <strong>#%d</strong>）已到达今日的日限额。", p.SubscriptionID),
			"本日剩余请求将自动改用账户余额扣费，日限会在 UTC 0 点重置。",
			link,
		)
	case subscriptionNotifyKindLimitWeekly():
		subject = "订阅已到达周限额"
		plainBody = fmt.Sprintf(
			"您的订阅（订阅 ID #%d）已到达本周的周限额。"+
				"\n\n本周剩余请求将自动改用账户余额扣费，周限会在下周 UTC 周一 0 点重置。如需调整订阅或额度，请访问：\n%s",
			p.SubscriptionID, link)
		htmlBody = renderSubscriptionNotifyEmail(
			"订阅已到达周限额",
			fmt.Sprintf("您的订阅（订阅 ID <strong>#%d</strong>）已到达本周的周限额。", p.SubscriptionID),
			"本周剩余请求将自动改用账户余额扣费，周限会在下周 UTC 周一 0 点重置。",
			link,
		)
	case subscriptionNotifyKindExpired():
		subject = "订阅已过期"
		plainBody = fmt.Sprintf(
			"您的订阅（订阅 ID #%d）已过期，剩余额度已自动销毁。"+
				"\n\n如需继续使用订阅服务，请前往以下链接重新订阅：\n%s",
			p.SubscriptionID, link)
		htmlBody = renderSubscriptionNotifyEmail(
			"订阅已过期",
			fmt.Sprintf("您的订阅（订阅 ID <strong>#%d</strong>）已经过期，剩余额度已自动销毁。", p.SubscriptionID),
			"如需继续使用订阅服务，请重新订阅。",
			link,
		)
	default:
		subject = "订阅状态变更通知"
		plainBody = fmt.Sprintf(
			"您的订阅（订阅 ID #%d）状态发生变更（kind=%s）。"+
				"\n\n如需查看详情，请访问：\n%s",
			p.SubscriptionID, p.Kind, link)
		htmlBody = renderSubscriptionNotifyEmail(
			"订阅状态变更通知",
			fmt.Sprintf("您的订阅（订阅 ID <strong>#%d</strong>）状态发生变更：<code>%s</code>。",
				p.SubscriptionID, html.EscapeString(p.Kind)),
			"如需查看详情，请前往订阅页查看。",
			link,
		)
	}
	return subject, htmlBody, plainBody
}

// resolveRepurchaseURL 读取重新订阅链接。优先级：
//  1. SettingKeySubscriptionCreditPoolRepurchaseURL
//  2. SettingKeyFrontendURL（前端订阅页路径作为锚点：/subscriptions）
//  3. 空字符串（文案不包含链接时仍可读，但无跳转）
func (s *SubscriptionNotifyService) resolveRepurchaseURL(ctx context.Context) string {
	if s.settings == nil {
		return ""
	}
	if raw, err := s.settings.GetValue(ctx, SettingKeySubscriptionCreditPoolRepurchaseURL); err == nil {
		if v := strings.TrimSpace(raw); v != "" {
			return v
		}
	}
	frontend, err := s.settings.GetValue(ctx, SettingKeyFrontendURL)
	if err != nil || strings.TrimSpace(frontend) == "" {
		return ""
	}
	base := strings.TrimRight(strings.TrimSpace(frontend), "/")
	return base + "/subscriptions"
}

// renderSubscriptionNotifyEmail 渲染统一风格的通知邮件正文。
// 风格参考 buildBalanceLowEmailTemplate（轻量内联 CSS、双语副标题）。
func renderSubscriptionNotifyEmail(title, lead, hint, link string) string {
	linkBlock := ""
	if strings.TrimSpace(link) != "" {
		safeLink := html.EscapeString(strings.TrimSpace(link))
		linkBlock = fmt.Sprintf(
			`<p style="text-align:center;margin-top:24px;"><a href="%s" style="display:inline-block;padding:12px 32px;background:linear-gradient(135deg,#4f8cff 0%%,#1a2f5a 100%%);color:#fff;text-decoration:none;border-radius:6px;font-size:15px;font-weight:bold;">重新订阅 / Manage Subscription</a></p>`,
			safeLink)
	}
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;background:#f5f5f5;margin:0;padding:20px;">
<div style="max-width:600px;margin:0 auto;background:#fff;border-radius:8px;overflow:hidden;box-shadow:0 2px 8px rgba(0,0,0,0.08);">
<div style="background:linear-gradient(135deg,#4f8cff 0%%,#1a2f5a 100%%);color:#fff;padding:28px;text-align:center;">
<h1 style="margin:0;font-size:22px;">%s</h1>
</div>
<div style="padding:32px;color:#333;line-height:1.6;font-size:15px;">
<p style="margin:0 0 12px 0;">%s</p>
<p style="margin:0;color:#666;font-size:14px;">%s</p>
%s
</div>
<div style="background:#f8f9fa;padding:16px;text-align:center;color:#999;font-size:12px;">此邮件由系统自动发送，请勿回复。</div>
</div>
</body>
</html>`, html.EscapeString(title), lead, html.EscapeString(hint), linkBlock)
}

// subscriptionNotifyKind* helpers 构造 "limit_reached_<dim>" / "expired" 的 kind 字符串，
// 与 plan / domain 常量保持一致。
func subscriptionNotifyKindLimitTotal() string {
	return "limit_reached_" + SubscriptionLimitReachedTotal
}
func subscriptionNotifyKindLimitDaily() string {
	return "limit_reached_" + SubscriptionLimitReachedDaily
}
func subscriptionNotifyKindLimitWeekly() string {
	return "limit_reached_" + SubscriptionLimitReachedWeekly
}
func subscriptionNotifyKindExpired() string {
	return "expired"
}

// ------------------------------------------------------------
// Worker：周期性拉取 scheduler_outbox 中 subscription_notify 事件，分发给 service。
// ------------------------------------------------------------

// SubscriptionNotifyWorker 周期性拉取 scheduler_outbox 中 subscription_notify
// 事件并交给 SubscriptionNotifyService 处理。
//
// 设计选择：
//   - 不复用 SchedulerSnapshotService 的 dispatch（它的 watermark 与所有 outbox 事件共用，
//     注入 subscription_notify case 会让两类负载耦合在一起）；
//   - 也不为 SubscriptionNotifyWorker 引入持久化 watermark，启动时取 MaxID 作为起点，
//     符合 plan "通知最多发一次，失败仅记日志，不重试" 的容忍语义（重启窗口内丢通知可接受，
//     幂等性最终由 ledger 的 event_key 唯一索引在事务侧保证）。
type SubscriptionNotifyWorker struct {
	repo     SubscriptionNotifyOutboxRepository
	handler  *SubscriptionNotifyService
	interval time.Duration
	batch    int

	stopCh   chan struct{}
	stopOnce sync.Once
	wg       sync.WaitGroup

	watermark   int64
	watermarkMu sync.Mutex
}

// NewSubscriptionNotifyWorker 构造 worker。interval/batch <=0 走默认值。
func NewSubscriptionNotifyWorker(repo SubscriptionNotifyOutboxRepository, handler *SubscriptionNotifyService, interval time.Duration, batch int) *SubscriptionNotifyWorker {
	if interval <= 0 {
		interval = defaultSubscriptionNotifyInterval
	}
	if batch <= 0 {
		batch = defaultSubscriptionNotifyBatch
	}
	return &SubscriptionNotifyWorker{
		repo:     repo,
		handler:  handler,
		interval: interval,
		batch:    batch,
		stopCh:   make(chan struct{}),
	}
}

// Start 启动后台 goroutine。可重复调用安全：内部判断已停止则跳过。
func (w *SubscriptionNotifyWorker) Start() {
	if w == nil || w.repo == nil || w.handler == nil {
		return
	}
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		w.run()
	}()
}

// Stop 关闭 worker，等待当前 poll 完成。
func (w *SubscriptionNotifyWorker) Stop() {
	if w == nil {
		return
	}
	w.stopOnce.Do(func() {
		close(w.stopCh)
	})
	w.wg.Wait()
}

func (w *SubscriptionNotifyWorker) run() {
	// 启动时把 watermark 推到当前 MaxID，避免历史事件被重复处理。
	initCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	if maxID, err := w.repo.MaxSubscriptionNotifyID(initCtx); err == nil {
		w.setWatermark(maxID)
	} else {
		logger.LegacyPrintf("service.subscription_notify_worker", "[SubscriptionNotify] init max id failed: %v", err)
	}
	cancel()

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.pollOnce()
		}
	}
}

// pollOnce 拉取一批事件并按顺序分发。出错不退出，等待下次 tick。
// 暴露为非导出方法是为单元测试可单步驱动；导出请用 PollOnceForTest。
func (w *SubscriptionNotifyWorker) pollOnce() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	events, err := w.repo.ListSubscriptionNotifyAfter(ctx, w.getWatermark(), w.batch)
	if err != nil {
		logger.LegacyPrintf("service.subscription_notify_worker", "[SubscriptionNotify] poll failed: %v", err)
		return
	}
	if len(events) == 0 {
		return
	}
	for _, evt := range events {
		evtCtx, evtCancel := context.WithTimeout(context.Background(), subscriptionNotifyEventTimeout)
		var raw json.RawMessage
		if evt.Payload != nil {
			if encoded, err := json.Marshal(evt.Payload); err == nil {
				raw = encoded
			}
		}
		_ = w.handler.Handle(evtCtx, raw)
		evtCancel()
	}
	last := events[len(events)-1].ID
	w.setWatermark(last)
}

// PollOnceForTest 暴露给测试直接驱动一次拉取（生产代码请用 Start/Stop）。
func (w *SubscriptionNotifyWorker) PollOnceForTest() {
	w.pollOnce()
}

func (w *SubscriptionNotifyWorker) getWatermark() int64 {
	w.watermarkMu.Lock()
	defer w.watermarkMu.Unlock()
	return w.watermark
}

func (w *SubscriptionNotifyWorker) setWatermark(id int64) {
	w.watermarkMu.Lock()
	defer w.watermarkMu.Unlock()
	if id > w.watermark {
		w.watermark = id
	}
}

// ErrSubscriptionNotifyWorkerStopped 标识 worker 已停止；外部一般不会用到。
var ErrSubscriptionNotifyWorkerStopped = errors.New("subscription notify worker stopped")

// ------------------------------------------------------------
// 生产适配器：把 SiteMessageRepository / *EmailService 接到 worker 用的小接口上。
// ------------------------------------------------------------

// subscriptionNotifyMessengerAdapter 把站内消息写入直接走 SiteMessageRepository.Create。
// 用 senderID=recipientID 让系统通知以"自发"形式存储，绕开 SiteMessageService.Send 的
// admin 角色校验和每日条数限频。这与 SiteMessageService.SendLotteryPrize 使用任意
// campaign.CreatedBy 作为 senderID 的做法语义一致。
type subscriptionNotifyMessengerAdapter struct {
	repo SiteMessageRepository
}

func (a *subscriptionNotifyMessengerAdapter) SendSubscriptionMessage(ctx context.Context, recipientID int64, subject, content string) error {
	if a == nil || a.repo == nil {
		return errors.New("site message repository not configured")
	}
	now := time.Now()
	msg := &SiteMessage{
		SenderID:    recipientID,
		RecipientID: recipientID,
		Subject:     strings.TrimSpace(subject),
		Content:     strings.TrimSpace(content),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if msg.Subject == "" || msg.Content == "" {
		return errors.New("subscription site message: subject/content required")
	}
	return a.repo.Create(ctx, msg)
}

// subscriptionNotifyEmailerAdapter 直接调 EmailService.SendEmail。
// 失���由调用方记录日志，不在适配器层吞错。
type subscriptionNotifyEmailerAdapter struct {
	email *EmailService
}

func (a *subscriptionNotifyEmailerAdapter) SendSubscriptionEmail(ctx context.Context, to, subject, body string) error {
	if a == nil || a.email == nil {
		return errors.New("email service not configured")
	}
	return a.email.SendEmail(ctx, to, subject, body)
}
