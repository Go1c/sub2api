package service

// fork 侧上游余额探测/登录子系统（自 dev 的 account_usage_service.go 整块抽出）。

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	newAPIQuotaPerUnit                    = 500000.0
	newAPITokenQuotaNotUserBalanceMessage = "New API token quota is not user account balance; use upstream panel login to fetch user balance credentials"
	upstreamBalanceTimeout                = 10 * time.Second
	upstreamBalanceLoginTimeout           = 10 * time.Second
	upstreamBalanceBodyMax                = 4096
	upstreamBalanceCookieCredentialPrefix = "cookie:"
)

var upstreamBalancePaths = []string{
	"/v1/usage",
	"/api/v1/usage",
	"/dashboard/billing/credit_grants",
	"/v1/dashboard/billing/credit_grants",
	"/api/v1/balance",
	"/v1/balance",
	"/usage",
	"/api/usage/token/",
	"/api/usage/token",
}

// UpstreamBalanceResult records a lightweight upstream balance probe result.
type UpstreamBalanceResult struct {
	Enabled    bool       `json:"enabled"`
	CheckedAt  *time.Time `json:"checked_at,omitempty"`
	Success    bool       `json:"success"`
	StatusCode int        `json:"status_code,omitempty"`
	Path       string     `json:"path,omitempty"`
	Balance    *float64   `json:"balance,omitempty"`
	Currency   string     `json:"currency,omitempty"`
	Message    string     `json:"message,omitempty"`
	Raw        string     `json:"raw,omitempty"`
}

// UpstreamBalanceLoginInput is a one-shot upstream login request used to obtain
// user balance credentials without persisting the upstream password.
type UpstreamBalanceLoginInput struct {
	BaseURL  string
	Provider string
	Username string
	Password string
}

// UpstreamBalanceLoginCredentials contains the reusable user auth material
// extracted from a NewAPI/Sub2API login response.
type UpstreamBalanceLoginCredentials struct {
	Provider    string   `json:"provider"`
	AccessToken string   `json:"access_token"`
	UserID      string   `json:"user_id,omitempty"`
	Balance     *float64 `json:"balance,omitempty"`
	Currency    string   `json:"currency,omitempty"`
}

func isUpstreamBalanceEnabled(account *Account) bool {
	if account == nil || account.Extra == nil {
		return false
	}
	enabled, _ := account.Extra["upstream_balance_enabled"].(bool)
	if !enabled {
		return false
	}
	return account.Type == AccountTypeAPIKey || account.Type == AccountTypeUpstream
}

func upstreamBalanceBaseURL(account *Account) string {
	if account == nil {
		return ""
	}
	baseURL := strings.TrimSpace(account.GetCredential("base_url"))
	if baseURL != "" {
		return baseURL
	}
	if account.Type == AccountTypeAPIKey {
		return account.GetBaseURL()
	}
	return ""
}

func upstreamBalanceBaseURLs(baseURL string) []string {
	normalized := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if normalized == "" {
		return nil
	}
	candidates := []string{normalized}
	apiRoot := stripUpstreamBalanceAPISuffix(normalized)
	if apiRoot != "" && apiRoot != normalized {
		candidates = append(candidates, apiRoot)
	}
	return candidates
}

func stripUpstreamBalanceAPISuffix(baseURL string) string {
	lower := strings.ToLower(baseURL)
	for _, suffix := range []string{"/api/v1", "/api/v1beta", "/v1", "/v1beta"} {
		if strings.HasSuffix(lower, suffix) {
			return strings.TrimRight(baseURL[:len(baseURL)-len(suffix)], "/")
		}
	}
	return baseURL
}

