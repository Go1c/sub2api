package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
)

const codexRecoveryReason = "Codex 2FA recovery: upstream 401; scheduling stopped"

type CodexLoginService struct {
	store       *codexLoginStore
	accounts    AccountRepository
	admin       AdminService
	encryptor   SecretEncryptor
	runner      CodexLoginRunner
	oauth       *OpenAIOAuthService
	tokenCache  OpenAITokenCache
	invalidator TokenCacheInvalidator
	rate        *RateLimitService
	cfg         *config.Config
	cancel      context.CancelFunc
	once        sync.Once
	wg          sync.WaitGroup
}

func ProvideCodexLoginService(db *sql.DB, accounts AccountRepository, admin AdminService, encryptor SecretEncryptor,
	oauth *OpenAIOAuthService, cache OpenAITokenCache, invalidator TokenCacheInvalidator, cfg *config.Config) *CodexLoginService {
	return &CodexLoginService{store: &codexLoginStore{db}, accounts: accounts, admin: admin, encryptor: encryptor, oauth: oauth,
		tokenCache: cache, invalidator: invalidator, cfg: cfg,
		runner: &HTTPCodexLoginRunner{URL: cfg.CodexLogin.WorkerURL, Token: cfg.CodexLogin.WorkerToken}}
}

func (s *CodexLoginService) Available() bool {
	return s != nil && s.cfg != nil && s.cfg.CodexLogin.WorkerURL != "" && len(s.cfg.CodexLogin.WorkerToken) >= 32 && s.cfg.Totp.EncryptionKeyConfigured
}

func (s *CodexLoginService) Start() {
	if !s.Available() {
		return
	}
	s.once.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		s.cancel = cancel
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			ticker := time.NewTicker(3 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					s.tick(ctx)
				}
			}
		}()
	})
}
func (s *CodexLoginService) Stop() {
	if s != nil && s.cancel != nil {
		s.cancel()
		s.wg.Wait()
	}
}

func (s *CodexLoginService) Attach(rate *RateLimitService) {
	s.rate = rate
	if rate != nil {
		rate.codexLoginRecovery = s
	}
	s.Start()
}

func (s *CodexLoginService) Import(ctx context.Context, documents []string, options CodexLoginOptions) ([]int64, []CodexLoginFailure, error) {
	if !s.Available() {
		return nil, nil, errors.New("请配置登录 Worker 与固定 TOTP_ENCRYPTION_KEY")
	}
	if options.ProxyID != nil || options.ProxyIPGroupID == nil || *options.ProxyIPGroupID <= 0 {
		return nil, nil, errors.New("必须选择 IP 组，2FA 登录不允许服务器直连或单代理")
	}
	for _, id := range options.GroupIDs {
		if id <= 0 {
			return nil, nil, errors.New("分组 ID 无效")
		}
	}
	items, failures := ParseCodexLoginMaterials(documents)
	ids := []int64{}
	for i, item := range items {
		raw, _ := json.Marshal(item)
		encrypted, err := s.encryptor.Encrypt(string(raw))
		if err != nil {
			return ids, failures, errors.New("登录材料加密失败")
		}
		id, err := s.store.Put(ctx, item.Email, encrypted, options)
		if err != nil {
			failures = append(failures, CodexLoginFailure{0, i + 1, "账号任务正在处理或保存失败"})
			continue
		}
		ids = append(ids, id)
	}
	return ids, failures, nil
}
func (s *CodexLoginService) Jobs(ctx context.Context) ([]CodexLoginJob, error) {
	return s.store.List(ctx)
}
func (s *CodexLoginService) Retry(ctx context.Context, id int64) error {
	if !s.Available() {
		return errors.New("登录 Worker 未配置")
	}
	return s.store.Queue(ctx, id, true)
}

// Called at the existing 401 credential-owner boundary, including Spark shadows.
// It does not replay the user's request; ordinary gateway failover continues.
func (s *CodexLoginService) QueueUnauthorized(ctx context.Context, account *Account) bool {
	if !s.Available() || account == nil || !account.IsOpenAIOAuth() {
		return false
	}
	job, err := s.store.FindAccount(ctx, account.ID)
	if err != nil || job == nil {
		return false
	}
	fresh, err := s.accounts.GetByID(ctx, account.ID)
	if err != nil || fresh == nil {
		return false
	}
	if fresh.GetCredential("access_token") != account.GetCredential("access_token") {
		if s.invalidator != nil {
			_ = s.invalidator.InvalidateToken(ctx, fresh)
		}
		return true
	}
	// Don't silently restore an administrator-disabled account.
	if fresh.ErrorMessage == codexRecoveryReason {
		return true // Already owned by the durable recovery job; keep its marker intact.
	}
	if !fresh.Schedulable || fresh.Status != StatusActive {
		return false
	}
	if err = s.accounts.SetError(ctx, account.ID, codexRecoveryReason); err != nil {
		return false
	}
	if s.rate != nil {
		s.rate.notifyAccountSchedulingBlocked(account, time.Time{}, "codex_2fa_recovery")
	}
	if s.invalidator != nil {
		_ = s.invalidator.InvalidateToken(ctx, fresh)
	}
	if err = s.store.Queue(ctx, job.ID, false); err != nil {
		slog.Warn("codex_login_queue_failed", "account_id", account.ID)
	}
	return true
}

