# frontend-dashboard

LumioAPI（视觉参考 dragoncode.codes）的独立前端工程。与仓库根部的 `frontend/`（原 Sub2API 前端）解耦，可在**无后端**环境下通过内置 mock 直接预览。

## 本地开发

```bash
cd frontend-dashboard
corepack enable && corepack prepare pnpm@9 --activate   # 如已有 pnpm 跳过
pnpm install
cp .env.example .env
pnpm dev            # http://localhost:5174
```

## 构建

```bash
pnpm build          # 输出到 dist/
pnpm preview        # 预览 dist/ 构建产物 http://localhost:4174
```

## 对接真实后端

编辑 `.env`，取消注释并填写：

```
VITE_API_BASE_URL=http://localhost:8080
VITE_USE_MOCK=false
```

## 目录

```
src/
  api/           # axios 实例 + 业务接口封装
  assets/        # 样式与静态资源
  components/    # UI 组件（dashboard / layout / common）
  composables/   # Vue 组合式函数
  i18n/          # 文案
  mock/          # 离线 mock 数据与拦截器
  router/        # 路由表
  stores/        # Pinia stores
  views/         # 页面
```

## 设计令牌

- 主色 `brand-500 #4f8cff`
- 辅色（navy）`brand-900 #1a2f5a`
- 主字体 `Source Serif 4` + `Noto Serif SC`（Google Fonts CDN，可换本地）

## 部署

见仓库根 `doc/deploy-zeabur.md`。