func upstreamBalanceProbePaths(account *Account) []string {
	paths := make([]string, 0, len(upstreamBalancePaths)+3)
	switch upstreamBalanceProvider(account) {
	case "newapi":
		if hasNewAPIUserBalanceAuth(account) {
			return append(paths, "/api/user/self")
		}
		return paths
	case "sub2api":
		if hasUpstreamBalanceUserAccessToken(account) {
			return append(paths, "/api/v1/user/profile", "/api/user/profile")
		}
		return paths
	case "other":
		return append(paths, upstreamBalancePaths...)
	}
	if hasNewAPIUserBalanceAuth(account) {
		paths = append(paths, "/api/user/self")
	}
	if hasUpstreamBalanceUserAccessToken(account) {
		paths = append(paths, "/api/v1/user/profile", "/api/user/profile")
	}
	paths = append(paths, upstreamBalancePaths...)
	return paths
}

func hasNewAPIUserBalanceAuth(account *Account) bool {
	return upstreamBalanceNewAPIAccessToken(account) != "" && upstreamBalanceNewAPIUserID(account) != ""
}

func hasUpstreamBalanceUserAccessToken(account *Account) bool {
	return upstreamBalanceNewAPIAccessToken(account) != ""
}

func upstreamBalanceNewAPIAccessToken(account *Account) string {
	if account == nil || account.Extra == nil {
		return ""
	}
	for _, key := range []string{"upstream_balance_access_token", "newapi_access_token"} {
		if raw, ok := account.Extra[key]; ok {
			if value := strings.TrimSpace(fmt.Sprint(raw)); value != "" && value != "<nil>" {
				return value
			}
		}
	}
	return ""
}

func upstreamBalanceNewAPIUserID(account *Account) string {
	if account == nil || account.Extra == nil {
		return ""
	}
	for _, key := range []string{"upstream_balance_user_id", "newapi_user_id"} {
		if raw, ok := account.Extra[key]; ok {
			if value := strings.TrimSpace(fmt.Sprint(raw)); value != "" && value != "<nil>" {
				return value
			}
		}
	}
	return ""
}

func upstreamBalanceProvider(account *Account) string {
	if account == nil || account.Extra == nil {
		return "auto"
	}
	if raw, ok := account.Extra["upstream_balance_provider"]; ok {
		return normalizeUpstreamBalanceProvider(fmt.Sprint(raw))
	}
	return "auto"
}

func normalizeUpstreamBalanceProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "newapi", "new_api", "new-api":
		return "newapi"
	case "sub2api", "sub2_api", "sub2-api":
		return "sub2api"
	case "other", "generic":
		return "other"
	default:
		return "auto"
	}
}

type upstreamBalanceLoginProbe struct {
	provider string
	path     string
	payload  map[string]string
}

// FetchUpstreamBalanceLoginCredentials logs into a NewAPI/Sub2API-compatible
// upstream once and returns reusable balance auth fields. It does not store the
// upstream password.
func (s *AccountUsageService) FetchUpstreamBalanceLoginCredentials(ctx context.Context, input UpstreamBalanceLoginInput) (*UpstreamBalanceLoginCredentials, error) {
	baseURLs := upstreamBalanceBaseURLs(input.BaseURL)
	username := strings.TrimSpace(input.Username)
	password := input.Password
	if len(baseURLs) == 0 {
		return nil, fmt.Errorf("base_url is empty")
	}
	if username == "" || password == "" {
		return nil, fmt.Errorf("username or password is empty")
	}
	provider := normalizeUpstreamBalanceProvider(input.Provider)
	if provider == "other" {
		return nil, fmt.Errorf("upstream balance login does not support provider other")
	}

	reqCtx, cancel := context.WithTimeout(ctx, upstreamBalanceLoginTimeout)
	defer cancel()

	probes := make([]upstreamBalanceLoginProbe, 0, 2)
	if provider == "auto" || provider == "newapi" {
		probes = append(probes, upstreamBalanceLoginProbe{
			provider: "newapi",
			path:     "/api/user/login",
			payload: map[string]string{
				"username": username,
				"password": password,
			},
		})
	}
	if provider == "auto" || provider == "sub2api" {
		probes = append(probes, upstreamBalanceLoginProbe{
			provider: "sub2api",
			path:     "/api/v1/auth/login",
			payload: map[string]string{
				"email":    username,
				"password": password,
			},
		})
	}

	var lastErr error
	for _, baseURL := range baseURLs {
		for _, probe := range probes {
			result, err := fetchUpstreamBalanceLoginPath(reqCtx, baseURL, probe)
			if err == nil {
				return result, nil
			}
			lastErr = err
		}
	}
	if lastErr != nil {
		return nil, fmt.Errorf("upstream balance login failed: %w", lastErr)
	}
	return nil, fmt.Errorf("upstream balance login failed")
}

