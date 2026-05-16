package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/payment"
)

func TestNormalizeMapayAPIBase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{input: "https://mzf.mapay.cc", want: "https://mzf.mapay.cc/xpay/epay"},
		{input: "https://mzf.mapay.cc/", want: "https://mzf.mapay.cc/xpay/epay"},
		{input: "https://mzf.mapay.cc/xpay/epay", want: "https://mzf.mapay.cc/xpay/epay"},
		{input: "https://mzf.mapay.cc/xpay/epay/", want: "https://mzf.mapay.cc/xpay/epay"},
		{input: "https://mzf.mapay.cc/xpay/epay/mapi.php", want: "https://mzf.mapay.cc/xpay/epay"},
		{input: "https://mzf.mapay.cc/xpay/epay/submit.php", want: "https://mzf.mapay.cc/xpay/epay"},
		{input: "https://mzf.mapay.cc/xpay/epay/api.php?act=order", want: "https://mzf.mapay.cc/xpay/epay"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			if got := normalizeMapayAPIBase(tt.input); got != tt.want {
				t.Fatalf("normalizeMapayAPIBase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestMapayCreateAPIPaymentSendsChannelIDAndPreservesPayAmount(t *testing.T) {
	t.Parallel()

	var gotPath string
	var gotForm url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if err := r.ParseForm(); err != nil {
			t.Errorf("ParseForm: %v", err)
		}
		gotForm = r.PostForm
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":1,"msg":"ok","trade_no":"mapay-trade-1","urlscheme":"alipays://pay","qrcode":"qr-content","money":"9.97"}`))
	}))
	defer server.Close()

	provider := newTestMapay(t, server.URL+"/xpay/epay/mapi.php", map[string]string{
		"channelId":       "default-channel",
		"channelIdAlipay": "alipay-channel",
	})
	resp, err := provider.CreatePayment(context.Background(), payment.CreatePaymentRequest{
		OrderID:     "order-1",
		Amount:      "10.00",
		PaymentType: payment.TypeAlipay,
		Subject:     "Lumio Credits",
		NotifyURL:   "https://app.example.test/notify",
		ReturnURL:   "https://app.example.test/return",
		ClientIP:    "203.0.113.8",
		IsMobile:    true,
	})
	if err != nil {
		t.Fatalf("CreatePayment returned error: %v", err)
	}
	if gotPath != "/xpay/epay/mapi.php" {
		t.Fatalf("create path = %q, want /xpay/epay/mapi.php", gotPath)
	}
	for key, want := range map[string]string{
		"pid":          "pid-1",
		"type":         payment.TypeAlipay,
		"out_trade_no": "order-1",
		"notify_url":   "https://app.example.test/notify",
		"return_url":   "https://app.example.test/return",
		"name":         "Lumio Credits",
		"money":        "10.00",
		"clientip":     "203.0.113.8",
		"device":       "mobile",
		"channel_id":   "alipay-channel",
		"sign_type":    signTypeMD5,
	} {
		if got := gotForm.Get(key); got != want {
			t.Fatalf("form[%s] = %q, want %q (form=%v)", key, got, want, gotForm)
		}
	}
	if gotForm.Get("sign") == "" {
		t.Fatalf("form[sign] is empty (form=%v)", gotForm)
	}
	if resp.TradeNo != "mapay-trade-1" {
		t.Fatalf("TradeNo = %q, want mapay-trade-1", resp.TradeNo)
	}
	if resp.PayURL != "alipays://pay" {
		t.Fatalf("PayURL = %q, want urlscheme fallback", resp.PayURL)
	}
	if resp.QRCode != "qr-content" {
		t.Fatalf("QRCode = %q, want qr-content", resp.QRCode)
	}
	if resp.PayAmount != "9.97" {
		t.Fatalf("PayAmount = %q, want 9.97", resp.PayAmount)
	}
}

func TestMapayCreatePopupPaymentBuildsSubmitURL(t *testing.T) {
	t.Parallel()

	provider := newTestMapay(t, "https://mzf.mapay.cc/xpay/epay/api.php", map[string]string{
		"paymentMode":    paymentModePopup,
		"channelIdWxpay": "wx-channel",
	})
	resp, err := provider.CreatePayment(context.Background(), payment.CreatePaymentRequest{
		OrderID:     "order-2",
		Amount:      "12.34",
		PaymentType: payment.TypeWxpay,
		Subject:     "Lumio Pro",
		ClientIP:    "198.51.100.4",
	})
	if err != nil {
		t.Fatalf("CreatePayment returned error: %v", err)
	}
	u, err := url.Parse(resp.PayURL)
	if err != nil {
		t.Fatalf("PayURL parse: %v", err)
	}
	if got := u.Scheme + "://" + u.Host + u.Path; got != "https://mzf.mapay.cc/xpay/epay/submit.php" {
		t.Fatalf("submit URL = %q, want https://mzf.mapay.cc/xpay/epay/submit.php", got)
	}
	q := u.Query()
	for key, want := range map[string]string{
		"pid":          "pid-1",
		"type":         payment.TypeWxpay,
		"out_trade_no": "order-2",
		"notify_url":   "https://merchant.example.test/notify",
		"return_url":   "https://merchant.example.test/return",
		"name":         "Lumio Pro",
		"money":        "12.34",
		"clientip":     "198.51.100.4",
		"channel_id":   "wx-channel",
		"sign_type":    signTypeMD5,
	} {
		if got := q.Get(key); got != want {
			t.Fatalf("query[%s] = %q, want %q (query=%v)", key, got, want, q)
		}
	}
	if q.Get("sign") == "" {
		t.Fatalf("query[sign] is empty (query=%v)", q)
	}
}

