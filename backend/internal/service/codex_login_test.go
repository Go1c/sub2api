package service

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/stretchr/testify/require"
)

func TestCodexLoginLiveCredentialFormat(t *testing.T) {
	path := os.Getenv("CODEX_LOGIN_TEST_RESULT")
	if path == "" {
		t.Skip("optional private single-account live result")
	}
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	var result CodexLoginResult
	require.NoError(t, json.Unmarshal(raw, &result))
	identity, err := openai.DecodeIDToken(result.Tokens.IDToken)
	require.NoError(t, err)
	info, err := validateCodexLoginResult(CodexLoginMaterial{Email: identity.Email, WorkspaceID: result.AccountID}, &result)
	require.NoError(t, err)
	require.NotEmpty(t, info.RefreshToken)
	require.Equal(t, "organization", result.WorkspaceKind)
}

func TestCodexLoginParseJSONAndText(t *testing.T) {
	for _, content := range []string{
		`[{"email":"A@example.com","password":" p ","2fa_sk":"JBSWY3DPEHPK3PXP"}]`,
		"A@example.com---- p ----JBSWY3DPEHPK3PXP",
	} {
		items, failures := ParseCodexLoginMaterials([]string{content})
		require.Empty(t, failures)
		require.Len(t, items, 1)
		require.Equal(t, "a@example.com", items[0].Email)
		require.Equal(t, " p ", items[0].Password)
	}
}

func TestCodexLoginParsePartialAndDuplicate(t *testing.T) {
	items, failures := ParseCodexLoginMaterials([]string{`[
 {"email":"a@example.com","password":"private-value","2fa":"JBSWY3DPEHPK3PXP"},
 {"email":"b@example.com","password":"private-value","2fa":"!invalid"},
 {"email":"a@example.com","password":"private-value","2fa":"JBSWY3DPEHPK3PXP"}]`})
	require.Len(t, items, 1)
	require.Len(t, failures, 2)
	require.NotContains(t, failures[0].Message, "private-value")
}

func TestCodexLoginSafeDiagnosticMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":false,"code":"credential_probe_failed","diagnostics":{"stage":"credential_probe","http_status":403,"exception_type":"Timeout","body":"PRIVATE-TOKEN"}}`))
	}))
	defer server.Close()
	runner := &HTTPCodexLoginRunner{URL: server.URL, Client: server.Client()}
	_, err := runner.Login(context.Background(), CodexLoginMaterial{}, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "额度验证")
	require.Contains(t, err.Error(), "HTTP 403")
	require.NotContains(t, err.Error(), "PRIVATE-TOKEN")
}

func TestCodexLoginDiagnosticsRejectUntrustedText(t *testing.T) {
	message := formatCodexLoginError("PRIVATE-TOKEN", CodexLoginDiagnostics{Stage: "PRIVATE-TOKEN", HTTPStatus: 9999, ExceptionType: "PRIVATE-TOKEN"})
	require.NotContains(t, message, "PRIVATE-TOKEN")
	require.NotContains(t, message, "9999")
	require.Equal(t, "登录或凭据验证失败", message)
}

func TestCodexLoginMFARejectedMessage(t *testing.T) {
	message := formatCodexLoginError("mfa_rejected", CodexLoginDiagnostics{Stage: "totp", HTTPStatus: 403, ProxyID: 22})
	require.Contains(t, message, "2FA 被上游拒绝")
	require.NotContains(t, message, "密码错误")
	require.NotContains(t, message, "未提交账号登录")
	require.Contains(t, message, "代理 #22")
}

func TestCodexLoginFailedSelectionIsLogged(t *testing.T) {
	for _, tc := range []struct {
		code, stage string
		proxy       int64
		selected    bool
	}{
		{"mfa_rejected", "totp", 22, true},
		{"proxy_unavailable", "proxy_selection", 0, false},
	} {
		t.Run(tc.code, func(t *testing.T) {
			var logs bytes.Buffer
			old := slog.Default()
			slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
			defer slog.SetDefault(old)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewEncoder(w).Encode(CodexLoginResult{Code: tc.code, Diagnostics: CodexLoginDiagnostics{Stage: tc.stage, HTTPStatus: 403, ProxyID: tc.proxy}})
			}))
			defer server.Close()
			runner := &HTTPCodexLoginRunner{URL: server.URL, Client: server.Client()}
			_, err := runner.Login(context.Background(), CodexLoginMaterial{}, []CodexLoginProxy{{ID: 22, URL: "http://SECRET:80"}})
			require.Error(t, err)
			require.Equal(t, tc.selected, bytes.Contains(logs.Bytes(), []byte("codex_login_proxy_selected")))
			require.NotContains(t, logs.String(), "SECRET")
		})
	}
}

func TestCodexLoginRejectionAndProbeMessages(t *testing.T) {
	require.Contains(t, formatCodexLoginError("additional_verification_required", CodexLoginDiagnostics{Stage: "oauth_bootstrap", HTTPStatus: 403}), "授权初始化被上游拒绝")
	require.Contains(t, formatCodexLoginError("additional_verification_required", CodexLoginDiagnostics{Stage: "password", HTTPStatus: 403}), "额外验证")
	require.NotContains(t, formatCodexLoginError("additional_verification_required", CodexLoginDiagnostics{Stage: "password", HTTPStatus: 403}), "密码错误")
	require.Contains(t, formatCodexLoginError("invalid_totp", CodexLoginDiagnostics{Stage: "totp", HTTPStatus: 401}), "2FA 验证失败")
	message := formatCodexLoginError("proxy_unavailable", CodexLoginDiagnostics{Stage: "proxy_selection", CandidateCount: 10, TriedCount: 5})
	require.Contains(t, message, "已探测 5/10 个候选")
	require.Contains(t, message, "未提交账号登录")
	require.NotContains(t, message, "代理 #")
}
