package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"strconv"
	"strings"
	"sync"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/dgraph-io/ristretto"
	"golang.org/x/sync/singleflight"
)

// MaxExpiresAt is the maximum allowed expiration date (year 2099)
// This prevents time.Time JSON serialization errors (RFC 3339 requires year <= 9999)
var MaxExpiresAt = time.Date(2099, 12, 31, 23, 59, 59, 0, time.UTC)

// MaxValidityDays is the maximum allowed validity days for subscriptions (100 years)
const MaxValidityDays = 36500

var (
	ErrSubscriptionNotFound       = infraerrors.NotFound("SUBSCRIPTION_NOT_FOUND", "subscription not found")
	ErrSubscriptionExpired        = infraerrors.Forbidden("SUBSCRIPTION_EXPIRED", "subscription has expired")
	ErrSubscriptionSuspended      = infraerrors.Forbidden("SUBSCRIPTION_SUSPENDED", "subscription is suspended")
	ErrSubscriptionAlreadyExists  = infraerrors.Conflict("SUBSCRIPTION_ALREADY_EXISTS", "subscription already exists for this user and group")
	ErrSubscriptionAssignConflict = infraerrors.Conflict("SUBSCRIPTION_ASSIGN_CONFLICT", "subscription exists but request conflicts with existing assignment semantics")
	ErrGroupNotSubscriptionType   = infraerrors.BadRequest("GROUP_NOT_SUBSCRIPTION_TYPE", "group is not a subscription type")
	ErrInvalidInput               = infraerrors.BadRequest("INVALID_INPUT", "at least one of resetDaily, resetWeekly, or resetMonthly must be true")
	ErrDailyLimitExceeded         = infraerrors.TooManyRequests("DAILY_LIMIT_EXCEEDED", "daily usage limit exceeded")
	ErrWeeklyLimitExceeded        = infraerrors.TooManyRequests("WEEKLY_LIMIT_EXCEEDED", "weekly usage limit exceeded")
	ErrMonthlyLimitExceeded       = infraerrors.TooManyRequests("MONTHLY_LIMIT_EXCEEDED", "monthly usage limit exceeded")
	ErrSubscriptionNilInput       = infraerrors.BadRequest("SUBSCRIPTION_NIL_INPUT", "subscription input cannot be nil")
	ErrAdjustWouldExpire          = infraerrors.BadRequest("ADJUST_WOULD_EXPIRE", "adjustment would result in expired subscription (remaining days must be > 0)")
	// 订阅额度池新增错误
	ErrSQLDBUnavailable              = infraerrors.InternalServer("SQL_DB_UNAVAILABLE", "sql.DB not wired into repository")
	ErrSubscriptionRenewalNotAllowed = infraerrors.Conflict("SUBSCRIPTION_RENEWAL_NOT_ALLOWED", "current subscription is still usable, cannot purchase a new one before exhaustion or expiration")
	ErrAlreadyHasUsableSubscription  = infraerrors.Conflict("ALREADY_HAS_USABLE_SUBSCRIPTION", "user already has a usable subscription")
)

// SubscriptionService 订阅服务
type SubscriptionService struct {
	groupRepo                            GroupRepository
	userSubRepo                          UserSubscriptionRepository
	creditLedgerRepo                     SubscriptionCreditLedgerRepository
	billingCacheService                  *BillingCacheService
	entClient                            *dbent.Client
	subscriptionMultiplePurchasesEnabled func(context.Context) bool

	// L1 缓存：加速中间件热路径的订阅查询
	subCacheL1     *ristretto.Cache
	subCacheGroup  singleflight.Group
	subCacheTTL    time.Duration
	subCacheJitter int // 抖动百分比
	creditSubMiss  sync.Map

	maintenanceQueue *SubscriptionMaintenanceQueue
}

func (s *SubscriptionService) SetSubscriptionCreditLedgerRepository(repo SubscriptionCreditLedgerRepository) *SubscriptionService {
	s.creditLedgerRepo = repo
	return s
}

func (s *SubscriptionService) SetSubscriptionMultiplePurchasesEnabledReader(fn func(context.Context) bool) *SubscriptionService {
	if s != nil {
		s.subscriptionMultiplePurchasesEnabled = fn
	}
	return s
}

func (s *SubscriptionService) multiplePurchasesEnabled(ctx context.Context) bool {
	return s != nil && s.subscriptionMultiplePurchasesEnabled != nil && s.subscriptionMultiplePurchasesEnabled(ctx)
}

// NewSubscriptionService 创建订阅服务
func NewSubscriptionService(groupRepo GroupRepository, userSubRepo UserSubscriptionRepository, billingCacheService *BillingCacheService, entClient *dbent.Client, cfg *config.Config) *SubscriptionService {
	svc := &SubscriptionService{
		groupRepo:           groupRepo,
		userSubRepo:         userSubRepo,
		billingCacheService: billingCacheService,
		entClient:           entClient,
	}
	svc.initSubCache(cfg)
	svc.initMaintenanceQueue(cfg)
	return svc
}

func (s *SubscriptionService) initMaintenanceQueue(cfg *config.Config) {
	if cfg == nil {
		return
	}
	mc := cfg.SubscriptionMaintenance
	if mc.WorkerCount <= 0 || mc.QueueSize <= 0 {
		return
	}
	s.maintenanceQueue = NewSubscriptionMaintenanceQueue(mc.WorkerCount, mc.QueueSize)
}

// Stop stops the maintenance worker pool.
func (s *SubscriptionService) Stop() {
	if s == nil {
		return
	}
	if s.maintenanceQueue != nil {
		s.maintenanceQueue.Stop()
	}
}

// initSubCache 初始化订阅 L1 缓存
func (s *SubscriptionService) initSubCache(cfg *config.Config) {
	if cfg == nil {
		return
	}
	sc := cfg.SubscriptionCache
	if sc.L1Size <= 0 || sc.L1TTLSeconds <= 0 {
		return
	}
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: int64(sc.L1Size) * 10,
		MaxCost:     int64(sc.L1Size),
		BufferItems: 64,
	})
	if err != nil {
		log.Printf("Warning: failed to init subscription L1 cache: %v", err)
		return
	}
	s.subCacheL1 = cache
	s.subCacheTTL = time.Duration(sc.L1TTLSeconds) * time.Second
	s.subCacheJitter = sc.JitterPercent
}

// subCacheKey 生成订阅缓存 key（热路径，避免 fmt.Sprintf 开销）
func subCacheKey(userID, groupID int64) string {
	return "sub:" + strconv.FormatInt(userID, 10) + ":" + strconv.FormatInt(groupID, 10)
}

func creditSubCacheKey(userID int64) string {
	return "credit-sub:" + strconv.FormatInt(userID, 10)
}

type creditSubCacheEntry struct {
	found bool
	sub   *UserSubscription
}

func (s *SubscriptionService) cacheCreditSubscriptionMiss(userID int64) {
	if s == nil || s.subCacheTTL <= 0 {
		return
	}
	s.creditSubMiss.Store(userID, time.Now().Add(s.jitteredTTL(s.subCacheTTL)))
}