func TestMapayVerifyNotificationIncludesPIDMetadata(t *testing.T) {
	t.Parallel()

	provider := newTestMapay(t, "https://mzf.mapay.cc", nil)
	params := map[string]string{
		"pid":          "notify-pid",
		"trade_no":     "trade-3",
		"out_trade_no": "order-3",
		"money":        "25.00",
		"trade_status": tradeStatusSuccess,
	}
	params["sign"] = mapaySign(params, "pkey-1")
	params["sign_type"] = signTypeMD5

	form := url.Values{}
	for k, v := range params {
		form.Set(k, v)
	}
	notification, err := provider.VerifyNotification(context.Background(), form.Encode(), nil)
	if err != nil {
		t.Fatalf("VerifyNotification returned error: %v", err)
	}
	if notification.Status != payment.ProviderStatusSuccess {
		t.Fatalf("Status = %q, want success", notification.Status)
	}
	if notification.Metadata["pid"] != "notify-pid" {
		t.Fatalf("metadata pid = %q, want notify-pid", notification.Metadata["pid"])
	}
}

func TestMapayQueryOrderMapsStatusOneToPaid(t *testing.T) {
	t.Parallel()

	var gotPath string
	var gotForm url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if err := r.ParseForm(); err != nil {
			t.Errorf("ParseForm: %v", err)
		}
		gotForm = r.PostForm
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":1,"msg":"ok","status":1,"money":"25.50"}`))
	}))
	defer server.Close()

	provider := newTestMapay(t, server.URL+"/xpay/epay/submit.php", nil)
	resp, err := provider.QueryOrder(context.Background(), "order-4")
	if err != nil {
		t.Fatalf("QueryOrder returned error: %v", err)
	}
	if gotPath != "/xpay/epay/api.php" {
		t.Fatalf("query path = %q, want /xpay/epay/api.php", gotPath)
	}
	for key, want := range map[string]string{
		"act":          "order",
		"pid":          "pid-1",
		"key":          "pkey-1",
		"out_trade_no": "order-4",
	} {
		if got := gotForm.Get(key); got != want {
			t.Fatalf("form[%s] = %q, want %q (form=%v)", key, got, want, gotForm)
		}
	}
	if resp.Status != payment.ProviderStatusPaid {
		t.Fatalf("Status = %q, want paid", resp.Status)
	}
	if resp.Amount != 25.50 {
		t.Fatalf("Amount = %v, want 25.50", resp.Amount)
	}
	if resp.Metadata["pid"] != "pid-1" {
		t.Fatalf("metadata pid = %q, want pid-1", resp.Metadata["pid"])
	}
}

func newTestMapay(t *testing.T, apiBase string, overrides map[string]string) *Mapay {
	t.Helper()

	config := map[string]string{
		"pid":       "pid-1",
		"pkey":      "pkey-1",
		"apiBase":   apiBase,
		"notifyUrl": "https://merchant.example.test/notify",
		"returnUrl": "https://merchant.example.test/return",
	}
	for k, v := range overrides {
		config[k] = v
	}
	provider, err := NewMapay("test-instance", config)
	if err != nil {
		t.Fatalf("NewMapay: %v", err)
	}
	provider.httpClient = http.DefaultClient
	return provider
}

func TestMapaySignExcludesEmptySignAndSignType(t *testing.T) {
	t.Parallel()

	pkey := "pkey-1"
	base := map[string]string{
		"a": "1",
		"b": "2",
	}
	withIgnored := map[string]string{
		"b":         "2",
		"a":         "1",
		"empty":     "",
		"sign":      strings.Repeat("0", 32),
		"sign_type": signTypeMD5,
	}

	if got, want := mapaySign(withIgnored, pkey), mapaySign(base, pkey); got != want {
		t.Fatalf("mapaySign ignored fields mismatch: got %q, want %q", got, want)
	}
}
