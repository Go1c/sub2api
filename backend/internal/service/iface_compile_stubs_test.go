//go:build unit

package service

import (
	"context"
	"time"
)

// Compile-only interface fillers for test stubs that predate methods added
// during the 0.2.17 merge. Behavior tests that need real side effects should
// replace the specific method instead of relying on these no-ops.
func (*contentModerationTestUserRepo) AdjustBalance(context.Context, int64, float64) (BalanceChange, error) {
	return BalanceChange{}, nil
}

func (*paymentFulfillmentAffiliateRepoStub) BindInviterWithSignupBonus(context.Context, AffiliateSignupBonusRequest) (*AffiliateSignupBonusResult, error) {
	return nil, nil
}


func (*availableGroupModelGroupRepo) DeleteCascadeIfEmpty(context.Context, int64) ([]int64, error) {
	return nil, nil
}

func (groupRepoNoop) DeleteCascadeIfEmpty(context.Context, int64) ([]int64, error) { return nil, nil }

func (*groupRepoStubForFallbackCycle) DeleteCascadeIfEmpty(context.Context, int64) ([]int64, error) {
	return nil, nil
}

func (*groupRepoStubForGroupUpdate) DeleteCascadeIfEmpty(context.Context, int64) ([]int64, error) {
	return nil, nil
}

func (*groupRepoStubForInvalidRequestFallback) DeleteCascadeIfEmpty(context.Context, int64) ([]int64, error) {
	return nil, nil
}

func (*mockGroupRepoForGateway) DeleteCascadeIfEmpty(context.Context, int64) ([]int64, error) {
	return nil, nil
}

func (*mockGroupRepoForGemini) DeleteCascadeIfEmpty(context.Context, int64) ([]int64, error) {
	return nil, nil
}

func (*stubGroupRepoForAvailable) DeleteCascadeIfEmpty(context.Context, int64) ([]int64, error) {
	return nil, nil
}

func (*subscriptionGroupRepoStub) DeleteCascadeIfEmpty(context.Context, int64) ([]int64, error) {
	return nil, nil
}

func (*apiKeyCacheStub) DeleteCreateAttemptCount(context.Context, int64) error { return nil }

func (*authCacheStub) DeleteCreateAttemptCount(context.Context, int64) error { return nil }

func (*createLimitCacheStub) DeleteCreateAttemptCount(context.Context, int64) error { return nil }

func (*quotaStateCacheStub) DeleteCreateAttemptCount(context.Context, int64) error { return nil }

func (*opsRepoMock) GetLatestRetryAttemptForError(context.Context, int64) (*OpsRetryAttempt, error) {
	return nil, nil
}

func (*resetTokenCacheStub) IncrNotifyVerifyCodeAttempts(context.Context, string) (int, error) {
	return 0, nil
}

func (*authInvalidationCacheStub) IncrementCreateCount(context.Context, int64, time.Duration) (int64, error) {
	return 0, nil
}

func (*accountRepoStub) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*accountRepoStubForAdminList) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*accountRepoStubForBulkUpdate) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*accountRepoStubForClearAccountError) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*accountRepoStubForCompositeModelsList) ResetQuotaUsed(context.Context, int64) error {
	return nil
}

func (*anthropicWindowLimitRepo) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*bindFailRepoStub) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*clineModelLimitFailRepo) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*commandCodeRateLimitRepo) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*countingModelRateLimitRepo) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*dbFallbackRepoStub) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*duplicateAccountRepoStub) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*epAccountRepo) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*epTrackingRepo) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*errorPolicyRepoStub) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*fableSchedulingThresholdRepoStub) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*geminiBoundCacheRateLimitRepo) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*geminiErrorPolicyRepo) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*geminiSingleCheckRepoStub) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*grokAccountTestRateLimitRepo) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*grokCredentialBlockingRepo) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*grokCredentialCommitThenCancelRepo) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*grokCredentialPersistingRepo) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*grokCredentialRaceRepo) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*grokCredentialRereadFailureRepo) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*grokCredentialSequencedRepo) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*grokCredentialUncommittedDeadlineRepo) ResetQuotaUsed(context.Context, int64) error {
	return nil
}

func (*grokQuotaAccountRepo) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*groupAwareMockAccountRepo) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*longContextBillingRepoStub) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*mockAccountRepoForGemini) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*mockAccountRepoForPlatform) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*modelNotFoundAccountRepoStub) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*ollama429Repo) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*openAI429SnapshotRepo) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*openAIAccountTestRepo) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*overloadAccountRepoStub) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*raceCreateRepoStub) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*rateLimit429AccountRepoStub) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*rateLimitAccountRepoStub) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*rateLimitClearRepoStub) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*refreshAPIAccountRepo) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*refreshAPIAccountRepoWithRace) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*resetAccountQuotaRepoStub) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*sparkShadowRepoStub) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*teamLinkedAccountRepoStub) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*tempUnschedScopeRepoStub) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*thresholdSelectionAccountRepoStub) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*tokenRefreshAccountRepo) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*updateAccountCredsRepoStub) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*updateAccountOveragesRepoStub) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*updateExtraSpyRepo) ResetQuotaUsed(context.Context, int64) error { return nil }