func (s *SubscriptionService) hasCachedCreditSubscriptionMiss(userID int64) bool {
	if s == nil {
		return false
	}
	value, ok := s.creditSubMiss.Load(userID)
	if !ok {
		return false
	}
	expiresAt, ok := value.(time.Time)
	if !ok {
		s.creditSubMiss.Delete(userID)
		return false
	}
	if time.Now().Before(expiresAt) {
		return true
	}
	s.creditSubMiss.Delete(userID)
	return false
}

// jitteredTTL 为 TTL 添加抖动，避免集中过期
func (s *SubscriptionService) jitteredTTL(ttl time.Duration) time.Duration {
	if ttl <= 0 || s.subCacheJitter <= 0 {
		return ttl
	}
	pct := s.subCacheJitter
	if pct > 100 {
		pct = 100
	}
	delta := float64(pct) / 100
	factor := 1 - delta + rand.Float64()*(2*delta)
	if factor <= 0 {
		return ttl
	}
	return time.Duration(float64(ttl) * factor)
}

// subscriptionGroupIDOrZero 解引用订阅 GroupID；nil（额度池订阅）返回 0。
// 0 不会与真实分组 ID 冲突（分组 ID 由 BIGSERIAL 从 1 起），可安全用作缓存 key。
func subscriptionGroupIDOrZero(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

// InvalidateSubCache 失效指定用户+分组的订阅 L1 缓存
func (s *SubscriptionService) InvalidateSubCache(userID, groupID int64) {
	if s.subCacheL1 == nil {
		return
	}
	s.subCacheL1.Del(subCacheKey(userID, groupID))
	s.subCacheL1.Del(creditSubCacheKey(userID))
	s.creditSubMiss.Delete(userID)
}

// InvalidateSubCacheSync flushes ristretto's asynchronous delete buffer before
// callers reload a fresh database snapshot.
func (s *SubscriptionService) InvalidateSubCacheSync(userID, groupID int64) {
	s.InvalidateSubCache(userID, groupID)
	if s.subCacheL1 != nil {
		s.subCacheL1.Wait()
	}
}

// AssignSubscriptionInput 分配订阅输入
type AssignSubscriptionInput struct {
	UserID       int64
	GroupID      int64
	ValidityDays int
	AssignedBy   int64
	Notes        string
}

// AssignSubscription 分配订阅给用户（不允许重复分配）
func (s *SubscriptionService) AssignSubscription(ctx context.Context, input *AssignSubscriptionInput) (*UserSubscription, error) {
	sub, _, err := s.assignSubscriptionWithReuse(ctx, input)
	if err != nil {
		return nil, err
	}
	return sub, nil
}

// AssignOrExtendSubscription 分配或续期订阅（用于兑换码等场景）
// 如果用户已有同分组的订阅：
//   - 未过期：从当前过期时间累加天数
//   - 已过期：从当前时间开始计算新的过期时间，并激活订阅
//
// 如果没有订阅：创建新订阅
func (s *SubscriptionService) AssignOrExtendSubscription(ctx context.Context, input *AssignSubscriptionInput) (*UserSubscription, bool, error) {
	return s.assignOrExtendSubscription(ctx, input, false)
}

func (s *SubscriptionService) assignOrExtendSubscription(ctx context.Context, input *AssignSubscriptionInput, deferCacheInvalidation bool) (*UserSubscription, bool, error) {
	// 检查分组是否存在且为订阅类型
	group, err := s.groupRepo.GetByID(ctx, input.GroupID)
	if err != nil {
		return nil, false, fmt.Errorf("group not found: %w", err)
	}
	if !group.IsSubscriptionType() {
		return nil, false, ErrGroupNotSubscriptionType
	}

	// 查询是否已有订阅
	existingSub, err := s.userSubRepo.GetByUserIDAndGroupID(ctx, input.UserID, input.GroupID)
	if err != nil {
		// 不存在记录是正常情况，其他错误需要返回
		existingSub = nil
	}

	validityDays := input.ValidityDays
	if validityDays <= 0 {
		validityDays = 30
	}
	if validityDays > MaxValidityDays {
		validityDays = MaxValidityDays
	}

	// 已有订阅，执行续期（在事务中完成所有更新）
	if existingSub != nil {
		now := time.Now()
		var newExpiresAt time.Time

		if existingSub.ExpiresAt.After(now) {
			// 未过期：从当前过期时间累加
			newExpiresAt = existingSub.ExpiresAt.AddDate(0, 0, validityDays)
		} else {
			// 已过期：从当前时间开始计算
			newExpiresAt = now.AddDate(0, 0, validityDays)
		}

		// 确保不超过最大过期时间
		if newExpiresAt.After(MaxExpiresAt) {
			newExpiresAt = MaxExpiresAt
		}

		if err := s.withSubscriptionUpdateTx(ctx, func(txCtx context.Context) error {
			if err := s.userSubRepo.ExtendExpiry(txCtx, existingSub.ID, newExpiresAt); err != nil {
				return fmt.Errorf("extend subscription: %w", err)
			}
			if existingSub.Status != SubscriptionStatusActive {
				if err := s.userSubRepo.UpdateStatus(txCtx, existingSub.ID, SubscriptionStatusActive); err != nil {
					return fmt.Errorf("update subscription status: %w", err)
				}
			}
			if input.Notes != "" {
				newNotes := existingSub.Notes
				if newNotes != "" {
					newNotes += "\n"
				}
				newNotes += input.Notes
				if err := s.userSubRepo.UpdateNotes(txCtx, existingSub.ID, newNotes); err != nil {
					return fmt.Errorf("update subscription notes: %w", err)
				}
			}
			return nil
		}); err != nil {
			return nil, false, err
		}

		// 失效订阅缓存
		s.maybeInvalidateAssignmentCaches(input.UserID, input.GroupID, deferCacheInvalidation)

		// 返回更新后的订阅
		sub, err := s.userSubRepo.GetByID(ctx, existingSub.ID)
		return sub, true, err // true 表示是续期
	}

	// 没有订阅，创建新订阅
	sub, err := s.createSubscription(ctx, input)
	if err != nil {
		return nil, false, err
	}

	// 失效订阅缓存
	s.maybeInvalidateAssignmentCaches(input.UserID, input.GroupID, deferCacheInvalidation)

	return sub, false, nil // false 表示是新建
}

func (s *SubscriptionService) maybeInvalidateAssignmentCaches(userID, groupID int64, deferred bool) {
	// Payment fulfillment owns an outer transaction and performs a synchronous
	// invalidation after commit. Invalidating inside that transaction can reload
	// the pre-commit subscription into cache.
	if deferred {
		return
	}

	s.InvalidateSubCache(userID, groupID)
	if s.billingCacheService != nil {
		go func() {
			cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = s.billingCacheService.InvalidateSubscription(cacheCtx, userID, groupID)
		}()
	}
}

func (s *SubscriptionService) withSubscriptionUpdateTx(ctx context.Context, fn func(context.Context) error) error {
	if dbent.TxFromContext(ctx) != nil || s.entClient == nil {
		return fn(ctx)
	}
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	txCtx := dbent.NewTxContext(ctx, tx)
	if err := fn(txCtx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// createSubscription 创建新订阅（内部方法）
func (s *SubscriptionService) createSubscription(ctx context.Context, input *AssignSubscriptionInput) (*UserSubscription, error) {
	validityDays := input.ValidityDays
	if validityDays <= 0 {
		validityDays = 30
	}
	if validityDays > MaxValidityDays {
		validityDays = MaxValidityDays
	}

	now := time.Now()
	expiresAt := now.AddDate(0, 0, validityDays)
	if expiresAt.After(MaxExpiresAt) {
		expiresAt = MaxExpiresAt
	}

	gid := input.GroupID
	sub := &UserSubscription{
		UserID:     input.UserID,
		GroupID:    &gid,
		StartsAt:   now,
		ExpiresAt:  expiresAt,
		Status:     SubscriptionStatusActive,
		AssignedAt: now,
		Notes:      input.Notes,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	// 只有当 AssignedBy > 0 时才设置（0 表示系统分配，如兑换码）
	if input.AssignedBy > 0 {
		sub.AssignedBy = &input.AssignedBy
	}

	if creator, ok := s.userSubRepo.(subscriptionUsableGuardedCreator); ok {
		created, err := creator.CreateSubscriptionWithUsableGuard(ctx, sub, s.multiplePurchasesEnabled(ctx))
		if err != nil {
			return nil, err
		}
		return s.userSubRepo.GetByID(ctx, created.ID)
	}
	if !s.multiplePurchasesEnabled(ctx) {
		hasUsable, err := s.userSubRepo.HasUsableCreditSubscription(ctx, input.UserID)
		if err != nil {
			return nil, err
		}
		if hasUsable {
			return nil, ErrAlreadyHasUsableSubscription
		}
	}
	if err := s.userSubRepo.Create(ctx, sub); err != nil {
		return nil, err
	}

	// 重新获取完整订阅信息（包含关联）
	return s.userSubRepo.GetByID(ctx, sub.ID)
}

type subscriptionUsableGuardedCreator interface {
	CreateSubscriptionWithUsableGuard(ctx context.Context, sub *UserSubscription, allowMultiple bool) (*UserSubscription, error)
}

// BulkAssignSubscriptionInput 批量分配订阅输入
type BulkAssignSubscriptionInput struct {
	UserIDs      []int64
	GroupID      int64
	ValidityDays int
	AssignedBy   int64
	Notes        string
}

// BulkAssignResult 批量分配结果
type BulkAssignResult struct {
	SuccessCount  int
	CreatedCount  int
	ReusedCount   int
	FailedCount   int
	Subscriptions []UserSubscription
	Errors        []string
	Statuses      map[int64]string
}

// BulkAssignSubscription 批量分配订阅
func (s *SubscriptionService) BulkAssignSubscription(ctx context.Context, input *BulkAssignSubscriptionInput) (*BulkAssignResult, error) {
	result := &BulkAssignResult{
		Subscriptions: make([]UserSubscription, 0),
		Errors:        make([]string, 0),
		Statuses:      make(map[int64]string),
	}

	for _, userID := range input.UserIDs {
		sub, reused, err := s.assignSubscriptionWithReuse(ctx, &AssignSubscriptionInput{
			UserID:       userID,
			GroupID:      input.GroupID,
			ValidityDays: input.ValidityDays,
			AssignedBy:   input.AssignedBy,
			Notes:        input.Notes,
		})
		if err != nil {
			result.FailedCount++
			result.Errors = append(result.Errors, fmt.Sprintf("user %d: %v", userID, err))
			result.Statuses[userID] = "failed"
		} else {
			result.SuccessCount++
			result.Subscriptions = append(result.Subscriptions, *sub)
			if reused {
				result.ReusedCount++
				result.Statuses[userID] = "reused"
			} else {
				result.CreatedCount++
				result.Statuses[userID] = "created"
			}
		}
	}

	return result, nil
}

func (s *SubscriptionService) assignSubscriptionWithReuse(ctx context.Context, input *AssignSubscriptionInput) (*UserSubscription, bool, error) {
	// 检查分组是否存在且为订阅类型
	group, err := s.groupRepo.GetByID(ctx, input.GroupID)
	if err != nil {
		return nil, false, fmt.Errorf("group not found: %w", err)
	}
	if !group.IsSubscriptionType() {
		return nil, false, ErrGroupNotSubscriptionType
	}

	// 检查是否已存在订阅；若已存在，则按幂等成功返回现有订阅
	exists, err := s.userSubRepo.ExistsByUserIDAndGroupID(ctx, input.UserID, input.GroupID)
	if err != nil {
		return nil, false, err
	}
	if exists {
		sub, getErr := s.userSubRepo.GetByUserIDAndGroupID(ctx, input.UserID, input.GroupID)
		if getErr != nil {
			return nil, false, getErr
		}
		if conflictReason, conflict := detectAssignSemanticConflict(sub, input); conflict {
			return nil, false, ErrSubscriptionAssignConflict.WithMetadata(map[string]string{
				"conflict_reason": conflictReason,
			})
		}
		return sub, true, nil
	}

	sub, err := s.createSubscription(ctx, input)
	if err != nil {
		return nil, false, err
	}

	// 失效订阅缓存
	s.InvalidateSubCache(input.UserID, input.GroupID)
	if s.billingCacheService != nil {
		userID, groupID := input.UserID, input.GroupID
		go func() {
			cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = s.billingCacheService.InvalidateSubscription(cacheCtx, userID, groupID)
		}()
	}

	return sub, false, nil
}

func detectAssignSemanticConflict(existing *UserSubscription, input *AssignSubscriptionInput) (string, bool) {
	if existing == nil || input == nil {
		return "", false
	}

	normalizedDays := normalizeAssignValidityDays(input.ValidityDays)
	if !existing.StartsAt.IsZero() {
		expectedExpiresAt := existing.StartsAt.AddDate(0, 0, normalizedDays)
		if expectedExpiresAt.After(MaxExpiresAt) {
			expectedExpiresAt = MaxExpiresAt
		}
		if !existing.ExpiresAt.Equal(expectedExpiresAt) {
			return "validity_days_mismatch", true
		}
	}

	existingNotes := strings.TrimSpace(existing.Notes)
	inputNotes := strings.TrimSpace(input.Notes)
	if existingNotes != inputNotes {
		return "notes_mismatch", true
	}

	return "", false
}

func normalizeAssignValidityDays(days int) int {
	if days <= 0 {
		days = 30
	}
	if days > MaxValidityDays {
		days = MaxValidityDays
	}
	return days
}

// RevokeSubscription 撤销订阅
func (s *SubscriptionService) RevokeSubscription(ctx context.Context, subscriptionID int64) error {
	// 先获取订阅信息用于失效缓存
	sub, err := s.userSubRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return err
	}

	if err := s.userSubRepo.Delete(ctx, subscriptionID); err != nil {
		return err
	}

	// 失效订阅缓存（额度池订阅 group_id 可为 nil，传 0 表示无分组）
	subGroupID := subscriptionGroupIDOrZero(sub.GroupID)
	s.InvalidateSubCache(sub.UserID, subGroupID)
	if s.billingCacheService != nil {
		userID, groupID := sub.UserID, subGroupID
		go func() {
			cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = s.billingCacheService.InvalidateSubscription(cacheCtx, userID, groupID)
		}()
	}

	return nil
}

// ExtendSubscription 调整订阅时长（正数延长，负数缩短）
func (s *SubscriptionService) ExtendSubscription(ctx context.Context, subscriptionID int64, days int) (*UserSubscription, error) {
	sub, err := s.userSubRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, ErrSubscriptionNotFound
	}

	// 限制调整天数范围
	if days > MaxValidityDays {
		days = MaxValidityDays
	}
	if days < -MaxValidityDays {
		days = -MaxValidityDays
	}

	now := time.Now()
	isExpired := !sub.ExpiresAt.After(now)

	// 如果订阅已过期，不允许负向调整
	if isExpired && days < 0 {
		return nil, infraerrors.BadRequest("CANNOT_SHORTEN_EXPIRED", "cannot shorten an expired subscription")
	}

	// 计算新的过期时间
	var newExpiresAt time.Time
	if isExpired {
		// 已过期：从当前时间开始增加天数
		newExpiresAt = now.AddDate(0, 0, days)
	} else {
		// 未过期：从原过期时间增加/减少天数
		newExpiresAt = sub.ExpiresAt.AddDate(0, 0, days)
	}

	if newExpiresAt.After(MaxExpiresAt) {
		newExpiresAt = MaxExpiresAt
	}

	// 检查新的过期时间必须大于当前时间
	if !newExpiresAt.After(now) {
		return nil, ErrAdjustWouldExpire
	}

	if err := s.userSubRepo.ExtendExpiry(ctx, subscriptionID, newExpiresAt); err != nil {
		return nil, err
	}

	// 如果订阅已过期，恢复为active状态
	if sub.Status == SubscriptionStatusExpired {
		if err := s.userSubRepo.UpdateStatus(ctx, subscriptionID, SubscriptionStatusActive); err != nil {
			return nil, err
		}
	}

	// 失效订阅缓存（额度池订阅 group_id 可为 nil，传 0 表示无分组）
	subGroupID := subscriptionGroupIDOrZero(sub.GroupID)
	s.InvalidateSubCache(sub.UserID, subGroupID)
	if s.billingCacheService != nil {
		userID, groupID := sub.UserID, subGroupID
		go func() {
			cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = s.billingCacheService.InvalidateSubscription(cacheCtx, userID, groupID)
		}()
	}

	return s.userSubRepo.GetByID(ctx, subscriptionID)
}

// GetByID 根据ID获取订阅
func (s *SubscriptionService) GetByID(ctx context.Context, id int64) (*UserSubscription, error) {
	return s.userSubRepo.GetByID(ctx, id)
}

// GetActiveSubscription 获取用户对特定分组的有效订阅
// 使用 L1 缓存 + singleflight 加速中间件热路径。
// 返回缓存对象的浅拷贝，调用方可安全修改字段而不会污染缓存或触发 data race。
func (s *SubscriptionService) GetActiveSubscription(ctx context.Context, userID, groupID int64) (*UserSubscription, error) {
	key := subCacheKey(userID, groupID)

	// L1 缓存命中：返回浅拷贝
	if s.subCacheL1 != nil {
		if v, ok := s.subCacheL1.Get(key); ok {
			if sub, ok := v.(*UserSubscription); ok {
				cp := *sub
				return &cp, nil
			}
		}
	}

	// singleflight 防止并发击穿
	value, err, _ := s.subCacheGroup.Do(key, func() (any, error) {
		sub, err := s.userSubRepo.GetActiveByUserIDAndGroupID(ctx, userID, groupID)
		if err != nil {
			return nil, err // 直接透传 repo 已翻译的错误（NotFound → ErrSubscriptionNotFound，其他错误原样返回）
		}
		// 写入 L1 缓存
		if s.subCacheL1 != nil {
			_ = s.subCacheL1.SetWithTTL(key, sub, 1, s.jitteredTTL(s.subCacheTTL))
		}
		return sub, nil
	})
	if err != nil {
		return nil, err
	}
	// singleflight 返回的也是缓存指针，需要浅拷贝
	sub, ok := value.(*UserSubscription)
	if !ok || sub == nil {
		return nil, ErrSubscriptionNotFound
	}
	cp := *sub
	return &cp, nil
}

// GetUsableCreditSubscription 返回用户当前最早创建的可消费额度池订阅。
// Scope 与窗口限额由调用方结合实际请求分组继续检查。
func (s *SubscriptionService) GetUsableCreditSubscription(ctx context.Context, userID int64) (*UserSubscription, error) {
	if s == nil || s.userSubRepo == nil {
		return nil, ErrSubscriptionNotFound
	}
	key := creditSubCacheKey(userID)
	if s.hasCachedCreditSubscriptionMiss(userID) {
		return nil, ErrSubscriptionNotFound
	}
	if s.subCacheL1 != nil {
		if v, ok := s.subCacheL1.Get(key); ok {
			switch cached := v.(type) {
			case creditSubCacheEntry:
				if !cached.found || cached.sub == nil {
					return nil, ErrSubscriptionNotFound
				}
				cp := *cached.sub
				return &cp, nil
			case *creditSubCacheEntry:
				if cached == nil || !cached.found || cached.sub == nil {
					return nil, ErrSubscriptionNotFound
				}
				cp := *cached.sub
				return &cp, nil
			}
		}
	}

	value, err, _ := s.subCacheGroup.Do(key, func() (any, error) {
		sub, err := s.userSubRepo.GetUsableCreditSubscription(ctx, userID)
		if err != nil {
			if errors.Is(err, ErrSubscriptionNotFound) {
				s.cacheCreditSubscriptionMiss(userID)
				return &creditSubCacheEntry{}, nil
			}
			return nil, err
		}
		s.creditSubMiss.Delete(userID)
		entry := &creditSubCacheEntry{found: true, sub: sub}
		if s.subCacheL1 != nil {
			_ = s.subCacheL1.SetWithTTL(key, entry, 1, s.jitteredTTL(s.subCacheTTL))
			s.subCacheL1.Wait()
		}
		return entry, nil
	})
	if err != nil {
		return nil, err
	}
	entry, ok := value.(*creditSubCacheEntry)
	if !ok || entry == nil || !entry.found || entry.sub == nil {
		return nil, ErrSubscriptionNotFound
	}
	cp := *entry.sub
	return &cp, nil
}

func (s *SubscriptionService) ListUsableCreditSubscriptions(ctx context.Context, userID int64) ([]UserSubscription, error) {
	if s == nil || s.userSubRepo == nil {
		return nil, nil
	}
	return s.userSubRepo.ListUsableCreditSubscriptions(ctx, userID)
}

// GetRenewalEligibility 返回用户是否允许购买新的额度池订阅。
func (s *SubscriptionService) GetRenewalEligibility(ctx context.Context, userID int64) (RenewalEligibility, error) {
	if s == nil || s.userSubRepo == nil {
		return RenewalEligibility{Allowed: true, Reason: RenewalReasonNoSubscription}, nil
	}
	return s.userSubRepo.GetRenewalEligibility(ctx, userID)
}

// ListUserSubscriptions 获取用户的所有订阅
func (s *SubscriptionService) ListUserSubscriptions(ctx context.Context, userID int64) ([]UserSubscription, error) {
	subs, err := s.userSubRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	normalizeExpiredWindows(subs)
	normalizeSubscriptionStatus(subs)
	return subs, nil
}

// ListActiveUserSubscriptions 获取用户的所有有效订阅
func (s *SubscriptionService) ListActiveUserSubscriptions(ctx context.Context, userID int64) ([]UserSubscription, error) {
	subs, err := s.userSubRepo.ListActiveByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	normalizeExpiredWindows(subs)
	return subs, nil
}

// ListGroupSubscriptions 获取分组的所有订阅
func (s *SubscriptionService) ListGroupSubscriptions(ctx context.Context, groupID int64, page, pageSize int) ([]UserSubscription, *pagination.PaginationResult, error) {
	params := pagination.PaginationParams{Page: page, PageSize: pageSize}
	subs, pag, err := s.userSubRepo.ListByGroupID(ctx, groupID, params)
	if err != nil {
		return nil, nil, err
	}
	normalizeExpiredWindows(subs)
	normalizeSubscriptionStatus(subs)
	return subs, pag, nil
}

// List 获取所有订阅（分页，支持筛选和排序）
func (s *SubscriptionService) List(ctx context.Context, page, pageSize int, filters UserSubscriptionListFilters) ([]UserSubscription, *pagination.PaginationResult, error) {
	params := pagination.PaginationParams{Page: page, PageSize: pageSize}
	subs, pag, err := s.userSubRepo.List(ctx, params, filters)
	if err != nil {
		return nil, nil, err
	}
	normalizeExpiredWindows(subs)
	normalizeSubscriptionStatus(subs)
	return subs, pag, nil
}

type AdminUpdateSubscriptionInput struct {
	QuotaLimitUSD  *float64
	DailyLimitUSD  *float64
	WeeklyLimitUSD *float64
	ExpiresAt      *time.Time
	Status         *string
	Reason         string
}

func (s *SubscriptionService) AdminUpdateSubscription(ctx context.Context, subscriptionID int64, input AdminUpdateSubscriptionInput) (*UserSubscription, error) {
	if subscriptionID <= 0 {
		return nil, ErrSubscriptionNotFound
	}
	if err := validateAdminUpdateSubscriptionInput(input); err != nil {
		return nil, err
	}
	if s.entClient != nil {
		return s.adminUpdateSubscriptionInTransaction(ctx, subscriptionID, input)
	}
	return s.adminUpdateSubscriptionWithoutTransaction(ctx, subscriptionID, input)
}

func (s *SubscriptionService) adminUpdateSubscriptionInTransaction(ctx context.Context, subscriptionID int64, input AdminUpdateSubscriptionInput) (*UserSubscription, error) {
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, err
	}
	txCtx := dbent.NewTxContext(ctx, tx)
	sub, err := s.applyAdminSubscriptionUpdate(txCtx, subscriptionID, input, false, func(entry *SubscriptionCreditLedgerEntry) error {
		return createSubscriptionCreditLedgerWithEnt(txCtx, tx.Client(), entry)
	})
	if err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return nil, fmt.Errorf("%w; rollback failed: %v", err, rollbackErr)
		}
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	s.invalidateUpdatedSubscription(ctx, sub)
	return s.userSubRepo.GetByID(ctx, subscriptionID)
}

