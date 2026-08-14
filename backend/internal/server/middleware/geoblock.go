package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/geoip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"

	"github.com/gin-gonic/gin"
)

// GeoBlock 返回一个按地区拦截「网页前端」访问的中间件（合规用途）。
//
// 配置在运行时按请求从 get 读取（由 SettingService 提供 DB 设置 + TTL 缓存），
// 因此管理后台保存后无需重启即时生效。中间件始终挂载，禁用时（rt.Enabled=false）
// 内部直接放行，成本极低。
//
// 仅作用于浏览器网页请求（SPA 页面与静态资源）：命中受限地区的客户端 IP
// 会收到 451 阻断页。所有 API 接口（中转 /v1 等、管理 /api/、支付 webhook、
// 健康检查）一律放行，不受影响。
//
// 白名单（rt.Whitelist，IP/CIDR）内的客户端始终放行，即使地处受限地区。
//
// 行为为 fail-open：当无法确定来源国家（私有 IP、解析失败、库无记录）时放行，
// 仅在确切判定为受限地区时才拦截，避免误伤。
func GeoBlock(get func(ctx context.Context) config.GeoBlockConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 仅拦截网页请求；所有 API 路径直接放行。
		if isAPIPath(c.Request.URL.Path) {
			c.Next()
			return
		}

		rt := get(c.Request.Context())
		if !rt.Enabled {
			c.Next()
			return
		}

		// 按运行时配置构建受限国家集合（统一大写，便于与 ISO 码比较）。
		blocked := make(map[string]struct{}, len(rt.Countries))
		for _, country := range rt.Countries {
			country = strings.ToUpper(strings.TrimSpace(country))
			if country != "" {
				blocked[country] = struct{}{}
			}
		}
		// 兜底：开了开关却没配国家时，默认拦截中国大陆，避免静默失效。
		if len(blocked) == 0 {
			blocked["CN"] = struct{}{}
		}

		// 清洗白名单（IP / CIDR），过滤空白项。
		whitelist := make([]string, 0, len(rt.Whitelist))
		for _, w := range rt.Whitelist {
			if w = strings.TrimSpace(w); w != "" {
				whitelist = append(whitelist, w)
			}
		}

		clientIP := ip.GetTrustedClientIP(c)

		// 白名单豁免：命中的 IP 即使地处受限地区也始终放行。
		if len(whitelist) > 0 && ip.MatchesAnyPattern(clientIP, whitelist) {
			c.Next()
			return
		}

		country, ok := geoip.LookupCountryISO(clientIP)
		if !ok {
			c.Next()
			return
		}
		if _, hit := blocked[country]; !hit {
			c.Next()
			return
		}

		c.Data(http.StatusUnavailableForLegalReasons, "text/html; charset=utf-8", []byte(blockPageHTML))
		c.Abort()
	}
}

// isAPIPath 判断请求是否属于「API 接口」而非网页前端。
//
// 该前缀列表必须与 web.shouldBypassEmbeddedFrontend（internal/web/embed_on.go）
// 保持一致。此处刻意复制而非跨包调用：web 包的该函数受 `embed` 构建标签约束，
// 直接依赖会把构建标签耦合进来；复制一份可保证本中间件在所有构建下都能编译。
func isAPIPath(path string) bool {
	p := strings.TrimSpace(path)
	return strings.HasPrefix(p, "/api/") ||
		strings.HasPrefix(p, "/v1/") ||
		strings.HasPrefix(p, "/v1beta/") ||
		strings.HasPrefix(p, "/backend-api/") ||
		strings.HasPrefix(p, "/antigravity/") ||
		strings.HasPrefix(p, "/setup/") ||
		p == "/health" ||
		p == "/responses" ||
		strings.HasPrefix(p, "/responses/") ||
		p == "/alpha/search" ||
		p == "/x_search" ||
		strings.HasPrefix(p, "/images/")
}

// blockPageHTML 是自包含的双语阻断页：内联样式、不引用任何外部资源，
// 以兼容站点的 Content-Security-Policy。文案参照合规清单 §10.1。
const blockPageHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Service Not Available in Mainland China</title>
<style>
  html,body{height:100%;margin:0}
  body{
    display:flex;align-items:center;justify-content:center;
    min-height:100%;padding:24px;box-sizing:border-box;
    font-family:system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,"PingFang SC","Microsoft YaHei",sans-serif;
    background:
      radial-gradient(at 30% 12%, rgba(20,184,166,0.16) 0px, transparent 55%),
      radial-gradient(at 85% 0%, rgba(45,212,191,0.10) 0px, transparent 50%),
      linear-gradient(135deg,#1e293b 0%,#0f172a 55%,#020617 100%);
    color:#e2e8f0;
  }
  .card{
    max-width:560px;width:100%;text-align:center;
    background:rgba(255,255,255,0.04);
    backdrop-filter:blur(18px);
    border:1px solid rgba(20,184,166,0.28);
    border-radius:16px;padding:48px 40px;
    box-shadow:0 12px 40px rgba(0,0,0,0.45), 0 0 40px rgba(20,184,166,0.10), inset 0 1px 0 rgba(255,255,255,0.06);
  }
  .badge{
    display:inline-block;font-size:13px;letter-spacing:.08em;font-weight:500;
    color:#2dd4bf;border:1px solid rgba(45,212,191,0.4);
    background:rgba(20,184,166,0.08);
    border-radius:999px;padding:4px 14px;margin-bottom:24px;
  }
  h1{font-size:24px;margin:0 0 8px;font-weight:600;color:#fff}
  h2{font-size:15px;margin:0 0 28px;font-weight:400;color:#94a3b8}
  p{font-size:14px;line-height:1.7;margin:0 0 16px;color:#cbd5e1}
  .en{color:#64748b;font-size:13px}
  a{color:#2dd4bf;text-decoration:none}
  a:hover{text-decoration:underline}
</style>
</head>
<body>
  <div class="card">
    <span class="badge">451 · REGION RESTRICTED</span>
    <h1>本服务不向中国大陆地区提供</h1>
    <h2>Service Not Available in Mainland China</h2>
    <p>本服务不向位于中国大陆地区的用户、通常居住于中国大陆地区的用户、中国大陆注册或经营的实体，或代表上述个人/实体使用本服务的人提供访问、注册或技术支持。</p>
    <p class="en">This service is not offered to users located or ordinarily resident in Mainland China, to entities registered or operating in Mainland China, or to anyone using the service on their behalf.</p>
    <p>如果你认为这是误判，请通过合规申诉渠道 <a href="mailto:admin@lumio.games">admin@lumio.games</a> 联系我们。请勿使用 VPN、代理或其他方式规避地区限制。<br><span class="en">If you believe this is an error, please contact compliance support at <a href="mailto:admin@lumio.games">admin@lumio.games</a>. Do not use a VPN, proxy or other means to bypass this restriction.</span></p>
  </div>
</body>
</html>`