func fetchUpstreamBalanceLoginPath(ctx context.Context, baseURL string, probe upstreamBalanceLoginProbe) (*UpstreamBalanceLoginCredentials, error) {
	body, err := json.Marshal(probe.payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+probe.path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	raw, _ := io.ReadAll(io.LimitReader(resp.Body, upstreamBalanceBodyMax))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%s %s", probe.path, resp.Status)
	}
	result, err := parseUpstreamBalanceLoginCredentials(probe.provider, raw, resp.Cookies())
	if err != nil {
		return nil, fmt.Errorf("%s %w", probe.path, err)
	}
	if probe.provider == "newapi" {
		if err := verifyNewAPIUpstreamBalanceLogin(ctx, baseURL, result); err != nil {
			return nil, fmt.Errorf("%s %w", probe.path, err)
		}
	}
	return result, nil
}

func parseUpstreamBalanceLoginCredentials(provider string, body []byte, cookies []*http.Cookie) (*UpstreamBalanceLoginCredentials, error) {
	var payload any
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	if message := upstreamBalanceLoginFailureMessage(payload); message != "" {
		return nil, fmt.Errorf("%s", message)
	}

	accessToken := strings.TrimSpace(findStringValue(payload, []string{"access_token", "accessToken", "token"}, 0))
	if accessToken == "" {
		accessToken = upstreamBalanceCookieCredential(cookies)
	}
	if accessToken == "" {
		return nil, fmt.Errorf("access token or session cookie not found")
	}
	userID := strings.TrimSpace(findStringValue(payload, []string{"user_id", "userId", "uid"}, 0))
	if userID == "" {
		if id := findNumericValue(payload, []string{"id"}, 0); id != nil {
			userID = strconv.FormatInt(int64(*id), 10)
		}
	}

	balance, currency := parseUpstreamBalance(body)
	return &UpstreamBalanceLoginCredentials{
		Provider:    provider,
		AccessToken: accessToken,
		UserID:      userID,
		Balance:     balance,
		Currency:    currency,
	}, nil
}

func upstreamBalanceCookieCredential(cookies []*http.Cookie) string {
	for _, cookie := range cookies {
		if cookie == nil || strings.TrimSpace(cookie.Name) == "" || strings.TrimSpace(cookie.Value) == "" {
			continue
		}
		if strings.Contains(strings.ToLower(cookie.Name), "session") {
			return upstreamBalanceCookieCredentialPrefix + cookie.Name + "=" + cookie.Value
		}
	}
	for _, cookie := range cookies {
		if cookie == nil || strings.TrimSpace(cookie.Name) == "" || strings.TrimSpace(cookie.Value) == "" {
			continue
		}
		return upstreamBalanceCookieCredentialPrefix + cookie.Name + "=" + cookie.Value
	}
	return ""
}