func (s *SubscriptionService) adminUpdateSubscriptionWithoutTransaction(ctx context.Context, subscriptionID int64, input AdminUpdateSubscriptionInput) (*UserSubscription, error) {
	sub, err := s.applyAdminSubscriptionUpdate(ctx, subscriptionID, input, true, func(entry *SubscriptionCreditLedgerEntry) error {
		return s.creditLedgerRepo.Create(ctx, nil, entry)
	})
	if err != nil {
		return nil, err
	}
	s.invalidateUpdatedSubscription(ctx, sub)
	return s.userSubRepo.GetByID(ctx, subscriptionID)
}

func (s *SubscriptionService) applyAdminSubscriptionUpdate(
	ctx context.Context,
	subscriptionID int64,
	input AdminUpdateSubscriptionInput,
	writeLedgerBeforeUpdate bool,
	writeLedger func(*SubscriptionCreditLedgerEntry) error,
) (*UserSubscription, error) {
	sub, err := s.userSubRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}
	oldLimit := sub.QuotaLimitUSD
	quotaChanged := false
	if input.QuotaLimitUSD != nil && *input.QuotaLimitUSD != sub.QuotaLimitUSD {
		sub.QuotaLimitUSD = *input.QuotaLimitUSD
		quotaChanged = true
		if sub.QuotaUsedUSD >= sub.QuotaLimitUSD && sub.ExhaustedAt == nil {
			now := time.Now()
			sub.ExhaustedAt = &now
		}
		if sub.QuotaUsedUSD < sub.QuotaLimitUSD && sub.ExhaustedAt != nil {
			sub.ExhaustedAt = nil
		}
	}
	if input.DailyLimitUSD != nil {
		sub.DailyLimitUSD = input.DailyLimitUSD
	}
	if input.WeeklyLimitUSD != nil {
		sub.WeeklyLimitUSD = input.WeeklyLimitUSD
	}
	if input.ExpiresAt != nil {
		sub.ExpiresAt = *input.ExpiresAt
	}
	if input.Status != nil {
		sub.Status = *input.Status
	}

	var ledgerEntry *SubscriptionCreditLedgerEntry
	if quotaChanged {
		var err error
		ledgerEntry, err = s.adminAdjustLedgerEntry(sub, oldLimit, input.Reason)
		if err != nil {
			return nil, err
		}
	}
	if ledgerEntry != nil && writeLedgerBeforeUpdate {
		if err := writeLedger(ledgerEntry); err != nil {
			return nil, err
		}
	}

	if err := s.userSubRepo.Update(ctx, sub); err != nil {
		return nil, err
	}

	if ledgerEntry != nil && !writeLedgerBeforeUpdate {
		if err := writeLedger(ledgerEntry); err != nil {
			return nil, err
		}
	}
	return sub, nil
}