func (s *CodexLoginService) tick(parent context.Context) {
	leaseBytes := make([]byte, 16)
	if _, err := rand.Read(leaseBytes); err != nil {
		return
	}
	job, err := s.store.Claim(parent, hex.EncodeToString(leaseBytes))
	if err != nil {
		slog.Warn("codex_login_claim_failed")
		return
	}
	if job == nil {
		return
	}
	ctx, cancel := context.WithTimeout(parent, 240*time.Second)
	message := s.execute(ctx, job)
	cancel()
	finishCtx, finishCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer finishCancel()
	if err = s.store.Finish(finishCtx, job, message); err != nil {
		slog.Warn("codex_login_finish_failed", "job_id", job.ID)
	}
}

func (s *CodexLoginService) execute(ctx context.Context, job *CodexLoginJob) string {
	raw, err := s.encryptor.Decrypt(job.Encrypted)
	if err != nil {
		return "登录材料解密失败，请检查固定加密密钥"
	}
	var material CodexLoginMaterial
	if json.Unmarshal([]byte(raw), &material) != nil || material.Email != job.Email {
		return "登录材料格式错误"
	}
	var account *Account
	if job.AccountID != nil {
		account, err = s.accounts.GetByID(ctx, *job.AccountID)
		if err != nil || account == nil {
			return "原账号不存在"
		}
	} else {
		matches, findErr := s.accounts.FindByExtraField(ctx, "codex_login_job_id", job.ID)
		if findErr != nil {
			return "查询已有账号失败"
		}
		if len(matches) > 1 {
			return "发现重复账号，请人工核对"
		}
		if len(matches) == 1 {
			account = &matches[0]
		}
	}
	target := &Account{Platform: PlatformOpenAI, Type: AccountTypeOAuth, ProxyID: job.Options.ProxyID, ProxyIPGroupID: job.Options.ProxyIPGroupID}
	previousToken := ""
	if account != nil {
		previousToken = account.GetCredential("access_token")
		if !account.IsOpenAIOAuth() || account.IsCredentialShadow() {
			return "账号类型不支持恢复"
		}
		if account.GetCredential("email") != "" && !strings.EqualFold(account.GetCredential("email"), material.Email) {
			return "原账号邮箱不匹配"
		}
		material.WorkspaceID = account.GetCredential("chatgpt_account_id")
		if material.WorkspaceID == "" {
			return "原账号缺少 Team 标识"
		}
		target = account
		if s.tokenCache == nil {
			return "刷新锁不可用"
		}
		key := OpenAITokenCacheKey(account)
		locked, lockErr := s.tokenCache.AcquireRefreshLock(ctx, key, 270*time.Second)
		if lockErr != nil || !locked {
			return "账号正在刷新，请稍后重试"
		}
		defer func() {
			cleanup, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_ = s.tokenCache.ReleaseRefreshLock(cleanup, key)
		}()
	}
	proxies, err := codexLoginGroupCandidates(ctx, s.oauth.ipGroupResolver, target)
	if err != nil {
		return err.Error()
	}
	slog.Info("codex_login_proxy_candidates", "job_id", job.ID, "ip_group_id", *target.ProxyIPGroupID, "candidate_count", len(proxies))
	token, err := s.runner.Login(ctx, material, proxies)
	if err != nil {
		return err.Error()
	} // runner errors are fixed safe messages, never provider bodies.
	if token == nil {
		return "invalid login worker response"
	}
	if token.Email != material.Email || token.ChatGPTAccountID == "" || (material.WorkspaceID != "" && token.ChatGPTAccountID != material.WorkspaceID) {
		return "授权返回的邮箱或 Team 不匹配"
	}
	credentials := s.oauth.BuildAccountCredentials(token)
	if account == nil {
		if finder, ok := s.accounts.(interface {
			FindCodexLoginAccount(context.Context, string, string) (*Account, error)
		}); ok {
			account, err = finder.FindCodexLoginAccount(ctx, material.Email, token.ChatGPTAccountID)
			if err != nil {
				return "发现重复账号，请先核对同一邮箱与 Team 的账号配置"
			}
			if account != nil {
				previousToken = account.GetCredential("access_token")
				if s.tokenCache == nil {
					return "刷新锁不可用"
				}
				key := OpenAITokenCacheKey(account)
				locked, lockErr := s.tokenCache.AcquireRefreshLock(ctx, key, 270*time.Second)
				if lockErr != nil || !locked {
					return "原账号正在刷新，请稍后重试"
				}
				defer func() {
					cleanup, cancel := context.WithTimeout(context.Background(), 3*time.Second)
					defer cancel()
					_ = s.tokenCache.ReleaseRefreshLock(cleanup, key)
				}()
				if account.Status == StatusError {
					if !strings.Contains(account.ErrorMessage, "401") {
						return "原账号存在其他错误，未自动恢复"
					}
					if err = s.accounts.SetError(ctx, account.ID, codexRecoveryReason); err != nil {
						return "原账号状态更新失败"
					}
				}
			}
		}
	}
	if account == nil {
		models := map[string]string{}
		for _, id := range openai.DefaultModelIDs() {
			models[id] = id
		}
		credentials["model_mapping"] = models
		account, err = s.admin.CreateAccount(ctx, &CreateAccountInput{Name: material.Email, Platform: PlatformOpenAI, Type: AccountTypeOAuth,
			Credentials: credentials, Extra: map[string]any{"codex_login_job_id": job.ID, "codex_fingerprint_mode": "device"},
			ProxyID: job.Options.ProxyID, ProxyIPGroupID: job.Options.ProxyIPGroupID, GroupIDs: job.Options.GroupIDs, Concurrency: DefaultOpenAIAccountConcurrency, Priority: 50})
		if err != nil {
			return "账号配置创建失败，请检查分组与代理设置"
		}
	} else {
		// Re-read under the shared refresh lock: preserve every non-auth configuration key.
		account, err = s.accounts.GetByID(ctx, account.ID)
		if err != nil || account == nil {
			return "原账号不存在"
		}
		if account.GetCredential("access_token") != previousToken {
			return "账号凭据已被更新，未覆盖，请检查任务"
		}
		updater, ok := s.accounts.(interface {
			UpdateCodexLoginCredentials(context.Context, int64, string, map[string]any) error
		})
		if !ok {
			return "账号凭据更新接口不可用"
		}
		if err = updater.UpdateCodexLoginCredentials(ctx, account.ID, previousToken, credentials); err != nil {
			return "保存新凭据失败"
		}
		for key, value := range credentials {
			account.Credentials[key] = value
		}
	}
	if err = s.store.Bind(ctx, job, account.ID); err != nil {
		return "绑定账号配置失败"
	}
	if s.invalidator != nil {
		if err = s.invalidator.InvalidateToken(ctx, account); err != nil {
			return "新凭据已保存，缓存清理失败"
		}
	}
	if account.ErrorMessage == codexRecoveryReason {
		if s.rate != nil {
			if err = s.accounts.ClearTempUnschedulable(ctx, account.ID); err != nil {
				return "新凭据已保存，清理调度冷却失败"
			}
			if s.rate.tempUnschedCache != nil {
				if err = s.rate.tempUnschedCache.DeleteTempUnsched(ctx, account.ID); err != nil {
					return "新凭据已保存，清理调度缓存失败"
				}
			}
		}
		restorer, ok := s.accounts.(interface {
			CompleteCodexLoginRecovery(context.Context, int64, string, string) (bool, error)
		})
		if !ok {
			return "新凭据已保存，账号恢复接口不可用"
		}
		restored, restoreErr := restorer.CompleteCodexLoginRecovery(ctx, account.ID, token.AccessToken, codexRecoveryReason)
		if restoreErr != nil || !restored {
			return "新凭据已保存，账号状态已被修改，未自动启用"
		}
		if s.rate != nil {
			s.rate.notifyAccountSchedulingBlockCleared(account.ID)
		}
	}
	return ""
}

// Candidate filtering uses configured live group members; Worker probes actual HTTP/SOCKS connectivity.
func codexLoginGroupCandidates(ctx context.Context, resolver *openAIIPGroupResolver, account *Account) ([]CodexLoginProxy, error) {
	if account == nil || account.ProxyIPGroupID == nil || *account.ProxyIPGroupID <= 0 || resolver == nil {
		return nil, errors.New("必须配置有效 IP 组，禁止服务器直连")
	}
	group, err := resolver.groups.GetByID(ctx, *account.ProxyIPGroupID)
	if err != nil || group == nil {
		return nil, errors.New("IP 组不可用，禁止服务器直连")
	}
	members, err := resolver.loadLiveMembers(ctx, group)
	if err != nil || len(members) == 0 {
		return nil, errors.New("IP 组没有启用且未过期的代理")
	}
	if len(members) > 256 {
		return nil, errors.New("登录 IP 组最多支持 256 个代理")
	}
	result := make([]CodexLoginProxy, 0, len(members))
	for _, proxy := range members {
		result = append(result, CodexLoginProxy{ID: proxy.ID, URL: proxy.URL()})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}
