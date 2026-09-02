---
name: dev-environment
description: 本地开发环境、技术栈、常见坑点与解决方案；搭环境、排查本地构建 / 数据库 / 批量改账号问题时查
metadata:
  type: doc
  level: L1
  status: 已交付
---

# 开发环境与常见坑点

> 项目环境配置、技术栈、踩过的坑。本地必跑命令 / CI 见 [`testing.md`](./testing.md),分支 / 同步见 [`workflow.md`](./workflow.md)。

## 项目基本信息

| 项 | 说明 |
|----|------|
| 上游 | `Wei-Shaw/sub2api` |
| Fork | `Go1c/sub2api`(线上品牌 LumioAPI) |
| 技术栈 | Go(Ent ORM + Gin)+ Vue3(pnpm) |
| 数据库 | PostgreSQL 16 + Redis |
| 包管理 | 后端 go modules,前端 **pnpm**(不是 npm) |

## 本地环境

- **PostgreSQL 16**:端口 5432,user/pwd/db 默认均 `sub2api`,超级用户 `postgres`/`postgres`。
- **Redis**:端口 6379,无密码。
- **工具**:`golangci-lint` v2.7、`pnpm`(`npm i -g pnpm` 或 `corepack enable && corepack prepare pnpm@9 --activate`)。

## 常见坑点

1. **pnpm-lock.yaml 必须同步提交** —— 改 `package.json` 后 `cd frontend && pnpm install` 再提交 lock,否则 CI `--frozen-lockfile` 失败。
2. **npm/pnpm node_modules 冲突** —— 报 `EPERM` 时 `rm -rf node_modules && pnpm install`。
3. **PowerShell 转义 bcrypt 的 `$`** —— bcrypt hash 写进 SQL 文件用 `psql -f` 执行,别用 `psql -c`。
4. **psql 不支持中文路径** —— 复制到纯英文路径再 `-f`。
5. **PostgreSQL 密码重置** —— 改 `pg_hba.conf` 为 `trust` → 重启 → `ALTER USER ... WITH PASSWORD` → 改回 `scram-sha-256` 重启。
6. **Go interface 加方法后补全 test stub** —— 否则 `does not implement interface`;`grep -r "type.*\(Stub\|Mock\).*struct" internal/` 逐一补。
7. **Windows psql 连 localhost 走 IPv6** —— 直接用 `127.0.0.1`。
8. **Windows 无 make** —— 用 Makefile 原始命令,如 `go test -tags=unit ./...` / `-tags=integration`。
9. **Ent schema 改后必须重新生成** —— `cd backend && go generate ./ent && git add ent/`。
10. **批量改账号会冲掉模型映射** —— OpenAI 账号(如 Codex 模型)在跨平台批量修改时白名单 / 映射可能被覆盖,后端选不到可用账号、返回 `Service temporarily unavailable`。修复:批量中补回透传映射(如 `gpt-5.3-codex -> gpt-5.3-codex-spark`);批量前按平台分组,不混选不同平台账号。
11. **PR 前检查清单** —— unit + integration 测试、`golangci-lint`、lock 同步、test stub 补全、ent 生成代码已提交。

## 相关

- 测试与 CI:[`testing.md`](./testing.md) · 工作流:[`workflow.md`](./workflow.md) · 风格:[`code-style.md`](./code-style.md)
- 参考:[Ent](https://entgo.io/docs/getting-started) · [Vue3](https://vuejs.org/) · [pnpm](https://pnpm.io/)