func (s *SubscriptionService) adminAdjustLedgerEntry(sub *UserSubscription, oldLimit float64, reason string) (*SubscriptionCreditLedgerEntry, error) {
	if strings.TrimSpace(reason) == "" {
		return nil, infraerrors.BadRequest("ADMIN_ADJUST_REASON_REQUIRED", "reason is required when adjusting subscription quota")
	}
	if s.creditLedgerRepo == nil && s.entClient == nil {
		return nil, infraerrors.InternalServer("SUBSCRIPTION_LEDGER_REPOSITORY_UNAVAILABLE", "subscription credit ledger repository is not configured")
	}
	delta := sub.QuotaLimitUSD - oldLimit
	return &SubscriptionCreditLedgerEntry{
		UserID:            sub.UserID,
		SubscriptionID:    sub.ID,
		GroupID:           sub.GroupID,
		Type:              SubscriptionCreditLedgerAdminAdjust,
		DeltaUSD:          delta,
		RemainingAfterUSD: math.Max(sub.QuotaLimitUSD-sub.QuotaUsedUSD, 0),
		Reason:            reason,
		Metadata: map[string]any{
			"old_quota_limit_usd": oldLimit,
			"new_quota_limit_usd": sub.QuotaLimitUSD,
			"quota_used_usd":      sub.QuotaUsedUSD,
		},
	}, nil
}

