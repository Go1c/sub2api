# 部署到 Zeabur

部署目标：Zeabur（托管 PaaS），源码从 `publish` 分支拉取，构建 `frontend-dashboard/`。

## 前置

- [ ] Zeabur 账号，已关联 GitHub。
- [ ] 本地已 `pnpm build` 验证过 `frontend-dashboard/dist/` 正常生成。
- [ ] `publish` 分支上已有需要的 commits（从 `dev` 合入），见 `doc/branching.md`。

## 推荐部署方式（静态站点）

`frontend-dashboard/` 是纯前端 SPA，最简单是把构建产物作为静态站点部署。

### 方式 A：Zeabur 识别 Vite 自动构建

1. Zeabur 控制台 → New Project → Deploy from GitHub。
2. 选择本仓库 `Go1c/sub2api`，分支 **`publish`**。
3. Root Directory 设为 `frontend-dashboard`。Zeabur 会自动识别 Vite。
4. Build 配置（通常自动）：
   - Install: `pnpm install --frozen-lockfile`
   - Build: `pnpm build`
   - Output: `dist`
5. Environment variables（Settings → Variables）：
   ```
   VITE_API_BASE_URL=https://your-backend.zeabur.app    # 若连接真后端
   VITE_USE_MOCK=false
   VITE_SITE_NAME=Dragon Code
   ```
   留空 `VITE_API_BASE_URL` 会启用 mock（不推荐生产，但适合先上线看前端）。
6. Save，触发第一次部署。

### 方式 B：`zeabur.json` 显式声明

如果自动识别有问题，在 `frontend-dashboard/` 根目录加：

```json
{
  "framework": "vite",
  "install": "pnpm install --frozen-lockfile",
  "build": "pnpm build",
  "output": "dist"
}
```

## 与后端联调

若 Zeabur 上也部署了 sub2api 后端：

1. 后端服务的内部/公开域名填入前端 `VITE_API_BASE_URL`。
2. 后端需开 CORS 允许前端域名（见 `backend/internal/config/cors.yaml` 或等效配置，参考根 `README.md` "CORS 允许来源" 一节）。
3. 若用相同根域的子域，可在 Zeabur 配置 rewrite 把 `/api/*` 指回后端，前端 `VITE_API_BASE_URL` 留空、保留相对路径即可。

## 域名与 HTTPS

- Zeabur 默认给 `*.zeabur.app` 域名，已启 TLS。
- 自定义域名：Zeabur 控制台 → Domains → Add。

## 发布流程

```bash
# 1. 在 dev 上充分测试
git checkout dev && pnpm --dir frontend-dashboard build
# 2. 合入 publish
git checkout publish
git merge --no-ff dev -m "release: vX.Y.Z"
# 3. 打 tag 并推送
git tag -a vX.Y.Z -m "release X.Y.Z"
git push origin publish --tags
# 4. Zeabur 检测到 publish 更新，自动部署
```

## 回滚

Zeabur 控制台 → Deployments → 选择上一个绿色 deploy → Redeploy。
或 `git revert <bad-sha>` 到 `publish`，push 重新触发。

## 常见坑

| 现象 | 原因 | 解决 |
|------|------|------|
| 部署成功但页面白屏 | SPA 路由没配 rewrite | Zeabur Settings → 勾选 "Rewrite to index.html" 或加 `_redirects: /* /index.html 200` |
| 字体加载慢 | Google Fonts CDN 被网络限制 | 换成本地托管 —— 把 WOFF2 放 `public/fonts/` 并在 `tokens.css` 用 `@font-face` |
| 环境变量不生效 | 变量名没有 `VITE_` 前缀 | Vite 只暴露 `VITE_*` 前缀的环境变量给客户端 |
| build 失败 `Cannot find module 'node'` | 镜像缺 Node 22+ | package.json 里 `engines.node >= 18` 已声明；若 Zeabur 仍取了旧版本，Settings 里强制指定 |
