---
name: compliance-geo-block
description: 中国大陆 IP 网页访问拦截（屏蔽网站、保持 API 开放）的实现、配置与验收 / 改 geo_block 行为或排查拦截不生效时查这篇
metadata:
  type: doc
  level: L2
  status: 已上线
---

# 合规：中国大陆地区网页访问拦截（Geo Block）

简介：让中国大陆 IP 无法访问网站前端页面，同时保持中转 API / 管理 API / 支付 webhook / 健康检查始终开放。这是「AI 退出中国大陆合规清单」落地的第一步（仅完成网页屏蔽，其余项为后续路线图）。

## 背景 / 目标

落地合规清单（[`../assets/ai-exit-mainland-china-compliance-checklist.pdf`](../assets/ai-exit-mainland-china-compliance-checklist.pdf)）的第一步：产品取舍是**只屏蔽网站，不屏蔽 API 接口**。

合规清单（§13）原本建议「全链路退出」（注册、支付、清退存量、删数据等）；本步骤仅完成「网页屏蔽」，其余项为后续工作，见清单末尾路线图。

拦截范围对照：

| 范围 | 行为 |
|------|------|
| 网页 SPA 前端（`/`、`/login`、`/dashboard`、`/assets/*` 等） | 命中受限地区 → 返回 `451` 阻断页 |
| 中转 API（`/v1`、`/v1beta`、`/responses`、`/images/*`、`/antigravity/*`、`/backend-api/*`） | **始终放行** |
| 管理 API（`/api/*`） | **始终放行** |
| 支付 webhook（`/api/v1/payment/webhook/*`）、健康检查（`/health`） | **始终放行** |

## 设计

### 判定方式（实现面）

- 内置 **MaxMind GeoLite2-Country** 数据库，通过 `go:embed` 打进二进制（`backend/internal/pkg/geoip/data/GeoLite2-Country.mmdb`），自包含、不依赖宿主 volume。
- 部署为 Caddy 自托管、前面无 CDN，因此不能用 `CF-IPCountry` 这类头，必须后端自带 GeoIP。
- **Fail-open**：无法确定来源国家（私有 IP、解析失败、库无记录）时**放行**，仅在确切判定为受限地区时才拦截，避免误伤。

### 配置（开启方式）

`deploy/config.example.yaml` → `geo_block` 段，或用环境变量：

```yaml
geo_block:
  enabled: true          # 默认 false
  countries: ["CN"]      # ISO 3166-1 alpha-2，默认 ["CN"]
  whitelist:             # 豁免拦截的 IP/CIDR，默认空
    - "203.0.113.10"     # 例：管理员固定 IP
    - "198.51.100.0/24"  # 例：办公室出口网段
```

```bash
# 环境变量等价写法
GEO_BLOCK_ENABLED=true
GEO_BLOCK_COUNTRIES=CN
```

**白名单（豁免 IP）**：`geo_block.whitelist` 里的 IP 或 CIDR 不会被拦截——即使来自中国大陆也始终放行。用于管理员、运维、办公室出口 IP、合规申诉测试等场景。匹配在地区判定**之前**进行，优先级最高。

它和 `server.trusted_proxies` 不是一回事：`trusted_proxies` 决定后端如何**取到真实客户端 IP**；`whitelist` 是在取到真实 IP 之后，决定**哪些 IP 豁免地区拦截**。

### 前置依赖：必须配置 trusted_proxies

国家判定依赖真实客户端 IP。本服务前面是 Caddy 反代（转发到 `localhost:8080`，并设置 `X-Real-IP` / `X-Forwarded-For`），因此必须配置：

```yaml
server:
  trusted_proxies: ["127.0.0.1"]   # 视实际反代来源调整
```

否则后端只能看到反代的内网 IP（私有地址），判不出国家 → fail-open 放行 → 拦截不生效。

### 更新 GeoIP 数据库（运维面）

数据库为静态嵌入，需定期更新（建议每月，配合合规月度抽查）：

1. 从 MaxMind（免费注册）或可信镜像获取最新 `GeoLite2-Country.mmdb`。
2. 覆盖 `backend/internal/pkg/geoip/data/GeoLite2-Country.mmdb`。
3. 重新构建并部署。
4. 跑 `go test ./internal/pkg/geoip/...` 确认 CN/US 判定仍正确。

## 已决策

- **只屏蔽网站、不屏蔽 API**——产品取舍，保证已有 API 调用方不受影响。
- **GeoIP 内置 mmdb（go:embed）**——自托管 + 无 CDN，无法依赖反代头部，且要避免宿主 volume 依赖。
- **Fail-open**——判不出国家时放行，宁可漏拦也不误伤正常用户。
- **白名单优先于地区判定**——保证管理员/运维/合规测试始终可达。

## 待解决

合规清单其余项（后续路线图）：注册拦截、支付拦截、清退存量用户、删除数据等「全链路退出」。本步骤仅完成网页屏蔽。

留证（合规留证包，按清单 §11）需归档：

- 本文档 + `geo_block` 配置截图
- 拦截测试截图（大陆 IP 451、API 正常）
- mmdb 更新记录（来源、日期、版本）

### 验收

```bash
cd backend
go build ./... && go vet ./...
go test ./internal/pkg/geoip/... ./internal/server/middleware/...
```

本地起服务（`geo_block.enabled=true`、`server.trusted_proxies: ["127.0.0.1"]`）后：

```bash
# 网页：大陆 IP → 451
curl -s -o /dev/null -w "%{http_code}\n" -H "X-Forwarded-For: 114.114.114.114" http://localhost:8080/
# 期望 451

# 中转 API：大陆 IP → 不被地区拦截（按正常鉴权返回，而非 451 阻断页）
curl -s -o /dev/null -w "%{http_code}\n" -H "X-Forwarded-For: 114.114.114.114" http://localhost:8080/v1/models
# 期望 非 451

# 非大陆 IP → 网页正常
curl -s -o /dev/null -w "%{http_code}\n" -H "X-Forwarded-For: 8.8.8.8" http://localhost:8080/
# 期望 200
```

## 相关

- [合规清单 PDF](../assets/ai-exit-mainland-china-compliance-checklist.pdf)
- 相关代码：
  - `backend/internal/pkg/geoip/geoip.go` — 内置 mmdb + `LookupCountryISO`
  - `backend/internal/server/middleware/geoblock.go` — 拦截中间件 + 阻断页
  - `backend/internal/server/router.go` — 在前端中间件之前接线
  - `backend/internal/config/config.go` — `GeoBlockConfig` + 默认值
  - `deploy/config.example.yaml` — `geo_block` 样例配置
