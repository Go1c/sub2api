package service

import (
	"context"
	"encoding/json"
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
	_, err := runner.Login(context.Background(), CodexLoginMaterial{}, "")
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
