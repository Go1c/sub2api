package middleware

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"

	"github.com/gin-gonic/gin"
)

func init() { gin.SetMode(gin.TestMode) }

// staticGeoBlock wraps a fixed GeoBlockConfig as the runtime getter the middleware expects.
func staticGeoBlock(cfg config.GeoBlockConfig) func(ctx context.Context) config.GeoBlockConfig {
	return func(ctx context.Context) config.GeoBlockConfig { return cfg }
}

// newGeoTestRouter 构造一个挂载了 GeoBlock 中间件的测试路由。
// 受信任的客户端 IP 取决于 trusted proxies 与 X-Forwarded-For，
// 这里把 loopback 设为可信代理，使 X-Forwarded-For 生效。
func newGeoTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	r := gin.New()
	if err := r.SetTrustedProxies([]string{"127.0.0.1"}); err != nil {
		t.Fatalf("SetTrustedProxies: %v", err)
	}
	r.Use(GeoBlock(staticGeoBlock(config.GeoBlockConfig{Enabled: true, Countries: []string{"CN"}})))
	// 网页与 API 路由都返回 200，方便区分是否被中间件拦截。
	r.GET("/", func(c *gin.Context) { c.String(http.StatusOK, "page") })
	r.GET("/dashboard", func(c *gin.Context) { c.String(http.StatusOK, "page") })
	r.GET("/v1/models", func(c *gin.Context) { c.String(http.StatusOK, "api") })
	r.GET("/api/v1/auth/login", func(c *gin.Context) { c.String(http.StatusOK, "api") })
	return r
}

func doReq(r *gin.Engine, path, xff string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.RemoteAddr = "127.0.0.1:12345"
	if xff != "" {
		req.Header.Set("X-Forwarded-For", xff)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestGeoBlock(t *testing.T) {
	const cnIP = "114.114.114.114"
	const usIP = "8.8.8.8"

	tests := []struct {
		name     string
		path     string
		xff      string
		wantCode int
	}{
		{name: "webpage from CN blocked", path: "/", xff: cnIP, wantCode: http.StatusUnavailableForLegalReasons},
		{name: "spa route from CN blocked", path: "/dashboard", xff: cnIP, wantCode: http.StatusUnavailableForLegalReasons},
		{name: "relay API from CN allowed", path: "/v1/models", xff: cnIP, wantCode: http.StatusOK},
		{name: "management API from CN allowed", path: "/api/v1/auth/login", xff: cnIP, wantCode: http.StatusOK},
		{name: "webpage from US allowed", path: "/", xff: usIP, wantCode: http.StatusOK},
		{name: "webpage from unknown/private allowed", path: "/", xff: "", wantCode: http.StatusOK},
	}

	r := newGeoTestRouter(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := doReq(r, tt.path, tt.xff)
			if w.Code != tt.wantCode {
				t.Fatalf("%s %s (xff=%q): got %d, want %d", http.MethodGet, tt.path, tt.xff, w.Code, tt.wantCode)
			}
		})
	}
}

func TestGeoBlockWhitelist(t *testing.T) {
	r := gin.New()
	if err := r.SetTrustedProxies([]string{"127.0.0.1"}); err != nil {
		t.Fatalf("SetTrustedProxies: %v", err)
	}
	r.Use(GeoBlock(staticGeoBlock(config.GeoBlockConfig{
		Enabled:   true,
		Countries: []string{"CN"},
		Whitelist: []string{"114.114.114.114", "203.0.113.0/24"},
	})))
	r.GET("/", func(c *gin.Context) { c.String(http.StatusOK, "page") })

	tests := []struct {
		name     string
		xff      string
		wantCode int
	}{
		{name: "whitelisted exact CN ip allowed", xff: "114.114.114.114", wantCode: http.StatusOK},
		{name: "whitelisted CIDR ip allowed", xff: "203.0.113.5", wantCode: http.StatusOK},
		{name: "non-whitelisted CN ip still blocked", xff: "223.5.5.5", wantCode: http.StatusUnavailableForLegalReasons},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if w := doReq(r, "/", tt.xff); w.Code != tt.wantCode {
				t.Fatalf("xff=%q: got %d, want %d", tt.xff, w.Code, tt.wantCode)
			}
		})
	}
}

// TestGeoBlockLiveServer 起一个真实 TCP 监听，端到端验证阻断页内容与 API 放行。
func TestGeoBlockLiveServer(t *testing.T) {
	srv := httptest.NewServer(newGeoTestRouter(t))
	defer srv.Close()

	get := func(path, xff string) (int, string) {
		req, err := http.NewRequest(http.MethodGet, srv.URL+path, nil)
		if err != nil {
			t.Fatalf("new request: %v", err)
		}
		req.Header.Set("X-Forwarded-For", xff)
		resp, err := srv.Client().Do(req)
		if err != nil {
			t.Fatalf("do request: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()
		b, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, string(b)
	}

	// 大陆 IP 访问网页 → 451 + 阻断页文案
	if code, body := get("/", "114.114.114.114"); code != http.StatusUnavailableForLegalReasons {
		t.Fatalf("webpage from CN: got %d, want 451", code)
	} else if !strings.Contains(body, "本服务不向中国大陆地区提供") {
		t.Fatalf("block page missing expected copy, got: %.120s", body)
	}

	// 大陆 IP 访问中转 API → 不被地区拦截，正常到达 handler
	if code, body := get("/v1/models", "114.114.114.114"); code != http.StatusOK || body != "api" {
		t.Fatalf("relay API from CN: got %d %q, want 200 \"api\"", code, body)
	}
}

func TestGeoBlockDisabledPassesThrough(t *testing.T) {
	// 中间件始终挂载（router.go），运行时 Enabled=false 时内部直接放行——
	// 即使来源是受限地区也不拦截。
	r := gin.New()
	if err := r.SetTrustedProxies([]string{"127.0.0.1"}); err != nil {
		t.Fatalf("SetTrustedProxies: %v", err)
	}
	r.Use(GeoBlock(staticGeoBlock(config.GeoBlockConfig{Enabled: false, Countries: []string{"CN"}})))
	r.GET("/", func(c *gin.Context) { c.String(http.StatusOK, "page") })

	if w := doReq(r, "/", "114.114.114.114"); w.Code != http.StatusOK {
		t.Fatalf("disabled geo block should pass through CN webpage: got %d, want 200", w.Code)
	}
}

func TestGeoBlockEmptyCountriesFallsBackToCN(t *testing.T) {
	// 开了开关但运行时国家列表为空时，仍兜底拦截 CN，不会「放行一切」。
	r := gin.New()
	if err := r.SetTrustedProxies([]string{"127.0.0.1"}); err != nil {
		t.Fatalf("SetTrustedProxies: %v", err)
	}
	r.Use(GeoBlock(staticGeoBlock(config.GeoBlockConfig{Enabled: true, Countries: nil})))
	r.GET("/", func(c *gin.Context) { c.String(http.StatusOK, "page") })

	if w := doReq(r, "/", "114.114.114.114"); w.Code != http.StatusUnavailableForLegalReasons {
		t.Fatalf("empty countries should fall back to CN block: got %d, want 451", w.Code)
	}
}
