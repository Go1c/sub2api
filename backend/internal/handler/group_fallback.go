package handler

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"go.uber.org/zap"
)

// groupFallbackResult 包含分组级兜底的结果
type groupFallbackResult struct {
	APIKey       *service.APIKey           // 克隆到兜底分组 B 的 API Key（nil 表示无兜底）
	Subscription *service.UserSubscription // 重新解析的兜底分组 B 的订阅（可能为 nil，按余额计费）
	Group        *service.Group            // 解析出的兜底分组 B
	ErrorWritten bool                      // true 表示已写错误响应到客户端，调用方必须立即 return
}

// tryGroupFallback 尝试分组级兜底：当前分组 A 的上游账号全部不可用时，切换到分组 B 重试。
// 参数：
//   - ctx: 请求上下文
//   - reqLog: 请求日志记录器
//   - currentAPIKey: 当前使用的 API Key（必须有 Group）
//   - groupFallbackUsed: 单跳守卫标志，防止级联（已使用过分组兜底则直接返回零值）
//   - streamStarted: 流是否已开始写出（已写出则不可兜底，直接返回零值）
//   - resolveGroup: 闭包，根据分组 ID 解析分组对象
//   - resolveSubscription: 闭包，重新解析兜底分组的订阅/额度池
//   - checkBilling: 闭包，校验兜底分组的计费资格
//   - writeBillingErr: 闭包，写入计费错误响应
//
// 返回值：
//   - APIKey: 克隆到兜底分组 B 的 API Key（nil 表示无兜底或不适用）
//   - Subscription: 重新解析的兜底分组 B 的订阅（可能为 nil，按余额计费）
//   - Group: 解析出的兜底分组 B
//   - ErrorWritten: true 表示已写错误响应，调用方必须立即 return
func tryGroupFallback(
	ctx context.Context,
	reqLog *zap.Logger,
	currentAPIKey *service.APIKey,
	groupFallbackUsed bool,
	streamStarted bool,
	resolveGroup func(context.Context, int64) (*service.Group, error),
	resolveSubscription func(context.Context, *service.APIKey, *service.Group) *service.UserSubscription,
	checkBilling func(context.Context, *service.User, *service.APIKey, *service.Group, *service.UserSubscription) error,
	writeBillingErr func(status int, code, message string, retryAfter int),
) groupFallbackResult {
	// 守卫：单跳防环 + 前置条件检查
	if groupFallbackUsed ||
		streamStarted ||
		currentAPIKey == nil ||
		currentAPIKey.Group == nil ||
		currentAPIKey.Group.FallbackGroupIDOnExhausted == nil ||
		*currentAPIKey.Group.FallbackGroupIDOnExhausted <= 0 {
		return groupFallbackResult{}
	}

	fallbackGroupID := *currentAPIKey.Group.FallbackGroupIDOnExhausted

	// 解析兜底分组 B
	fallbackGroup, err := resolveGroup(ctx, fallbackGroupID)
	if err != nil {
		reqLog.Warn("gateway.group_fallback_resolve_failed",
			zap.Int64("fallback_group_id", fallbackGroupID),
			zap.Int64("current_group_id", currentAPIKey.Group.ID),
			zap.Error(err),
		)
		return groupFallbackResult{}
	}

	// 校验兜底分组 B
	if fallbackGroup.ID == currentAPIKey.Group.ID {
		reqLog.Warn("gateway.group_fallback_self_reference",
			zap.Int64("group_id", fallbackGroup.ID),
		)
		return groupFallbackResult{}
	}
	if !fallbackGroup.IsActive() {
		reqLog.Warn("gateway.group_fallback_not_active",
			zap.Int64("fallback_group_id", fallbackGroup.ID),
			zap.String("status", fallbackGroup.Status),
		)
		return groupFallbackResult{}
	}
	if fallbackGroup.Platform != currentAPIKey.Group.Platform {
		reqLog.Warn("gateway.group_fallback_platform_mismatch",
			zap.Int64("fallback_group_id", fallbackGroup.ID),
			zap.String("expected_platform", currentAPIKey.Group.Platform),
			zap.String("actual_platform", fallbackGroup.Platform),
		)
		return groupFallbackResult{}
	}
	if fallbackGroup.FallbackGroupIDOnExhausted != nil {
		reqLog.Warn("gateway.group_fallback_chaining_not_allowed",
			zap.Int64("fallback_group_id", fallbackGroup.ID),
			zap.Int64("nested_fallback_id", *fallbackGroup.FallbackGroupIDOnExhausted),
		)
		return groupFallbackResult{}
	}

	// 克隆当前 API Key 到兜底分组 B（同一密钥身份，不同分组）
	fallbackKey := cloneAPIKeyWithGroup(currentAPIKey, fallbackGroup)

	// 重新解析兜底分组 B 的订阅/额度池
	// 关键：运行时换组必须重新解析订阅，否则会跳过额度优先、100% 扣钱包
	fallbackSubscription := resolveSubscription(ctx, fallbackKey, fallbackGroup)

	// 校验计费资格（余额、订阅额度、日/周/月限额等）
	if err := checkBilling(ctx, fallbackKey.User, fallbackKey, fallbackGroup, fallbackSubscription); err != nil {
		reqLog.Info("gateway.group_fallback_billing_ineligible",
			zap.Int64("fallback_group_id", fallbackGroup.ID),
			zap.Error(err),
		)
		status, code, message, retryAfter := billingErrorDetails(err)
		writeBillingErr(status, code, message, retryAfter)
		return groupFallbackResult{ErrorWritten: true}
	}

	// 成功：返回兜底分组克隆的 key + 订阅 + 分组对象
	reqLog.Info("gateway.group_fallback_switch",
		zap.Int64("from_group_id", currentAPIKey.Group.ID),
		zap.Int64("to_group_id", fallbackGroup.ID),
		zap.String("platform", fallbackGroup.Platform),
	)
	return groupFallbackResult{
		APIKey:       fallbackKey,
		Subscription: fallbackSubscription,
		Group:        fallbackGroup,
		ErrorWritten: false,
	}
}