func verifyNewAPIUpstreamBalanceLogin(ctx context.Context, baseURL string, credentials *UpstreamBalanceLoginCredentials) error {
	if credentials == nil || strings.TrimSpace(credentials.AccessToken) == "" {
		return fmt.Errorf("access token or session cookie not found")
	}
	if strings.TrimSpace(credentials.UserID) == "" {
		return fmt.Errorf("user id not found")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/user/self", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if cookie, ok := upstreamBalanceCookieAuthValue(credentials.AccessToken); ok {
		req.Header.Set("Cookie", cookie)
	} else {
		req.Header.Set("Authorization", "Bearer "+credentials.AccessToken)
	}
	req.Header.Set("New-Api-User", credentials.UserID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	raw, _ := io.ReadAll(io.LimitReader(resp.Body, upstreamBalanceBodyMax))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("/api/user/self %s", resp.Status)
	}
	balance, currency := parseUpstreamBalance(raw)
	if balance == nil {
		return fmt.Errorf("/api/user/self user balance not found")
	}
	credentials.Balance = balance
	credentials.Currency = currency
	return nil
}

func findStringValue(value any, keys []string, depth int) string {
	if depth > 4 {
		return ""
	}
	switch v := value.(type) {
	case map[string]any:
		for _, key := range keys {
			if raw, ok := v[key].(string); ok {
				return raw
			}
		}
		for _, raw := range v {
			if parsed := findStringValue(raw, keys, depth+1); parsed != "" {
				return parsed
			}
		}
	case []any:
		for _, raw := range v {
			if parsed := findStringValue(raw, keys, depth+1); parsed != "" {
				return parsed
			}
		}
	}
	return ""
}

func findNumericValue(value any, keys []string, depth int) *float64 {
	if depth > 4 {
		return nil
	}
	switch v := value.(type) {
	case map[string]any:
		for _, key := range keys {
			if raw, ok := v[key]; ok {
				if parsed, ok := numericFromAny(raw); ok {
					return &parsed
				}
			}
		}
		for _, raw := range v {
			if parsed := findNumericValue(raw, keys, depth+1); parsed != nil {
				return parsed
			}
		}
	case []any:
		for _, raw := range v {
			if parsed := findNumericValue(raw, keys, depth+1); parsed != nil {
				return parsed
			}
		}
	}
	return nil
}

func upstreamBalanceLoginFailureMessage(value any) string {
	payload, ok := asStringMap(value)
	if !ok {
		return ""
	}
	if success, ok := payload["success"].(bool); ok && !success {
		if message := findStringValue(payload, []string{"message", "error"}, 0); message != "" {
			return message
		}
		return "upstream login failed"
	}
	if code, ok := numericFromAny(payload["code"]); ok && code != 0 {
		if message := findStringValue(payload, []string{"message", "error"}, 0); message != "" {
			return message
		}
		return fmt.Sprintf("upstream login failed with code %.0f", code)
	}
	return ""
}

func upstreamBalanceCookieAuthValue(token string) (string, bool) {
	trimmed := strings.TrimSpace(token)
	if trimmed == "" {
		return "", false
	}
	if cookie, ok := strings.CutPrefix(trimmed, upstreamBalanceCookieCredentialPrefix); ok {
		cookie = strings.TrimSpace(cookie)
		return cookie, cookie != ""
	}
	if strings.HasPrefix(strings.ToLower(trimmed), "session=") {
		return trimmed, true
	}
	return "", false
}

func parseUpstreamBalance(body []byte) (*float64, string) {
	var payload any
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		return nil, ""
	}
	if isNewAPITokenUsageValue(payload) {
		return nil, ""
	}
	if balance, currency, ok := parseNewAPIUserBalance(payload); ok {
		return balance, currency
	}
	if balance := findNumericValue(payload, []string{
		"wallet_balance",
		"account_balance",
		"balance",
	}, 0); balance != nil {
		currency := findStringValue(payload, []string{"currency", "unit"}, 0)
		return balance, currency
	}
	if balance, currency, ok := parseGenericUpstreamCreditBalance(payload); ok {
		return balance, currency
	}
	return nil, ""
}

