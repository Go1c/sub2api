package service

import (
	"bytes"
	"context"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
)

// CodexLoginMaterial is never included in a public job response or error.
type CodexLoginMaterial struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	Secret      string `json:"totp_secret"`
	WorkspaceID string `json:"account_id,omitempty"`
}

type CodexLoginFailure struct {
	Document int    `json:"document"`
	Index    int    `json:"index"`
	Message  string `json:"message"`
}

func ParseCodexLoginMaterials(documents []string) ([]CodexLoginMaterial, []CodexLoginFailure) {
	var items []CodexLoginMaterial
	var failures []CodexLoginFailure
	seen := map[string]bool{}
	totalBytes := 0
	for d, content := range documents {
		totalBytes += len(content)
		if totalBytes > 1<<20 || d >= 20 {
			failures = append(failures, CodexLoginFailure{d + 1, 0, "导入内容超过限制"})
			break
		}
		content = strings.TrimSpace(strings.TrimPrefix(content, "\ufeff"))
		var rows []any
		if strings.HasPrefix(content, "{") || strings.HasPrefix(content, "[") {
			var value any
			if json.Unmarshal([]byte(content), &value) != nil {
				failures = append(failures, CodexLoginFailure{d + 1, 0, "JSON 格式错误"})
				continue
			}
			if object, ok := value.(map[string]any); ok {
				if accounts, exists := object["accounts"]; exists {
					value = accounts
				} else if data, exists := object["data"]; exists {
					value = data
				} else {
					value = []any{object}
				}
			}
			var ok bool
			rows, ok = value.([]any)
			if !ok {
				failures = append(failures, CodexLoginFailure{d + 1, 0, "需要账号对象或数组"})
				continue
			}
		} else {
			for _, line := range strings.Split(content, "\n") {
				if strings.TrimSpace(line) == "" {
					continue
				}
				parts := strings.Split(strings.TrimSuffix(line, "\r"), "----")
				if len(parts) != 3 {
					rows = append(rows, nil)
					continue
				}
				rows = append(rows, map[string]any{"email": parts[0], "password": parts[1], "totp_secret": parts[2]})
			}
		}
		for i, value := range rows {
			row, ok := value.(map[string]any)
			fail := func(message string) { failures = append(failures, CodexLoginFailure{d + 1, i + 1, message}) }
			if !ok {
				fail("账号字段格式错误")
				continue
			}
			text := func(key string) string { v, _ := row[key].(string); return v }
			material := CodexLoginMaterial{Email: strings.ToLower(strings.TrimSpace(text("email"))), Password: text("password"), WorkspaceID: text("account_id")}
			for _, key := range []string{"totp_secret", "2fa_secret", "2fa_sk", "2fa", "otp_secret", "secret"} {
				if v := text(key); v != "" {
					material.Secret = v
					break
				}
			}
			material.Secret = strings.ToUpper(strings.NewReplacer(" ", "", "-", "", "\t", "").Replace(material.Secret))
			material.Secret = strings.TrimRight(material.Secret, "=")
			address, err := mail.ParseAddress(material.Email)
			decoded, secretErr := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(material.Secret)
			switch {
			case err != nil || address.Address != material.Email || len(material.Email) > 254:
				fail("邮箱格式错误")
			case material.Password == "" || len(material.Password) > 1024:
				fail("密码为空或过长")
			case secretErr != nil || len(decoded) < 10 || len(decoded) > 128:
				fail("2FA 密钥格式错误")
			case len(material.WorkspaceID) > 100:
				fail("工作空间 ID 格式错误")
			case seen[material.Email]:
				fail("同一批次邮箱重复")
			case len(items) >= 100:
				fail("单次最多导入 100 个账号")
			default:
				seen[material.Email] = true
				items = append(items, material)
			}
		}
	}
	return items, failures
}

