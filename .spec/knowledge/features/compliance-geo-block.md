---
name: compliance-geo-block
description: 中国大陆 IP 网页访问拦截（屏蔽网站、保持 API 开放）的实现、配置与验收 / 改 geo_block 行为或排查拦截不生效时查这篇
metadata:
  type: doc
  level: L2
  status: 已交付
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

开关 / 国家 / 白名单现在是 **DB 运行时设置**，管理后台保存后**即时生效，无需重启**：

- Setting keys（`backend/internal/service/domain_constants.go`）：`geo_block_enabled`（"true"/"false"）、`geo_block_countries`（JSON 字符串数组，如 `["CN"]`）、`geo_block_whitelist`（JSON 字符串数组，IP/CIDR）。
- 读取：`SettingService.GetGeoBlockRuntime(ctx)` 返回 `config.GeoBlockConfig`，带 60s TTL 进程内缓存 + singleflight（与 version-bounds / balance-usage-gate 同一套运行时读取模式）。
- 写入：`SettingService.UpdateGeoBlockRuntime(ctx, cfg)` 校验+规范化后写三个 key，并**立即刷新缓存**保证生效。校验规则：countries 去空格转大写，必须是 2 位 A-Z（非法报错）；whitelist 每项必须能被 `net.ParseIP` / `net.ParseCIDR` 解析（非法报错）。
- 默认值回退顺序（逐字段）：**DB 值 → `cfg.GeoBlock` 配置种子 → 内置默认**（enabled=false、countries=`["CN"]`、whitelist 空）。key 从未写过 → 用配置种子；DB 写过空值 → 视为显式重置回内置默认。
- 管理端点（仿 overload-cooldown 的独立子端点，套 admin 鉴权）：
  - `GET /api/v1/admin/settings/geo-block` → `{ "enabled": bool, "countries": []string, "whitelist": []string }`
  - `PUT /api/v1/admin/settings/geo-block` → 同 shape，返回更新后的同 shape。JSON 字段名严格为 `enabled` / `countries` / `whitelist`（全小写）。

中间件**始终挂载**（`router.go`，前端服务中间件之前），运行时禁用（enabled=false）时内部直接放行，成本极低。

`deploy/config.example.yaml` → `geo_block` 段（或同名环境变量）现在仅作为 **DB 未写过时的种子默认**：

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
- **配置改为 DB 运行时设置（后台改完即时生效）**——原来「启动时读配置文件、静态判断」改为 `SettingService.GetGeoBlockRuntime` 带 TTL 缓存读取 DB；中间件签名改为 `GeoBlock(get func(ctx) config.GeoBlockConfig)` 并始终挂载。中间件只依赖 `config`，不 import `service`，避免循环依赖。配置文件/环境变量降级为「DB 未写过时的种子默认」。

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
  - `backend/internal/server/middleware/geoblock.go` — 拦截中间件（运行时 getter 签名）+ 阻断页
  - `backend/internal/server/router.go` — 始终挂载，注入 `GetGeoBlockRuntime` 作为 getter
  - `backend/internal/service/setting_service_geo_block.go` — `GetGeoBlockRuntime` / `UpdateGeoBlockRuntime` + 校验/规范化 + TTL 缓存
  - `backend/internal/service/domain_constants.go` — `geo_block_enabled/countries/whitelist` setting keys
  - `backend/internal/handler/admin/geo_block_handler.go` — GET/PUT `/admin/settings/geo-block`
  - `backend/internal/handler/dto/settings.go` — `GeoBlockSettings` DTO（enabled/countries/whitelist）
  - `backend/internal/config/config.go` — `GeoBlockConfig`（现作为 DB 未写时的种子默认）
  - `deploy/config.example.yaml` — `geo_block` 样例配置（种子默认）