func (*affiliateSignupBonusRepoStub) WithdrawQuota(context.Context, int64, float64, string) (*AffiliateWithdrawResult, error) {
	return nil, nil
}

func (*adminUpdateUserSubRepoStub) GetByIDForUpdate(ctx context.Context, id int64) (*UserSubscription, error) {
	panic("test stub")
}

func (*adminUpdateUserSubRepoStub) GetByIDIncludeDeleted(ctx context.Context, id int64) (*UserSubscription, error) {
	panic("test stub")
}

func (*adminUpdateUserSubRepoStub) Restore(ctx context.Context, subscriptionID int64, restoredStatus string) (*UserSubscription, error) {
	panic("test stub")
}

func (*adminUpdateUserSubRepoStub) ExistsActiveByUserIDAndGroupID(ctx context.Context, userID, groupID int64) (bool, error) {
	panic("test stub")
}

func (*contentModerationTestUserRepo) SetBalance(ctx context.Context, id int64, value float64) (BalanceChange, error) {
	panic("test stub")
}

func (*contentModerationTestUserRepo) ExistsByEmailAlias(ctx context.Context, email string) (bool, error) {
	panic("test stub")
}

func (*contentModerationTestUserRepo) CreateWithEmailAliasGuard(context.Context, *User) error {
	return nil
}



func (*opsRepoMock) InsertRetryAttempt(ctx context.Context, input *OpsInsertRetryAttemptInput) (int64, error) {
	panic("test stub")
}

func (*opsRepoMock) UpdateRetryAttempt(ctx context.Context, input *OpsUpdateRetryAttemptInput) error {
	panic("test stub")
}

func (*opsRepoMock) ListRetryAttemptsByErrorID(ctx context.Context, sourceErrorID int64, limit int) ([]*OpsRetryAttempt, error) {
	panic("test stub")
}

func (*paymentFulfillmentAffiliateRepoStub) RecordInviteLog(ctx context.Context, entry AffiliateInviteLogEntry) error {
	panic("test stub")
}

func (*paymentFulfillmentAffiliateRepoStub) ListInviteLogs(ctx context.Context, filter AffiliateInviteLogFilter) ([]AffiliateInviteLog, int64, error) {
	panic("test stub")
}

func (*paymentFulfillmentAffiliateRepoStub) GetInviteeRechargeTotal(ctx context.Context, inviterID int64) (float64, error) {
	panic("test stub")
}

func (*resetTokenCacheStub) GetVerificationCode(ctx context.Context, email string) (*VerificationCodeData, error) {
	panic("test stub")
}

func (*resetTokenCacheStub) SetVerificationCode(ctx context.Context, email string, data *VerificationCodeData, ttl time.Duration) error {
	panic("test stub")
}

func (*resetTokenCacheStub) DeleteVerificationCode(ctx context.Context, email string) error {
	panic("test stub")
}

func (*resetTokenCacheStub) IncrVerificationCodeAttempts(ctx context.Context, email string) (int, error) {
	panic("test stub")
}

func (*resetTokenCacheStub) GetNotifyVerifyCode(ctx context.Context, email string) (*VerificationCodeData, error) {
	panic("test stub")
}

func (*resetTokenCacheStub) SetNotifyVerifyCode(ctx context.Context, email string, data *VerificationCodeData, ttl time.Duration) error {
	panic("test stub")
}

func (*resetTokenCacheStub) DeleteNotifyVerifyCode(ctx context.Context, email string) error {
	panic("test stub")
}

func (*resetTokenCacheStub) SetPasswordResetToken(ctx context.Context, email string, data *PasswordResetTokenData, ttl time.Duration) error {
	panic("test stub")
}

func (*resetTokenCacheStub) DeletePasswordResetToken(ctx context.Context, email string) error {
	panic("test stub")
}

func (*resetTokenCacheStub) IsPasswordResetEmailInCooldown(ctx context.Context, email string) bool {
	panic("test stub")
}

func (*resetTokenCacheStub) SetPasswordResetEmailCooldown(ctx context.Context, email string, ttl time.Duration) error {
	panic("test stub")
}

func (*resetTokenCacheStub) IncrNotifyCodeUserRate(ctx context.Context, userID int64, window time.Duration) (int64, error) {
	panic("test stub")
}

func (*resetTokenCacheStub) GetNotifyCodeUserRate(ctx context.Context, userID int64) (int64, error) {
	panic("test stub")
}