type CodexLoginDiagnostics struct {
	Stage         string `json:"stage"`
	ProxyID       int64  `json:"proxy_id"`
	HTTPStatus    int    `json:"http_status"`
	ExceptionType string `json:"exception_type"`
}

type CodexLoginResult struct {
	Diagnostics CodexLoginDiagnostics `json:"diagnostics"`
	ProxyID     int64                 `json:"proxy_id"`

	Success       bool   `json:"success"`
	Code          string `json:"code"`
	AccountID     string `json:"account_id"`
	WorkspaceKind string `json:"workspace_kind"`
	WorkspaceName string `json:"workspace_name"`
	Tokens        struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		IDToken      string `json:"id_token"`
	} `json:"tokens"`
}

type CodexLoginProxy struct {
	ID  int64  `json:"id"`
	URL string `json:"url"`
}

type CodexLoginRunner interface {
	Login(context.Context, CodexLoginMaterial, []CodexLoginProxy) (*OpenAITokenInfo, error)
}

type HTTPCodexLoginRunner struct {
	URL, Token string
	Client     *http.Client
}

func (r *HTTPCodexLoginRunner) Login(ctx context.Context, material CodexLoginMaterial, proxies []CodexLoginProxy) (*OpenAITokenInfo, error) {
	body, _ := json.Marshal(struct {
		CodexLoginMaterial
		Proxies []CodexLoginProxy `json:"proxy_candidates"`
	}{material, proxies})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(r.URL, "/")+"/login", bytes.NewReader(body))
	if err != nil {
		return nil, errors.New("登录 Worker 地址无效")
	}
	req.Header.Set("Authorization", "Bearer "+r.Token)
	req.Header.Set("Content-Type", "application/json")
	client := r.Client
	if client == nil {
		client = &http.Client{Timeout: 210 * time.Second, Transport: &http.Transport{Proxy: nil}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	response, err := client.Do(req)
	if err != nil {
		return nil, errors.New("登录 Worker 不可用或超时")
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		switch response.StatusCode {
		case http.StatusUnauthorized:
			return nil, errors.New("登录 Worker 认证失败，请检查共享密钥")
		case http.StatusServiceUnavailable:
			return nil, errors.New("登录 Worker 暂时繁忙，请稍后重试")
		}
		return nil, fmt.Errorf("登录 Worker 拒绝请求（HTTP %d）", response.StatusCode)
	}
	var result CodexLoginResult
	if json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&result) != nil {
		return nil, errors.New("登录 Worker 响应无效")
	}
	if !result.Success {
		return nil, errors.New(formatCodexLoginError(result.Code, result.Diagnostics))
	}
	validProxy := false
	for _, candidate := range proxies {
		if candidate.ID == result.ProxyID && candidate.ID > 0 {
			validProxy = true
			break
		}
	}
	if !validProxy {
		return nil, errors.New("登录 Worker 未确认选中的 IP 组代理，请更新 Worker")
	}
	slog.Info("codex_login_proxy_selected", "proxy_id", result.ProxyID)
	return validateCodexLoginResult(material, &result)
}

func formatCodexLoginError(code string, diagnostics CodexLoginDiagnostics) string {
	message := codexLoginErrorMessage(code)
	stages := map[string]string{
		"proxy_selection": "IP 组代理连通性检查", "startup": "登录环境初始化", "oauth_bootstrap": "授权初始化", "email": "邮箱确认", "password": "密码验证",
		"totp": "2FA 验证", "workspace": "Team 选择", "token_exchange": "授权码换取凭据",
		"identity_validation": "账号身份核对", "credential_probe": "Codex 额度验证",
	}
	details := []string{}
	if diagnostics.ProxyID > 0 {
		details = append(details, fmt.Sprintf("代理 #%d", diagnostics.ProxyID))
	}
	if label, ok := stages[diagnostics.Stage]; ok {
		details = append(details, label)
	}
	if diagnostics.HTTPStatus >= 100 && diagnostics.HTTPStatus <= 599 {
		details = append(details, fmt.Sprintf("HTTP %d", diagnostics.HTTPStatus))
	}
	switch diagnostics.ExceptionType {
	case "Timeout", "TimeoutError", "ConnectTimeout", "ReadTimeout":
		details = append(details, "网络超时")
	case "ConnectionError", "ProxyError", "SSLError", "RequestsError", "RequestException":
		details = append(details, "网络、代理或 TLS 错误")
	case "FileNotFoundError", "ModuleNotFoundError", "ImportError":
		details = append(details, "Worker 依赖或文件缺失")
	case "ValueError", "JSONDecodeError":
		details = append(details, "响应格式异常")
	case "RuntimeError", "OSError":
		details = append(details, "Worker 运行异常")
	}
	if len(details) > 0 {
		message += "（" + strings.Join(details, "；") + "）"
	}
	return message
}