func createSubscriptionCreditLedgerWithEnt(ctx context.Context, client *dbent.Client, entry *SubscriptionCreditLedgerEntry) error {
	if client == nil {
		return infraerrors.InternalServer("ENT_CLIENT_UNAVAILABLE", "ent client is not configured")
	}
	if entry == nil {
		return ErrSubscriptionNilInput
	}
	builder := client.SubscriptionCreditLedger.Create().
		SetUserID(entry.UserID).
		SetSubscriptionID(entry.SubscriptionID).
		SetNillableGroupID(entry.GroupID).
		SetNillableAPIKeyID(entry.APIKeyID).
		SetNillableUsageLogID(entry.UsageLogID).
		SetNillableOrderID(entry.OrderID).
		SetType(entry.Type).
		SetDeltaUsd(entry.DeltaUSD).
		SetBalanceDeltaUsd(entry.BalanceDeltaUSD).
		SetRemainingAfterUsd(entry.RemainingAfterUSD).
		SetReason(entry.Reason).
		SetNillableEventKey(entry.EventKey)
	if entry.Metadata != nil {
		builder.SetMetadata(entry.Metadata)
	}
	return builder.Exec(ctx)
}

func (s *SubscriptionService) invalidateUpdatedSubscription(ctx context.Context, sub *UserSubscription) {
	if sub == nil {
		return
	}
	gid := subscriptionGroupIDOrZero(sub.GroupID)
	s.InvalidateSubCache(sub.UserID, gid)
	if s.billingCacheService != nil {
		_ = s.billingCacheService.InvalidateSubscription(ctx, sub.UserID, gid)
	}
}

