package service

import (
	"context"
	"errors"
	"maps"
	"net/http"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type codexRecoveryRepo struct {
	AccountRepository
	account  *Account
	restored bool
}

func (r *codexRecoveryRepo) GetByID(context.Context, int64) (*Account, error) {
	copy := *r.account
	copy.Credentials = maps.Clone(r.account.Credentials)
	return &copy, nil
}
func (r *codexRecoveryRepo) SetError(_ context.Context, _ int64, message string) error {
	r.account.Status = StatusError
	r.account.Schedulable = false
	r.account.ErrorMessage = message
	return nil
}
func (r *codexRecoveryRepo) UpdateCredentials(_ context.Context, _ int64, credentials map[string]any) error {
	r.account.Credentials = credentials
	return nil
}
func (r *codexRecoveryRepo) CompleteCodexLoginRecovery(_ context.Context, _ int64, token, reason string) (bool, error) {
	if r.account.ErrorMessage != reason || r.account.GetCredential("access_token") != token {
		return false, nil
	}
	r.restored = true
	r.account.Status = StatusActive
	r.account.Schedulable = true
	r.account.ErrorMessage = ""
	return true, nil
}

type codexRecoveryEncryptor struct{}

func (r *codexRecoveryRepo) UpdateCodexLoginCredentials(_ context.Context, _ int64, previous string, credentials map[string]any) error {
	if r.account.GetCredential("access_token") != previous {
		return errors.New("credential changed")
	}
	for k, v := range credentials {
		r.account.Credentials[k] = v
	}
	return nil
}

func (codexRecoveryEncryptor) Encrypt(s string) (string, error) { return s, nil }
func (codexRecoveryEncryptor) Decrypt(s string) (string, error) { return s, nil }

type codexRecoveryCache struct{ OpenAITokenCache }

func (codexRecoveryCache) AcquireRefreshLock(context.Context, string, time.Duration) (bool, error) {
	return true, nil
}
func (codexRecoveryCache) ReleaseRefreshLock(context.Context, string) error { return nil }

type codexRecoveryRunner func(context.Context, CodexLoginMaterial, []CodexLoginProxy) (*OpenAITokenInfo, error)

func (f codexRecoveryRunner) Login(ctx context.Context, m CodexLoginMaterial, p []CodexLoginProxy) (*OpenAITokenInfo, error) {
	return f(ctx, m, p)
}

func recoveryAccount() *Account {
	return &Account{ID: 7, ProxyIPGroupID: codexTestGroupID(), Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, Credentials: map[string]any{
		"access_token": "old", "refresh_token": "old-rt", "email": "a@example.com", "chatgpt_account_id": "team-id", "model_mapping": map[string]string{"custom": "model"},
	}}
}

func TestCodexLogin401StopsSchedulingAndQueuesOnce(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	account := recoveryAccount()
	repo := &codexRecoveryRepo{account: account}
	cfg := &config.Config{}
	cfg.CodexLogin.WorkerURL = "http://worker"
	cfg.CodexLogin.WorkerToken = "private-test-token-long-enough-32-chars"
	cfg.Totp.EncryptionKeyConfigured = true
	s := &CodexLoginService{cfg: cfg, accounts: repo, store: &codexLoginStore{db}}
	for i := 0; i < 2; i++ {
		mock.ExpectQuery("SELECT id,status,attempts FROM codex_login_jobs").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"id", "status", "attempts"}).AddRow(1, "queued", 0))
		if i == 0 {
			mock.ExpectExec("UPDATE codex_login_jobs SET status='queued'").WithArgs(int64(1)).WillReturnResult(sqlmock.NewResult(0, 1))
		}
		require.True(t, s.QueueUnauthorized(context.Background(), account))
	}
	require.Equal(t, StatusError, repo.account.Status)
	require.False(t, repo.account.Schedulable)
	require.Equal(t, codexRecoveryReason, repo.account.ErrorMessage)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCodexLoginRecoveryPreservesConfigAndOnlyResumesOnSuccess(t *testing.T) {
	for _, fails := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "failure"}[fails], func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer func() { _ = db.Close() }()
			repo := &codexRecoveryRepo{account: recoveryAccount()}
			_ = repo.SetError(context.Background(), 7, codexRecoveryReason)
			runner := codexRecoveryRunner(func(_ context.Context, m CodexLoginMaterial, proxies []CodexLoginProxy) (*OpenAITokenInfo, error) {
				require.Equal(t, "team-id", m.WorkspaceID)
				require.Equal(t, []CodexLoginProxy{{ID: 2, URL: "socks5://proxy.example:1080"}}, proxies)
				require.False(t, repo.account.Schedulable)
				if fails {
					return nil, errors.New("验证失败")
				}
				return &OpenAITokenInfo{AccessToken: "new", RefreshToken: "new-rt", IDToken: "new-id", Email: m.Email, ChatGPTAccountID: m.WorkspaceID, ExpiresAt: time.Now().Add(time.Hour).Unix()}, nil
			})
			s := &CodexLoginService{store: &codexLoginStore{db}, accounts: repo, runner: runner, encryptor: codexRecoveryEncryptor{}, tokenCache: codexRecoveryCache{}, oauth: &OpenAIOAuthService{ipGroupResolver: codexTestResolver()}}
			id := int64(7)
			job := &CodexLoginJob{ID: 1, AccountID: &id, Email: "a@example.com", Lease: "lease", Encrypted: `{"email":"a@example.com","password":"private","totp_secret":"JBSWY3DPEHPK3PXP"}`}
			if !fails {
				mock.ExpectExec("UPDATE codex_login_jobs SET account_id").WithArgs(int64(1), "lease", int64(7)).WillReturnResult(sqlmock.NewResult(0, 1))
			}
			message := s.execute(context.Background(), job)
			if fails {
				require.NotEmpty(t, message)
				require.False(t, repo.restored)
				require.False(t, repo.account.Schedulable)
			} else {
				require.Empty(t, message)
				require.True(t, repo.restored)
				require.True(t, repo.account.Schedulable)
				require.Equal(t, "new-rt", repo.account.GetCredential("refresh_token"))
				require.Equal(t, map[string]string{"custom": "model"}, repo.account.Credentials["model_mapping"])
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

type codex401Hook struct {
	calls int
	id    int64
}

func (h *codex401Hook) QueueUnauthorized(_ context.Context, a *Account) bool {
	h.calls++
	h.id = a.ID
	return true
}
func TestCodexLoginTokenRevoked401UsesRecoveryHook(t *testing.T) {
	hook := &codex401Hook{}
	svc := NewRateLimitService(&codexRecoveryRepo{account: recoveryAccount()}, nil, &config.Config{}, nil, nil)
	svc.codexLoginRecovery = hook
	disabled := svc.HandleUpstreamError(context.Background(), recoveryAccount(), http.StatusUnauthorized, nil, []byte(`{"error":{"code":"token_invalidated","message":"Encountered invalidated oauth token for user, failing request"}}`))
	require.True(t, disabled)
	require.Equal(t, 1, hook.calls)
	require.Equal(t, int64(7), hook.id)
}

func codexTestGroupID() *int64 { id := int64(1); return &id }

type codexTestGroups struct{ ProxyIPGroupRepository }

func (codexTestGroups) GetByID(context.Context, int64) (*ProxyIPGroup, error) {
	return &ProxyIPGroup{ID: 1, ProxyIDs: []int64{2}}, nil
}

type codexTestProxies struct{}

func (codexTestProxies) ListByIDs(context.Context, []int64) ([]Proxy, error) {
	return []Proxy{{ID: 2, Protocol: "socks5", Host: "proxy.example", Port: 1080, Status: StatusActive}}, nil
}
func codexTestResolver() *openAIIPGroupResolver {
	return &openAIIPGroupResolver{groups: codexTestGroups{}, proxies: codexTestProxies{}, now: time.Now}
}
func TestCodexLoginProxyCandidatesNeverFallBackDirect(t *testing.T) {
	for _, account := range []*Account{nil, {}, {ProxyID: codexTestGroupID()}} {
		_, err := codexLoginGroupCandidates(context.Background(), codexTestResolver(), account)
		require.Error(t, err)
	}
	rows, err := codexLoginGroupCandidates(context.Background(), codexTestResolver(), recoveryAccount())
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, int64(2), rows[0].ID)
}
