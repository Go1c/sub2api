# sub2api (LumioAPI) CDN 缓存规则方案

适用场景：部署在 Zeabur，国内用户访问慢，不想迁服务器，希望通过 Cloudflare / EdgeOne 等 CDN 做缓存加速。**不改代码，只配面板规则。**

## 项目结构回顾

- 后端：Go（go-zero + Gin），默认端口 8080
- 前端：Vue 3 + Vite，构建产物输出到 `backend/internal/web/dist`，嵌入 Go 二进制
- 前端产物带内容 hash（`vendor-vue.[hash].js` 等），天然适合长缓存
- 对外路由前缀：`/`（SPA）、`/api/v1/*`、`/v1/*`、`/v1beta/*`、`/antigravity/*`、`/setup/*`、`/payment/webhook/*`、`/health`

## 规则顺序（重要）

面板里从上往下按顺序命中，**第一条命中即停**。顺序错了会出事故。

---

## ① 必须 Bypass（最高优先级）

匹配任一路径 → **Bypass cache / 不缓存**

```
/api/v1/auth/*
/api/v1/user/*
/api/v1/admin/*
/api/v1/payment/*
/payment/webhook/*
/setup/*
/v1/messages
/v1/chat/completions
/v1beta/*
/antigravity/*
/responses
/chat/completions
/health
```

**理由**：认证态、支付、管理后台、LLM 推理、Webhook —— 缓存就是事故。

---

## ② 短缓存（Edge 5min / Browser 60s，GET only）

```
/api/v1/settings/public
/v1/models
/v1/usage
```

**配置**：
- Cache Eligible = **GET/HEAD only**
- Edge TTL：300 秒
- Browser TTL：60 秒

**理由**：公开元数据，改动不频繁，是真正能帮国内用户减负的命中点。

---

## ③ 强缓存（1 年 immutable）

匹配后缀（Vite 产物带内容 hash，改了就换名）：

```
/assets/*
*.js
*.css
*.woff2
*.woff
*.ttf
*.png
*.jpg
*.jpeg
*.svg
*.webp
*.ico
*.wasm
```

**配置**：
- Edge TTL：31536000 秒（1 年）
- Browser TTL：31536000 秒
- 响应头：`Cache-Control: public, max-age=31536000, immutable`

**理由**：文件名带 hash，内容变了文件名也变，放心锁死。

---

## ④ 首页 HTML —— 不缓存

```
/
/index.html
```

**配置**：Bypass 或 TTL=0。

**理由**：`index.html` 是运行时动态注入站点配置的，缓存了会出现设置不同步。

---

## 面板配置注意事项

### Cloudflare
- 用 **Cache Rules**（不是老的 Page Rules）
- 按上面顺序创建，Bypass 放最上面
- **橙云必须开**（代理模式），不然 CDN 根本没接上
- 开启 **Tiered Cache**（免费档也可用）
- 开启 **Brotli / Gzip**、**HTTP/3** / 0-RTT

### EdgeOne（腾讯）
- 规则里显式勾选"仅缓存 GET/HEAD"，POST 永远不能进缓存
- 免费个人套餐：国内节点需域名备案；未备案则只能用海外节点
- 回源 Host 记得设成 Zeabur 分配的域名

---

## 验证

配完之后：

```bash
# 第一次请求（MISS）
curl -I https://你的域名/assets/index-abc123.js

# 第二次请求（应该 HIT）
curl -I https://你的域名/assets/index-abc123.js
```

看响应头：
- Cloudflare：`cf-cache-status: HIT`
- EdgeOne：`x-cache: HIT` / `edgeone-cache-status`

动态接口（例如 `/api/v1/auth/me`）应该始终是 `BYPASS` 或 `DYNAMIC`，永远不该 HIT。

---

## 预期效果与局限

**会明显变快**：
- 前端打开速度（静态资源边缘命中）
- 未登录态访问首页的后续资源加载
- 公开元数据接口（`/v1/models` 等）

**收益有限**：
- 登录后的用户态 API（本来就 bypass）
- LLM 推理请求（POST、流式，无法缓存，只能靠链路优化）

**真正想让推理也快**，要么：
1. 给国内用户加一层就近反向代理入口（仍不迁主服务）
2. 换更靠近大陆的 Zeabur 区域（香港/东京/新加坡）
3. 对推理做分区域路由