func validateAdminUpdateSubscriptionInput(input AdminUpdateSubscriptionInput) error {
	if input.QuotaLimitUSD != nil && *input.QuotaLimitUSD <= 0 {
		return infraerrors.BadRequest("INVALID_SUBSCRIPTION_QUOTA_LIMIT", "quota_limit_usd must be greater than 0")
	}
	if input.DailyLimitUSD != nil && *input.DailyLimitUSD < 0 {
		return infraerrors.BadRequest("INVALID_SUBSCRIPTION_DAILY_LIMIT", "daily_limit_usd must be greater than or equal to 0")
	}
	if input.WeeklyLimitUSD != nil && *input.WeeklyLimitUSD < 0 {
		return infraerrors.BadRequest("INVALID_SUBSCRIPTION_WEEKLY_LIMIT", "weekly_limit_usd must be greater than or equal to 0")
	}
	if input.Status != nil && !isValidAdminSubscriptionStatus(*input.Status) {
		return infraerrors.BadRequest("INVALID_SUBSCRIPTION_STATUS", "status must be active, expired, suspended, or revoked")
	}
	return nil
}

func isValidAdminSubscriptionStatus(status string) bool {
	switch status {
	case SubscriptionStatusActive, SubscriptionStatusExpired, SubscriptionStatusSuspended, "revoked":
		return true
	default:
		return false
	}
}

func (s *SubscriptionService) ListSubscriptionLedger(ctx context.Context, subscriptionID int64, ledgerType string, page, pageSize int) ([]SubscriptionCreditLedgerEntry, *pagination.PaginationResult, error) {
	if subscriptionID <= 0 {
		return nil, nil, ErrSubscriptionNotFound
	}
	if s.creditLedgerRepo == nil {
		return nil, nil, infraerrors.InternalServer("SUBSCRIPTION_LEDGER_REPOSITORY_UNAVAILABLE", "subscription credit ledger repository is not configured")
	}
	params := pagination.PaginationParams{Page: page, PageSize: pageSize}
	return s.creditLedgerRepo.ListBySubscriptionID(ctx, subscriptionID, ledgerType, params)
}

// normalizeExpiredWindows 将已过期窗口的数据清零（仅影响返回数据，不影响数据库）
// 这确保前端显示正确的当前窗口状态，而不是过期窗口的历史数据
func normalizeExpiredWindows(subs []UserSubscription) {
	for i := range subs {
		sub := &subs[i]
		// 日窗口过期：清零展示数据
		if sub.NeedsDailyReset() {
			sub.DailyWindowStart = nil
			sub.DailyUsageUSD = 0
		}
		// 周窗口过期：清零展示数据
		if sub.NeedsWeeklyReset() {
			sub.WeeklyWindowStart = nil
			sub.WeeklyUsageUSD = 0
		}
		// 月窗口过期：清零展示数据
		if sub.NeedsMonthlyReset() {
			sub.MonthlyWindowStart = nil
			sub.MonthlyUsageUSD = 0
		}
	}
}

// normalizeSubscriptionStatus 根据实际过期时间修正状态（仅影响返回数据，不影响数据库）
// 这确保前端显示正确的状态，即使定时任务尚未更新数据库
func normalizeSubscriptionStatus(subs []UserSubscription) {
	now := time.Now()
	for i := range subs {
		sub := &subs[i]
		if sub.Status == SubscriptionStatusActive && !sub.ExpiresAt.After(now) {
			sub.Status = SubscriptionStatusExpired
		}
	}
}

// startOfDay 返回给定时间所在日期的零点（保持原时区）
func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// CheckAndActivateWindow 检查并激活窗口（首次使用时）
func (s *SubscriptionService) CheckAndActivateWindow(ctx context.Context, sub *UserSubscription) error {
	if sub.IsWindowActivated() {
		return nil
	}

	// 使用当天零点作为窗口起始时间
	windowStart := startOfDay(time.Now())
	return s.userSubRepo.ActivateWindows(ctx, sub.ID, windowStart)
}

