// Package provider contains concrete payment provider implementations.
package provider

import (
	"context"
	"crypto/hmac"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/payment"
)

const (
	mapayCodeSuccess     = 1
	mapayStatusPaid      = 1
	mapayHTTPTimeout     = 10 * time.Second
	maxMapayResponseSize = 1 << 20 // 1MB
	mapayAPIPath         = "/xpay/epay"
)

// Mapay implements payment.Provider for the Mapay EasyPay-compatible platform.
type Mapay struct {
	instanceID string
	config     map[string]string
	httpClient *http.Client
}

// NewMapay creates a new Mapay provider.
// config keys: pid, pkey, apiBase, notifyUrl, returnUrl, channelId, channelIdAlipay, channelIdWxpay
func NewMapay(instanceID string, config map[string]string) (*Mapay, error) {
	for _, k := range []string{"pid", "pkey", "apiBase", "notifyUrl", "returnUrl"} {
		if strings.TrimSpace(config[k]) == "" {
			return nil, fmt.Errorf("mapay config missing required key: %s", k)
		}
	}
	cfg := make(map[string]string, len(config))
	for k, v := range config {
		cfg[k] = v
	}
	cfg["apiBase"] = normalizeMapayAPIBase(cfg["apiBase"])
	return &Mapay{
		instanceID: instanceID,
		config:     cfg,
		httpClient: &http.Client{Timeout: mapayHTTPTimeout},
	}, nil
}

func normalizeMapayAPIBase(apiBase string) string {
	base := strings.TrimSpace(apiBase)
	if base == "" {
		return ""
	}
	if parsed, err := url.Parse(base); err == nil && parsed.Scheme != "" && parsed.Host != "" {
		parsed.RawQuery = ""
		parsed.Fragment = ""
		parsed.RawPath = ""
		parsed.Path = normalizeMapayPath(parsed.Path)
		return strings.TrimRight(parsed.String(), "/")
	}
	return strings.TrimRight(normalizeMapayPath(base), "/")
}

func normalizeMapayPath(path string) string {
	path = strings.TrimRight(strings.TrimSpace(path), "/")
	lower := strings.ToLower(path)
	for _, endpoint := range []string{"/submit.php", "/mapi.php", "/api.php"} {
		if strings.HasSuffix(lower, endpoint) {
			path = strings.TrimRight(path[:len(path)-len(endpoint)], "/")
			lower = strings.ToLower(path)
			break
		}
	}
	if lower == "" || lower == "/" {
		return mapayAPIPath
	}
	if strings.HasSuffix(lower, mapayAPIPath) {
		return path
	}
	return strings.TrimRight(path, "/") + mapayAPIPath
}

func (m *Mapay) apiBase() string {
	if m == nil {
		return ""
	}
	return normalizeMapayAPIBase(m.config["apiBase"])
}

func (m *Mapay) Name() string        { return "Mapay" }
func (m *Mapay) ProviderKey() string { return payment.TypeMapay }
func (m *Mapay) SupportedTypes() []payment.PaymentType {
	return []payment.PaymentType{payment.TypeAlipay, payment.TypeWxpay}
}

func (m *Mapay) MerchantIdentityMetadata() map[string]string {
	if m == nil {
		return nil
	}
	pid := strings.TrimSpace(m.config["pid"])
	if pid == "" {
		return nil
	}
	return map[string]string{"pid": pid}
}

func (m *Mapay) CreatePayment(ctx context.Context, req payment.CreatePaymentRequest) (*payment.CreatePaymentResponse, error) {
	mode := m.config["paymentMode"]
	if mode == paymentModePopup {
		return m.createRedirectPayment(req)
	}
	return m.createAPIPayment(ctx, req)
}

func (m *Mapay) createRedirectPayment(req payment.CreatePaymentRequest) (*payment.CreatePaymentResponse, error) {
	notifyURL, returnURL := m.resolveURLs(req)
	params := map[string]string{
		"pid":          m.config["pid"],
		"type":         req.PaymentType,
		"out_trade_no": req.OrderID,
		"notify_url":   notifyURL,
		"return_url":   returnURL,
		"name":         req.Subject,
		"money":        req.Amount,
		"clientip":     req.ClientIP,
	}
	if channelID := m.resolveChannelID(req.PaymentType); channelID != "" {
		params["channel_id"] = channelID
	}
	if req.IsMobile {
		params["device"] = deviceMobile
	}
	params["sign"] = mapaySign(params, m.config["pkey"])
	params["sign_type"] = signTypeMD5

	q := url.Values{}
	for k, v := range params {
		q.Set(k, v)
	}
	payURL := m.apiBase() + "/submit.php?" + q.Encode()
	return &payment.CreatePaymentResponse{PayURL: payURL}, nil
}

