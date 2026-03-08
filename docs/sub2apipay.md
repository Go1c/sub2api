# Sub2ApiPay 支付充值网关（已合并）

本仓库已合并 [touwaeriol/sub2apipay](https://github.com/touwaeriol/sub2apipay) 的支付充值网关代码，便于与 Sub2API 主项目一起维护与部署。

## 说明

Sub2ApiPay 是为 Sub2API 平台构建的自托管充值支付网关，支持：

- **支付宝、微信支付**（通过 EasyPay 易支付协议聚合）
- **Stripe** 信用卡支付
- 支付成功后自动调用 Sub2API 管理接口完成余额到账

## 在本仓库中的位置

| 内容       | 路径 |
| ---------- | ---- |
| Next.js 应用 | 根目录 `src/`、`app/`、`components/`、`lib/` 等 |
| Prisma 与数据库 | `prisma/` |
| 配置与脚本 | `next.config.ts`、`prisma.config.ts`、`start.sh`、`package.json`、`pnpm-*.yaml` 等 |
| Docker 部署 | `docker-compose*.yml`（与主项目 Compose 可能需区分使用） |

## 快速开始（仅运行支付网关）

支付网关为独立 Next.js 应用，需在仓库根目录执行：

```bash
pnpm install
cp .env.example .env   # 编辑 .env，填写 Sub2API 与支付商配置
pnpm prisma migrate deploy
pnpm dev
```

默认端口 3000。生产部署可参考原项目 [touwaeriol/sub2apipay](https://github.com/touwaeriol/sub2apipay) 的 Docker 与环境变量说明。

## 环境变量要点

- `SUB2API_BASE_URL`、`SUB2API_ADMIN_API_KEY`：对接 Sub2API 管理接口
- `ADMIN_TOKEN`：管理后台令牌
- `NEXT_PUBLIC_APP_URL`：本支付服务公网地址
- EasyPay：`EASY_PAY_PID`、`EASY_PAY_PKEY`、`EASY_PAY_API_BASE` 等
- Stripe：`STRIPE_SECRET_KEY`、`STRIPE_PUBLISHABLE_KEY`、`STRIPE_WEBHOOK_SECRET`

完整说明见仓库根目录 `.env.example`（来自 sub2apipay）。

## 与 Sub2API 主项目的关系

- **Sub2API 主应用**：Go 后端在 `backend/`，Vue 前端在 `frontend/`，部署与文档见主 [README.md](../README.md)。
- **Sub2ApiPay**：Next.js 应用在根目录 `src/`、`prisma/` 等，可与主应用同仓部署或单独部署。

合并后 CI（如 `.github/workflows/release.yml`）仍以 Sub2API 主应用为主；若需单独发布 Sub2ApiPay 镜像，可参考原仓库的 Docker 与 release 流程。