// AdminResetQuota manually resets the daily, weekly, and/or monthly usage windows.
// Uses startOfDay(now) as the new window start, matching automatic resets.
func (s *SubscriptionService) AdminResetQuota(ctx context.Context, subscriptionID int64, resetDaily, resetWeekly, resetMonthly bool) (*UserSubscription, error) {
	if !resetDaily && !resetWeekly && !resetMonthly {
		return nil, ErrInvalidInput
	}
	sub, err := s.userSubRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}
	windowStart := startOfDay(time.Now())
	if err := s.userSubRepo.ResetUsageWindows(ctx, sub.ID, resetDaily, resetWeekly, resetMonthly, windowStart); err != nil {
		return nil, err
	}
	// Invalidate L1 ristretto cache. Ristretto's Del() is asynchronous by design,
	// so call Wait() immediately after to flush pending operations and guarantee
	// the deleted key is not returned on the very next Get() call.
	gid := subscriptionGroupIDOrZero(sub.GroupID)
	s.InvalidateSubCache(sub.UserID, gid)
	if s.subCacheL1 != nil {
		s.subCacheL1.Wait()
	}
	if s.billingCacheService != nil {
		_ = s.billingCacheService.InvalidateSubscription(ctx, sub.UserID, gid)
	}
	// Return the refreshed subscription from DB
	return s.userSubRepo.GetByID(ctx, subscriptionID)
}

// CheckAndResetWindows 检查并重置过期的窗口
func (s *SubscriptionService) CheckAndResetWindows(ctx context.Context, sub *UserSubscription) error {
	// 使用当天零点作为新窗口起始时间
	windowStart := startOfDay(time.Now())
	needsInvalidateCache := false

	// 日窗口重置（24小时）
	if sub.NeedsDailyReset() {
		expectedWindowStart := sub.DailyWindowStart
		if err := s.userSubRepo.ResetDailyUsage(ctx, sub.ID, expectedWindowStart, windowStart); err != nil {
			return err
		}
		sub.DailyWindowStart = &windowStart
		sub.DailyUsageUSD = 0
		needsInvalidateCache = true
	}

	// 周窗口重置（7天）
	if sub.NeedsWeeklyReset() {
		expectedWindowStart := sub.WeeklyWindowStart
		if err := s.userSubRepo.ResetWeeklyUsage(ctx, sub.ID, expectedWindowStart, windowStart); err != nil {
			return err
		}
		sub.WeeklyWindowStart = &windowStart
		sub.WeeklyUsageUSD = 0
		needsInvalidateCache = true
	}

	// 月窗口重置（30天）
	if sub.NeedsMonthlyReset() {
		expectedWindowStart := sub.MonthlyWindowStart
		if err := s.userSubRepo.ResetMonthlyUsage(ctx, sub.ID, expectedWindowStart, windowStart); err != nil {
			return err
		}
		sub.MonthlyWindowStart = &windowStart
		sub.MonthlyUsageUSD = 0
		needsInvalidateCache = true
	}

	// 如果有窗口被重置，失效缓存以保持一致性
	if needsInvalidateCache {
		gid := subscriptionGroupIDOrZero(sub.GroupID)
		s.InvalidateSubCache(sub.UserID, gid)
		if s.billingCacheService != nil {
			_ = s.billingCacheService.InvalidateSubscription(ctx, sub.UserID, gid)
		}
	}

	return nil
}

// EnsureWindowMaintenance advances expired usage windows before a request is
// allowed to proceed. It returns a fresh database snapshot because a competing
// request may have won one of the conditional resets.
func (s *SubscriptionService) EnsureWindowMaintenance(ctx context.Context, sub *UserSubscription) (*UserSubscription, error) {
	if sub == nil {
		return nil, ErrSubscriptionNilInput
	}
	if !sub.IsWindowActivated() {
		if err := s.CheckAndActivateWindow(ctx, sub); err != nil {
			return nil, err
		}
	}
	if err := s.CheckAndResetWindows(ctx, sub); err != nil {
		return nil, err
	}

	// GetByID bypasses the service caches. This prevents a stale loser of the
	// CAS from validating limits against zeroed in-memory usage.
	refreshed, err := s.userSubRepo.GetByID(ctx, sub.ID)
	if err != nil {
		return nil, err
	}
	s.InvalidateSubCacheSync(sub.UserID, subscriptionGroupIDOrZero(sub.GroupID))
	return refreshed, nil
}

// CheckUsageLimits 检查使用限额（返回错误如果超限）
// 用于中间件的快速预检查，additionalCost 通常为 0
func (s *SubscriptionService) CheckUsageLimits(ctx context.Context, sub *UserSubscription, group *Group, additionalCost float64) error {
	if !sub.CheckDailyLimit(group, additionalCost) {
		return ErrDailyLimitExceeded
	}
	if !sub.CheckWeeklyLimit(group, additionalCost) {
		return ErrWeeklyLimitExceeded
	}
	if !sub.CheckMonthlyLimit(group, additionalCost) {
		return ErrMonthlyLimitExceeded
	}
	return nil
}

// ValidateAndCheckLimits 合并验证+限额检查（中间件热路径专用）
// 仅做内存检查，不触发 DB 写入。调用方可对已激活且过期的窗口同步推进并
// 重新读取数据库快照，避免 CAS 失败者基于本地清零值误判限额；全 nil 窗口
// 仍留给扣费事务首次激活。
func (s *SubscriptionService) ValidateAndCheckLimits(sub *UserSubscription, group *Group) (needsMaintenance bool, err error) {
	// 1. 验证订阅状态
	if sub.Status == SubscriptionStatusExpired {
		return false, ErrSubscriptionExpired
	}
	if sub.Status == SubscriptionStatusSuspended {
		return false, ErrSubscriptionSuspended
	}
	if sub.IsExpired() {
		return false, ErrSubscriptionExpired
	}
	// 总额度池耗尽：订阅作废，回落到余额闸门（ErrSubscriptionInvalid 属于可回落错误）。
	// 额度池耗尽后日/周窗口用量会冻结在上限以下，单靠窗口检查无法拦截，必须显式判定总池。
	// 仅对配置了正总池的订阅生效；QuotaLimitUSD == 0 的纯窗口订阅剩余恒为 0，不算耗尽。
	if sub.IsCreditPoolExhausted() {
		return false, ErrSubscriptionInvalid
	}

	// 2. 内存中修正过期窗口的用量，确保 CheckUsageLimits 不会误拒绝用户。
	//    实际的 DB 窗口重置由扣费事务完成。
	if sub.NeedsDailyReset() {
		sub.DailyUsageUSD = 0
		needsMaintenance = true
	}
	if sub.NeedsWeeklyReset() {
		sub.WeeklyUsageUSD = 0
		needsMaintenance = true
	}
	if sub.NeedsMonthlyReset() {
		sub.MonthlyUsageUSD = 0
		needsMaintenance = true
	}
	if !sub.IsWindowActivated() {
		needsMaintenance = true
	}

	// 3. 检查用量限额
	if !sub.CheckDailyLimit(group, 0) {
		return needsMaintenance, ErrDailyLimitExceeded
	}
	if !sub.CheckWeeklyLimit(group, 0) {
		return needsMaintenance, ErrWeeklyLimitExceeded
	}
	if !sub.CheckMonthlyLimit(group, 0) {
		return needsMaintenance, ErrMonthlyLimitExceeded
	}

	return needsMaintenance, nil
}