func numericFromAny(value any) (float64, bool) {
	switch v := value.(type) {
	case json.Number:
		parsed, err := v.Float64()
		return parsed, err == nil
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case string:
		trimmed := strings.TrimSpace(strings.TrimPrefix(v, "$"))
		parsed, err := strconv.ParseFloat(trimmed, 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func asStringMap(value any) (map[string]any, bool) {
	m, ok := value.(map[string]any)
	return m, ok
}

func isNewAPITokenUsageValue(value any) bool {
	data, ok := asStringMap(value)
	if !ok {
		return false
	}
	if nested, ok := asStringMap(data["data"]); ok {
		data = nested
	}
	return looksLikeNewAPITokenUsage(data)
}

func parseNewAPIUserBalance(value any) (*float64, string, bool) {
	payload, ok := asStringMap(value)
	if !ok {
		return nil, "", false
	}
	data := payload
	if nested, ok := asStringMap(payload["data"]); ok {
		data = nested
	}
	if looksLikeNewAPITokenUsage(data) {
		return nil, "", false
	}
	quota, hasQuota := numericFromAny(data["quota"])
	if !hasQuota {
		return nil, "", false
	}
	balance := quota / newAPIQuotaPerUnit
	return &balance, "USD", true
}

func shouldTryNextUpstreamBalanceProbe(result *UpstreamBalanceResult) bool {
	if result == nil || result.Success {
		return false
	}
	if result.StatusCode == http.StatusNotFound {
		return true
	}
	if result.Message == newAPITokenQuotaNotUserBalanceMessage {
		return false
	}
	if result.StatusCode >= 200 && result.StatusCode < 300 && result.Balance == nil {
		return true
	}
	return false
}

func applyUpstreamBalanceAuthHeaders(req *http.Request, account *Account, path, apiKey string) {
	if isUpstreamBalanceUserAuthPath(path) {
		if token := upstreamBalanceNewAPIAccessToken(account); token != "" {
			if cookie, ok := upstreamBalanceCookieAuthValue(token); ok {
				req.Header.Set("Cookie", cookie)
			} else {
				req.Header.Set("Authorization", "Bearer "+token)
			}
		}
		if path == "/api/user/self" {
			if userID := upstreamBalanceNewAPIUserID(account); userID != "" {
				req.Header.Set("New-Api-User", userID)
			}
		}
		return
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("x-api-key", apiKey)
}

func isUpstreamBalanceUserAuthPath(path string) bool {
	switch path {
	case "/api/user/self", "/api/v1/user/profile", "/api/user/profile":
		return true
	default:
		return false
	}
}

func compactUpstreamBalanceRaw(body []byte) string {
	raw := strings.TrimSpace(string(body))
	raw = strings.Join(strings.Fields(raw), " ")
	if len(raw) > upstreamBalanceBodyMax {
		return raw[:upstreamBalanceBodyMax]
	}
	return raw
}

func isNewAPITokenUsagePayload(body []byte) bool {
	var payload any
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		return false
	}
	return isNewAPITokenUsageValue(payload)
}

func parseGenericUpstreamCreditBalance(value any) (*float64, string, bool) {
	currency := findStringValue(value, []string{"currency", "unit"}, 0)
	balance := findNumericValue(value, []string{
		"total_usd_available",
		"user_usd_available",
		"usd_available",
		"total_available",
		"available_balance",
		"available",
		"remaining_balance",
		"credit",
		"credits",
		"amount",
	}, 0)
	if balance == nil {
		return nil, "", false
	}
	return balance, currency, true
}

func looksLikeNewAPITokenUsage(data map[string]any) bool {
	if data == nil {
		return false
	}
	if object, _ := data["object"].(string); object == "token_usage" {
		return true
	}
	if _, ok := data["unlimited_quota"]; ok {
		if _, hasTotalUsed := data["total_used"]; hasTotalUsed {
			return true
		}
		if _, hasUsedQuota := data["used_quota"]; hasUsedQuota {
			return true
		}
		if _, hasAvailable := data["total_available"]; hasAvailable {
			return true
		}
	}
	return false
}