func (m *Mapay) createAPIPayment(ctx context.Context, req payment.CreatePaymentRequest) (*payment.CreatePaymentResponse, error) {
	notifyURL, returnURL := m.resolveURLs(req)
	params := map[string]string{
		"pid":          m.config["pid"],
		"type":         req.PaymentType,
		"out_trade_no": req.OrderID,
		"notify_url":   notifyURL,
		"return_url":   returnURL,
		"name":         req.Subject,
		"money":        req.Amount,
		"clientip":     req.ClientIP,
	}
	if channelID := m.resolveChannelID(req.PaymentType); channelID != "" {
		params["channel_id"] = channelID
	}
	if req.IsMobile {
		params["device"] = deviceMobile
	}
	params["sign"] = mapaySign(params, m.config["pkey"])
	params["sign_type"] = signTypeMD5

	body, err := m.post(ctx, m.apiBase()+"/mapi.php", params)
	if err != nil {
		return nil, fmt.Errorf("mapay create: %w", err)
	}
	var resp struct {
		Code      int    `json:"code"`
		Msg       string `json:"msg"`
		TradeNo   string `json:"trade_no"`
		PayURL    string `json:"payurl"`
		QRCode    string `json:"qrcode"`
		URLScheme string `json:"urlscheme"`
		Money     string `json:"money"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("mapay parse: %w", err)
	}
	if resp.Code != mapayCodeSuccess {
		return nil, fmt.Errorf("mapay error: %s", resp.Msg)
	}
	payURL := resp.PayURL
	if payURL == "" {
		payURL = resp.URLScheme
	}
	return &payment.CreatePaymentResponse{
		TradeNo:   resp.TradeNo,
		PayURL:    payURL,
		QRCode:    resp.QRCode,
		PayAmount: resp.Money,
	}, nil
}

func (m *Mapay) resolveURLs(req payment.CreatePaymentRequest) (string, string) {
	notifyURL := req.NotifyURL
	if notifyURL == "" {
		notifyURL = m.config["notifyUrl"]
	}
	returnURL := req.ReturnURL
	if returnURL == "" {
		returnURL = m.config["returnUrl"]
	}
	return notifyURL, returnURL
}

func (m *Mapay) QueryOrder(ctx context.Context, tradeNo string) (*payment.QueryOrderResponse, error) {
	params := map[string]string{
		"act":          "order",
		"pid":          m.config["pid"],
		"key":          m.config["pkey"],
		"out_trade_no": tradeNo,
	}
	body, err := m.post(ctx, m.apiBase()+"/api.php", params)
	if err != nil {
		return nil, fmt.Errorf("mapay query: %w", err)
	}
	var resp struct {
		Code   int    `json:"code"`
		Msg    string `json:"msg"`
		Status int    `json:"status"`
		Money  string `json:"money"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("mapay parse query: %w", err)
	}
	status := payment.ProviderStatusPending
	if resp.Status == mapayStatusPaid {
		status = payment.ProviderStatusPaid
	}
	amount, _ := strconv.ParseFloat(resp.Money, 64)
	return &payment.QueryOrderResponse{
		TradeNo:  tradeNo,
		Status:   status,
		Amount:   amount,
		Metadata: m.MerchantIdentityMetadata(),
	}, nil
}

func (m *Mapay) VerifyNotification(_ context.Context, rawBody string, _ map[string]string) (*payment.PaymentNotification, error) {
	values, err := url.ParseQuery(rawBody)
	if err != nil {
		return nil, fmt.Errorf("parse notify: %w", err)
	}
	params := make(map[string]string)
	for k := range values {
		params[k] = values.Get(k)
	}
	sign := params["sign"]
	if sign == "" {
		return nil, fmt.Errorf("missing sign")
	}
	if !mapayVerifySign(params, m.config["pkey"], sign) {
		return nil, fmt.Errorf("invalid signature")
	}
	status := payment.ProviderStatusFailed
	if params["trade_status"] == tradeStatusSuccess {
		status = payment.ProviderStatusSuccess
	}
	amount, _ := strconv.ParseFloat(params["money"], 64)

	metadata := m.MerchantIdentityMetadata()
	if pid := strings.TrimSpace(params["pid"]); pid != "" {
		if metadata == nil {
			metadata = map[string]string{}
		}
		metadata["pid"] = pid
	}
	return &payment.PaymentNotification{
		TradeNo:  params["trade_no"],
		OrderID:  params["out_trade_no"],
		Amount:   amount,
		Status:   status,
		RawData:  rawBody,
		Metadata: metadata,
	}, nil
}

func (m *Mapay) Refund(context.Context, payment.RefundRequest) (*payment.RefundResponse, error) {
	return nil, fmt.Errorf("mapay refund is not implemented")
}

func (m *Mapay) resolveChannelID(paymentType string) string {
	if strings.HasPrefix(paymentType, "alipay") {
		if v := m.config["channelIdAlipay"]; v != "" {
			return v
		}
		return m.config["channelId"]
	}
	if v := m.config["channelIdWxpay"]; v != "" {
		return v
	}
	return m.config["channelId"]
}

func (m *Mapay) post(ctx context.Context, endpoint string, params map[string]string) ([]byte, error) {
	form := url.Values{}
	for k, v := range params {
		form.Set(k, v)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := m.httpClient
	if client == nil {
		client = &http.Client{Timeout: mapayHTTPTimeout}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxMapayResponseSize))
	if err != nil {
		return nil, err
	}
	return body, nil
}

func mapaySign(params map[string]string, pkey string) string {
	keys := make([]string, 0, len(params))
	for k, v := range params {
		if k == "sign" || k == "sign_type" || v == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var buf strings.Builder
	for i, k := range keys {
		if i > 0 {
			_ = buf.WriteByte('&')
		}
		_, _ = buf.WriteString(k + "=" + params[k])
	}
	_, _ = buf.WriteString(pkey)
	hash := md5.Sum([]byte(buf.String()))
	return hex.EncodeToString(hash[:])
}

func mapayVerifySign(params map[string]string, pkey string, sign string) bool {
	return hmac.Equal([]byte(mapaySign(params, pkey)), []byte(sign))
}