// RecordUsage 记录使用量到订阅
func (s *SubscriptionService) RecordUsage(ctx context.Context, subscriptionID int64, costUSD float64) error {
	return s.userSubRepo.IncrementUsage(ctx, subscriptionID, costUSD)
}

// SubscriptionProgress 订阅进度
type SubscriptionProgress struct {
	ID            int64                `json:"id"`
	GroupName     string               `json:"group_name"`
	ExpiresAt     time.Time            `json:"expires_at"`
	ExpiresInDays int                  `json:"expires_in_days"`
	Daily         *UsageWindowProgress `json:"daily,omitempty"`
	Weekly        *UsageWindowProgress `json:"weekly,omitempty"`
	Monthly       *UsageWindowProgress `json:"monthly,omitempty"`
}

// UsageWindowProgress 使用窗口进度
type UsageWindowProgress struct {
	LimitUSD        float64   `json:"limit_usd"`
	UsedUSD         float64   `json:"used_usd"`
	RemainingUSD    float64   `json:"remaining_usd"`
	Percentage      float64   `json:"percentage"`
	WindowStart     time.Time `json:"window_start"`
	ResetsAt        time.Time `json:"resets_at"`
	ResetsInSeconds int64     `json:"resets_in_seconds"`
}

// GetSubscriptionProgress 获取订阅使用进度
func (s *SubscriptionService) GetSubscriptionProgress(ctx context.Context, subscriptionID int64) (*SubscriptionProgress, error) {
	sub, err := s.userSubRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, ErrSubscriptionNotFound
	}

	group := sub.Group
	if group == nil && sub.GroupID != nil {
		group, err = s.groupRepo.GetByID(ctx, *sub.GroupID)
		if err != nil {
			return nil, err
		}
	}

	return s.calculateProgress(sub, group), nil
}

// calculateProgress 根据已加载的订阅和分组数据计算使用进度（纯内存计算，无 DB 查询）
func (s *SubscriptionService) calculateProgress(sub *UserSubscription, group *Group) *SubscriptionProgress {
	groupName := ""
	if group != nil {
		groupName = group.Name
	}
	progress := &SubscriptionProgress{
		ID:            sub.ID,
		GroupName:     groupName,
		ExpiresAt:     sub.ExpiresAt,
		ExpiresInDays: sub.DaysRemaining(),
	}

	// 日进度
	if limitPtr := sub.dailyLimitUSD(group); limitPtr != nil && sub.DailyWindowStart != nil {
		limit := *limitPtr
		resetsAt := sub.DailyWindowStart.Add(24 * time.Hour)
		progress.Daily = &UsageWindowProgress{
			LimitUSD:        limit,
			UsedUSD:         sub.DailyUsageUSD,
			RemainingUSD:    limit - sub.DailyUsageUSD,
			Percentage:      (sub.DailyUsageUSD / limit) * 100,
			WindowStart:     *sub.DailyWindowStart,
			ResetsAt:        resetsAt,
			ResetsInSeconds: int64(time.Until(resetsAt).Seconds()),
		}
		if progress.Daily.RemainingUSD < 0 {
			progress.Daily.RemainingUSD = 0
		}
		if progress.Daily.Percentage > 100 {
			progress.Daily.Percentage = 100
		}
		if progress.Daily.ResetsInSeconds < 0 {
			progress.Daily.ResetsInSeconds = 0
		}
	}

	// 周进度
	if limitPtr := sub.weeklyLimitUSD(group); limitPtr != nil && sub.WeeklyWindowStart != nil {
		limit := *limitPtr
		resetsAt := sub.WeeklyWindowStart.Add(7 * 24 * time.Hour)
		progress.Weekly = &UsageWindowProgress{
			LimitUSD:        limit,
			UsedUSD:         sub.WeeklyUsageUSD,
			RemainingUSD:    limit - sub.WeeklyUsageUSD,
			Percentage:      (sub.WeeklyUsageUSD / limit) * 100,
			WindowStart:     *sub.WeeklyWindowStart,
			ResetsAt:        resetsAt,
			ResetsInSeconds: int64(time.Until(resetsAt).Seconds()),
		}
		if progress.Weekly.RemainingUSD < 0 {
			progress.Weekly.RemainingUSD = 0
		}
		if progress.Weekly.Percentage > 100 {
			progress.Weekly.Percentage = 100
		}
		if progress.Weekly.ResetsInSeconds < 0 {
			progress.Weekly.ResetsInSeconds = 0
		}
	}

	// 月进度
	if group != nil && group.HasMonthlyLimit() && sub.MonthlyWindowStart != nil {
		limit := *group.MonthlyLimitUSD
		resetsAt := sub.MonthlyWindowStart.Add(30 * 24 * time.Hour)
		progress.Monthly = &UsageWindowProgress{
			LimitUSD:        limit,
			UsedUSD:         sub.MonthlyUsageUSD,
			RemainingUSD:    limit - sub.MonthlyUsageUSD,
			Percentage:      (sub.MonthlyUsageUSD / limit) * 100,
			WindowStart:     *sub.MonthlyWindowStart,
			ResetsAt:        resetsAt,
			ResetsInSeconds: int64(time.Until(resetsAt).Seconds()),
		}
		if progress.Monthly.RemainingUSD < 0 {
			progress.Monthly.RemainingUSD = 0
		}
		if progress.Monthly.Percentage > 100 {
			progress.Monthly.Percentage = 100
		}
		if progress.Monthly.ResetsInSeconds < 0 {
			progress.Monthly.ResetsInSeconds = 0
		}
	}

	return progress
}

// GetUserSubscriptionsWithProgress 获取用户所有订阅及进度
func (s *SubscriptionService) GetUserSubscriptionsWithProgress(ctx context.Context, userID int64) ([]SubscriptionProgress, error) {
	// ListActiveByUserID 已使用 .WithGroup() eager-load Group 关联，1 次查询获取所有数据
	subs, err := s.userSubRepo.ListActiveByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	progresses := make([]SubscriptionProgress, 0, len(subs))
	for i := range subs {
		sub := &subs[i]
		group := sub.Group
		if group == nil {
			continue
		}
		progresses = append(progresses, *s.calculateProgress(sub, group))
	}

	return progresses, nil
}

// ValidateSubscription 验证订阅是否有效
func (s *SubscriptionService) ValidateSubscription(ctx context.Context, sub *UserSubscription) error {
	if sub.Status == SubscriptionStatusExpired {
		return ErrSubscriptionExpired
	}
	if sub.Status == SubscriptionStatusSuspended {
		return ErrSubscriptionSuspended
	}
	if sub.IsExpired() {
		// 更新状态
		_ = s.userSubRepo.UpdateStatus(ctx, sub.ID, SubscriptionStatusExpired)
		return ErrSubscriptionExpired
	}
	return nil
}