func codexLoginErrorMessage(code string) string {
	switch code {
	case "proxy_required":
		return "必须配置 IP 组代理，禁止服务器直连"
	case "proxy_unavailable":
		return "IP 组在探测时限内没有可连通登录服务的代理，未提交账号登录"
	case "invalid_proxy":
		return "代理地址格式无效，禁止直连"
	case "invalid_password":
		return "账号密码错误"
	case "invalid_totp":
		return "2FA 验证失败，请检查密钥与服务器时间"
	case "workspace_selection_required":
		return "多个 Team，请在 JSON 指定 account_id"
	case "workspace_unavailable":
		return "原 Team 不可用，未切换个人空间"
	case "additional_verification_required":
		return "上游要求额外验证，需人工处理"
	case "rate_limited":
		return "上游限流，请稍后重试"
	case "credential_probe_failed":
		return "授权后的 Codex 额度验证失败"
	case "totp_factor_missing":
		return "登录会话未返回 2FA 验证方式"
	case "unexpected_redirect", "redirect_limit":
		return "登录重定向异常"
	case "identity_mismatch", "workspace_mismatch":
		return "授权后的邮箱或 Team 与预期不符"
	case "invalid_tokens", "incomplete_tokens":
		return "返回的授权凭据无效或不完整"
	case "login_worker_failed":
		return "登录 Worker 未正常返回，请检查运行依赖"
	case "upstream_unavailable":
		return "上游登录服务暂时不可用"
	case "login_rejected":
		return "登录请求被上游拒绝"
	case "login_timeout":
		return "登录超时"
	default:
		return "登录或凭据验证失败"
	}
}

func validateCodexLoginResult(material CodexLoginMaterial, result *CodexLoginResult) (*OpenAITokenInfo, error) {
	invalid := errors.New("登录返回的账号身份或工作空间不匹配")
	if result.AccountID == "" || (material.WorkspaceID != "" && material.WorkspaceID != result.AccountID) || result.Tokens.RefreshToken == "" {
		return nil, invalid
	}
	id, err := openai.DecodeIDToken(result.Tokens.IDToken)
	if err != nil || !strings.EqualFold(id.Email, material.Email) || id.OpenAIAuth == nil || id.OpenAIAuth.ChatGPTAccountID != result.AccountID {
		return nil, invalid
	}
	at, err := openai.DecodeIDToken(result.Tokens.AccessToken)
	if err != nil || at.OpenAIAuth == nil || at.OpenAIAuth.ChatGPTAccountID != result.AccountID || at.Exp <= time.Now().Unix() {
		return nil, invalid
	}
	return &OpenAITokenInfo{AccessToken: result.Tokens.AccessToken, RefreshToken: result.Tokens.RefreshToken, IDToken: result.Tokens.IDToken,
		ExpiresAt: at.Exp, ClientID: openai.ClientID, Email: material.Email, ChatGPTAccountID: result.AccountID, ChatGPTUserID: id.OpenAIAuth.ChatGPTUserID, PlanType: id.OpenAIAuth.ChatGPTPlanType}, nil
}
