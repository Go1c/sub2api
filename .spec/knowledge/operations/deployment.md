---
name: deployment
description: 部署与运维——Docker / 安装方式、datamanagementd、CDN 缓存规则；部署或排查线上时查
metadata:
  type: doc
  level: L1
  status: 生效
---

# 部署与运维

Sub2API 是用于分发与管理 AI 产品订阅 API 额度的网关平台（Go 后端 + Vue 前端，依赖 PostgreSQL 与 Redis）。本文汇总 Linux 服务器上的两种部署方式（Docker Compose / 二进制 systemd）、`datamanagementd` 数据管理守护进程的联动部署，以及 CDN 缓存规则配置，供部署上线与线上排障时查阅。

## 部署方式

`deploy/` 目录提供两套部署路径：

| 方式 | 适合场景 | Setup 向导 |
|------|---------|-----------|
| **Docker Compose** | 快速上手、一体化 | 不需要（自动初始化） |
| **二进制安装** | 生产服务器、systemd | Web 向导 |

`deploy/` 目录关键文件：

| 文件 | 说明 |
|------|------|
| `docker-compose.yml` | Docker Compose 配置（命名卷 named volumes） |
| `docker-compose.local.yml` | Docker Compose 配置（本地目录，便于迁移） |
| `docker-deploy.sh` | **一键 Docker 部署脚本（推荐）** |
| `.env.example` | Docker 环境变量模板 |
| `DOCKER.md` | Docker Hub 文档 |
| `install.sh` | 二进制一键安装脚本 |
| `install-datamanagementd.sh` | datamanagementd 一键安装脚本 |
| `sub2api.service` | systemd 服务单元文件 |
| `sub2api-datamanagementd.service` | datamanagementd systemd 服务单元文件 |
| `DATAMANAGEMENTD_CN.md` | datamanagementd 部署与联动说明（中文） |
| `config.example.yaml` | 配置文件示例 |

---

### Docker 部署（推荐）

#### 方式 1：一键部署（推荐）

使用自动化准备脚本完成最简部署：

```bash
# 下载并直接运行准备脚本
curl -sSL https://raw.githubusercontent.com/Wei-Shaw/sub2api/main/deploy/docker-deploy.sh | bash

# 或先下载再运行
curl -sSL https://raw.githubusercontent.com/Wei-Shaw/sub2api/main/deploy/docker-deploy.sh -o docker-deploy.sh
chmod +x docker-deploy.sh
./docker-deploy.sh
```

脚本做了什么：

- 下载 `docker-compose.local.yml` 与 `.env.example`
- 自动生成安全密钥（JWT_SECRET、TOTP_ENCRYPTION_KEY、POSTGRES_PASSWORD）
- 创建带生成密钥的 `.env` 文件
- 创建必要的数据目录（data/、postgres_data/、redis_data/）
- **打印生成的凭据**（POSTGRES_PASSWORD、JWT_SECRET 等）

运行脚本后：

```bash
# 启动服务
docker compose -f docker-compose.local.yml up -d

# 查看日志
docker compose -f docker-compose.local.yml logs -f sub2api

# 若 admin 密码为自动生成，从日志中查找：
docker compose -f docker-compose.local.yml logs sub2api | grep "admin password"

# 访问 Web UI
# http://localhost:8080
```

#### 方式 2：手动部署

如需手动控制：

```bash
# 克隆仓库
git clone https://github.com/Wei-Shaw/sub2api.git
cd sub2api/deploy

# 配置环境变量
cp .env.example .env
nano .env  # 设置 POSTGRES_PASSWORD 及其他必需变量

# 生成安全密钥（推荐）
JWT_SECRET=$(openssl rand -hex 32)
TOTP_ENCRYPTION_KEY=$(openssl rand -hex 32)
echo "JWT_SECRET=${JWT_SECRET}" >> .env
echo "TOTP_ENCRYPTION_KEY=${TOTP_ENCRYPTION_KEY}" >> .env

# 创建数据目录
mkdir -p data postgres_data redis_data

# 使用本地目录版本启动所有服务
docker compose -f docker-compose.local.yml up -d

# 查看日志（确认自动生成的 admin 密码）
docker compose -f docker-compose.local.yml logs -f sub2api

# 访问 Web UI
# http://localhost:8080
```

#### 部署版本对比

| 版本 | 数据存储 | 迁移 | 适合 |
|------|---------|------|------|
| **docker-compose.local.yml** | 本地目录（./data、./postgres_data、./redis_data） | ✅ 简单（整目录 tar） | 生产、需频繁备份/迁移 |
| **docker-compose.yml** | 命名卷（/var/lib/docker/volumes/） | ⚠️ 需 docker 命令 | 简单部署、无需迁移 |
| **docker-compose.dev.yml** | 本地目录（开发用） | ✅ | 本地开发 |

**推荐**：使用 `docker-compose.local.yml`（即 `docker-deploy.sh` 部署的版本），数据管理与迁移更方便。

#### 哪份 compose 会生效 / PG 调优键

| 路径 | 谁用 | PG 调优 |
|------|------|---------|
| `docker-compose.local.yml` | **`docker-deploy.sh` 实际下载并启动的那份**（脚本内另存为 `docker-compose.yml`） | ✅ 读取 `.env` 中 `POSTGRES_MAX_CONNECTIONS` / `POSTGRES_SHARED_BUFFERS` / `POSTGRES_EFFECTIVE_CACHE_SIZE` / `POSTGRES_MAINTENANCE_WORK_MEM` |
| `docker-compose.yml` | 手动 `docker compose -f docker-compose.yml` | ✅ 同上 |
| `docker-compose.dev.yml` | 本地开发 | ✅ 同上 |

调优键说明见 `deploy/.env.example`「PostgreSQL 服务端参数」一节。未设置时回落到 postgres 常见默认（`max_connections=100`、`shared_buffers=128MB` 等）。

#### 生产 PG 与安装脚本（运维决策，2026-07-21）

| 项 | 决策 |
|----|------|
| **publish / 线上 PG** | 与仓库 `docker-compose*.yml` **脱钩**；可能是托管实例或 `deploy-service.sh` 编排。**未上机验证**（`SHOW max_connections` 等）。当前线上部署稳定 → **不上机改参、不在本仓库假设线上等于 compose**。需要时另开运维任务。 |
| **`docker-deploy.sh` raw URL** | **保持**默认 `Wei-Shaw/sub2api/main/deploy`，**不改**指向 fork。一键安装跟上游；fork 侧 compose 改动（如 local PG 调优）请用仓库内 `deploy/` 或自行拷贝，勿依赖 raw 脚本自动同步。 |

> 仓库内 compose 的 `POSTGRES_*` 接线（local/dev 与 yml 对齐）只影响用本仓库 compose 起的栈，不自动作用于已稳定的生产。

#### 自动初始化（Auto-Setup）原理

使用 Docker Compose 且设置 `AUTO_SETUP=true` 时：

1. 首次运行系统会自动：
   - 连接 PostgreSQL 与 Redis
   - 执行数据库迁移（`backend/migrations/*.sql`），并记录到 `schema_migrations`
   - 生成 JWT 密钥（若未提供）
   - 创建 admin 账户（密码未提供则自动生成）
   - 写入 config.yaml
2. 无需手动 Setup 向导——配好 `.env` 直接启动即可
3. 若未设置 `ADMIN_PASSWORD`，从日志查找生成的密码：
   ```bash
   docker compose logs sub2api | grep "admin password"
   ```

#### 数据库迁移说明（PostgreSQL）

- 迁移按文件名字典序执行（如 `001_...sql`、`002_...sql`）
- `schema_migrations` 记录已执行的迁移（文件名 + 校验和）
- 迁移仅向前（forward-only）；回滚需恢复数据库备份或手写补偿 SQL

**校验 `users.allowed_groups` → `user_allowed_groups` 回填**

在 GORM→Ent 增量迁移过程中，`users.allowed_groups`（旧的 `BIGINT[]`）被规范化连接表 `user_allowed_groups(user_id, group_id)` 替代。运行以下查询对比旧数据与新连接表：

```sql
WITH old_pairs AS (
  SELECT DISTINCT u.id AS user_id, x.group_id
  FROM users u
  CROSS JOIN LATERAL unnest(u.allowed_groups) AS x(group_id)
  WHERE u.allowed_groups IS NOT NULL
)
SELECT
  (SELECT COUNT(*) FROM old_pairs)           AS old_pair_count,
  (SELECT COUNT(*) FROM user_allowed_groups) AS new_pair_count;
```

#### datamanagementd（数据管理）联动

如需启用管理后台“数据管理”功能，请额外部署宿主机 `datamanagementd`：

- 主进程固定探测 `/tmp/sub2api-datamanagement.sock`
- Docker 场景下需把宿主机 Socket 挂载到容器内同路径
- 详细步骤见下文「数据管理守护进程 datamanagementd」小节，或 `deploy/DATAMANAGEMENTD_CN.md`

#### 常用命令

**本地目录版本**（docker-compose.local.yml）：

```bash
# 启动服务
docker compose -f docker-compose.local.yml up -d

# 停止服务
docker compose -f docker-compose.local.yml down

# 查看日志
docker compose -f docker-compose.local.yml logs -f sub2api

# 仅重启 Sub2API
docker compose -f docker-compose.local.yml restart sub2api

# 更新到最新版本
docker compose -f docker-compose.local.yml pull
docker compose -f docker-compose.local.yml up -d

# 删除所有数据（谨慎！）
docker compose -f docker-compose.local.yml down
rm -rf data/ postgres_data/ redis_data/
```

**命名卷版本**（docker-compose.yml）：

```bash
# 启动服务
docker compose up -d

# 停止服务
docker compose down

# 查看日志
docker compose logs -f sub2api

# 仅重启 Sub2API
docker compose restart sub2api

# 更新到最新版本
docker compose pull
docker compose up -d

# 删除所有数据（谨慎！）
docker compose down -v
```

#### 环境变量（Docker Compose）

| 变量 | 必需 | 默认 | 说明 |
|------|------|------|------|
| `POSTGRES_PASSWORD` | **是** | - | PostgreSQL 密码 |
| `JWT_SECRET` | **推荐** | *(自动生成)* | JWT 密钥（固定可持久化会话） |
| `TOTP_ENCRYPTION_KEY` | **推荐** | *(自动生成)* | TOTP 加密密钥（固定可持久化 2FA） |
| `SERVER_PORT` | 否 | `8080` | 服务端口 |
| `ADMIN_EMAIL` | 否 | `admin@sub2api.local` | 管理员邮箱 |
| `ADMIN_PASSWORD` | 否 | *(自动生成)* | 管理员密码 |
| `TZ` | 否 | `Asia/Shanghai` | 时区 |
| `GEMINI_OAUTH_CLIENT_ID` | 否 | *(内置)* | Google OAuth client ID（Gemini OAuth）。留空则使用内置 Gemini CLI client。 |
| `GEMINI_OAUTH_CLIENT_SECRET` | 否 | *(内置)* | Google OAuth client secret（Gemini OAuth）。留空则使用内置 Gemini CLI client。 |
| `GEMINI_OAUTH_SCOPES` | 否 | *(默认)* | OAuth scopes（Gemini OAuth） |
| `GEMINI_QUOTA_POLICY` | 否 | *(空)* | Gemini 本地额度模拟的 JSON 覆盖（仅 Code Assist） |

全部可用选项见 `.env.example`。

> 注：`docker-deploy.sh` 脚本会自动为你生成 `JWT_SECRET`、`TOTP_ENCRYPTION_KEY`、`POSTGRES_PASSWORD`。

#### 便捷迁移（本地目录版本）

使用 `docker-compose.local.yml` 时所有数据都存于本地目录，迁移很简单：

```bash
# 源服务器：停止服务并打包
cd /path/to/deployment
docker compose -f docker-compose.local.yml down
cd ..
tar czf sub2api-complete.tar.gz deployment/

# 传输到新服务器
scp sub2api-complete.tar.gz user@new-server:/path/to/destination/

# 新服务器：解压并启动
tar xzf sub2api-complete.tar.gz
cd deployment/
docker compose -f docker-compose.local.yml up -d
```

整个部署（配置 + 数据）即完成迁移。

---

### Docker 镜像快速上手（裸 docker run）

如果只想用现成镜像（外部已有 PostgreSQL / Redis），可直接 `docker run`：

```bash
docker run -d \
  --name sub2api \
  -p 8080:8080 \
  -e DATABASE_URL="postgres://user:pass@host:5432/sub2api" \
  -e REDIS_URL="redis://host:6379" \
  weishaw/sub2api:latest
```

自带数据库的最小 Docker Compose：

```yaml
version: '3.8'

services:
  sub2api:
    image: weishaw/sub2api:latest
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=postgres://postgres:postgres@db:5432/sub2api?sslmode=disable
      - REDIS_URL=redis://redis:6379
    depends_on:
      - db
      - redis

  db:
    image: postgres:15-alpine
    environment:
      - POSTGRES_USER=postgres
      - POSTGRES_PASSWORD=postgres
      - POSTGRES_DB=sub2api
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    volumes:
      - redis_data:/data

volumes:
  postgres_data:
  redis_data:
```

镜像环境变量：

| 变量 | 说明 | 必需 | 默认 |
|------|------|------|------|
| `DATABASE_URL` | PostgreSQL 连接串 | 是 | - |
| `REDIS_URL` | Redis 连接串 | 是 | - |
| `PORT` | 服务端口 | 否 | `8080` |
| `GIN_MODE` | Gin 框架模式（`debug`/`release`） | 否 | `release` |

支持架构：`linux/amd64`、`linux/arm64`。

镜像标签：`latest`（最新稳定版）、`x.y.z`（指定版本）、`x.y`（次版本最新补丁）、`x`（主版本最新次版本）。

---

### 二进制安装（systemd）

适用于使用 systemd 的生产服务器。

#### 一行安装

```bash
curl -sSL https://raw.githubusercontent.com/Wei-Shaw/sub2api/main/deploy/install.sh | sudo bash
```

#### 手动安装

1. 从 [GitHub Releases](https://github.com/Wei-Shaw/sub2api/releases) 下载最新版本
2. 解压并把二进制复制到 `/opt/sub2api/`
3. 复制 `sub2api.service` 到 `/etc/systemd/system/`
4. 执行：
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable sub2api
   sudo systemctl start sub2api
   ```
5. 浏览器打开 Setup 向导完成配置

#### 安装脚本命令

```bash
# 安装
sudo ./install.sh

# 升级
sudo ./install.sh upgrade

# 卸载
sudo ./install.sh uninstall
```

#### 服务管理

```bash
sudo systemctl start sub2api      # 启动
sudo systemctl stop sub2api       # 停止
sudo systemctl restart sub2api    # 重启
sudo systemctl status sub2api     # 查看状态
sudo journalctl -u sub2api -f     # 查看日志
sudo systemctl enable sub2api     # 开机自启
```

#### 配置

**服务监听地址与端口**：安装时会提示配置，存于 systemd 服务文件的环境变量。安装后修改：

1. 编辑 systemd 服务：
   ```bash
   sudo systemctl edit sub2api
   ```
2. 添加或修改：
   ```ini
   [Service]
   Environment=SERVER_HOST=0.0.0.0
   Environment=SERVER_PORT=3000
   ```
3. 重载并重启：
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl restart sub2api
   ```

**Gemini OAuth 配置**：如需使用 AI Studio OAuth，把 OAuth client 凭据加入 systemd 服务文件：

1. 编辑服务文件：
   ```bash
   sudo nano /etc/systemd/system/sub2api.service
   ```
2. 在 `[Service]` 段（现有 `Environment=` 行之后）添加：
   ```ini
   Environment=GEMINI_OAUTH_CLIENT_ID=your-client-id.apps.googleusercontent.com
   Environment=GEMINI_OAUTH_CLIENT_SECRET=GOCSPX-your-client-secret
   ```
   如需使用“内置 Gemini CLI OAuth Client”（Code Assist / Google One），还需注入：
   ```ini
   Environment=GEMINI_CLI_OAUTH_CLIENT_SECRET=GOCSPX-your-built-in-secret
   ```
3. 重载并重启：
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl restart sub2api
   ```

> 注：Code Assist OAuth 无需任何配置——使用内置 Gemini CLI client。

**应用配置文件**：主配置位于 `/etc/sub2api/config.yaml`（由 Setup 向导创建）。

#### 前置依赖

- Linux 服务器（Ubuntu 20.04+、Debian 11+、CentOS 8+ 等）
- PostgreSQL 14+
- Redis 6+
- systemd

#### 目录结构

```
/opt/sub2api/
├── sub2api              # 主二进制
├── sub2api.backup       # 备份（升级后）
└── data/                # 运行时数据

/etc/sub2api/
└── config.yaml          # 配置文件
```

---

### Gemini OAuth 配置

Sub2API 支持三种方式接入 Gemini。

#### 方式 1：Code Assist OAuth（推荐给 GCP 用户）

**无需配置**——始终使用内置的 Gemini CLI OAuth client（public）。

1. 保持 `GEMINI_OAUTH_CLIENT_ID` 和 `GEMINI_OAUTH_CLIENT_SECRET` 为空
2. 在管理后台创建 Gemini OAuth 账户，类型选 **"Code Assist"**
3. 在浏览器完成 OAuth 流程

> 注：即使你为 AI Studio OAuth 配置了 `GEMINI_OAUTH_CLIENT_ID` / `GEMINI_OAUTH_CLIENT_SECRET`，Code Assist OAuth 仍会使用内置 Gemini CLI client。

**要求**：拥有 GCP 访问权限的 Google 账户、一个 GCP 项目（自动检测或手动指定）。

**获取 Project ID（自动检测失败时）**：
1. 打开 [Google Cloud Console](https://console.cloud.google.com/)
2. 点击顶部项目下拉
3. 从列表复制 Project ID（不是项目名）
4. 常见格式：`my-project-123456` 或 `cloud-ai-companion-xxxxx`

#### 方式 2：AI Studio OAuth（普通 Google 账户）

需要你自己的 OAuth client 凭据。

**步骤 1：在 Google Cloud Console 创建 OAuth Client**

1. 打开 [Google Cloud Console - Credentials](https://console.cloud.google.com/apis/credentials)
2. 新建或选择项目
3. **启用 Generative Language API**：APIs & Services → Library → 搜索 "Generative Language API" → Enable
4. **配置 OAuth 同意屏幕**（若未做）：
   - APIs & Services → OAuth consent screen
   - 选 "External" 用户类型
   - 填写 app 名称、用户支持邮箱、开发者联系方式
   - 添加 scopes：`https://www.googleapis.com/auth/generative-language.retriever`（可选加 `https://www.googleapis.com/auth/cloud-platform`）
   - 添加测试用户（你的 Google 账户邮箱）
5. **创建 OAuth 2.0 凭据**：
   - APIs & Services → Credentials → Create Credentials → OAuth client ID
   - Application type：**Web application**（或 **Desktop app**）
   - Name：如 "Sub2API Gemini"
   - Authorized redirect URIs：添加 `http://localhost:1455/auth/callback`
6. 复制 **Client ID** 与 **Client Secret**
7. **⚠️ 发布到 Production（重要）**：
   - APIs & Services → OAuth consent screen → 点击 "PUBLISH APP" 从 Testing 切到 Production
   - **Testing 模式限制**：仅手动添加的测试用户可认证（最多 100 人）；refresh token 7 天后过期；用户需定期重新添加
   - **Production 模式**：任意 Google 用户可认证，token 不过期
   - 注：敏感 scope 下 Google 可能要求验证（演示视频、隐私政策）

**步骤 2：配置环境变量**

```bash
GEMINI_OAUTH_CLIENT_ID=your-client-id.apps.googleusercontent.com
GEMINI_OAUTH_CLIENT_SECRET=GOCSPX-your-client-secret

# 可选：如需使用 Gemini CLI 内置 OAuth Client（Code Assist / Google One）
# 安全说明：本仓库不会内置该 client_secret，请在运行环境通过环境变量注入。
# GEMINI_CLI_OAUTH_CLIENT_SECRET=GOCSPX-your-built-in-secret
```

**步骤 3：在管理后台创建账户**

1. 创建 Gemini OAuth 账户，类型选 **"AI Studio"**
2. 完成 OAuth 流程
   - 同意后浏览器会重定向到 `http://localhost:1455/auth/callback?code=...&state=...`
   - 复制完整回调 URL（推荐）或仅 `code`，粘回管理后台

#### 方式 3：API Key（最简单）

1. 打开 [Google AI Studio](https://aistudio.google.com/app/apikey)
2. 点击 "Create API key"
3. 在管理后台创建 Gemini **API Key** 账户
4. 粘贴 API key（以 `AIza...` 开头）

#### 对比表

| 特性 | Code Assist OAuth | AI Studio OAuth | API Key |
|------|-------------------|-----------------|---------|
| 配置复杂度 | 简单（无需配置） | 中（需 OAuth client） | 简单 |
| 需要 GCP 项目 | 是 | 否 | 否 |
| 自定义 OAuth Client | 否（内置） | 是（必需） | 不适用 |
| 速率限制 | GCP 配额 | 标准 | 标准 |
| 适合 | GCP 开发者 | 需要 OAuth 的普通用户 | 快速测试 |

---

### TLS 指纹配置

Sub2API 支持 TLS 指纹模拟，使请求看起来像来自官方 Claude CLI（Node.js client）。

> 💡 访问 **[tls.sub2api.org](https://tls.sub2api.org/)** 获取不同设备和浏览器的 TLS 指纹信息。

**默认行为**：

- 内置 `claude_cli_v2` profile 模拟 Node.js 20.x + OpenSSL 3.x
- JA3 Hash：`1a28e69016765d92e3b381168d68922c`
- JA4：`t13d5911h1_a33745022dd6_1f22a2ca17c4`
- Profile 选择：`accountID % profileCount`

**配置**：

```yaml
gateway:
  tls_fingerprint:
    enabled: true  # 全局开关
    profiles:
      # 简单 profile（使用默认 cipher suites）
      profile_1:
        name: "Profile 1"

      # 带自定义 cipher suites 的 profile（紧凑数组格式）
      profile_2:
        name: "Profile 2"
        cipher_suites: [4866, 4867, 4865, 49199, 49195, 49200, 49196]
        curves: [29, 23, 24]
        point_formats: 0

      # 另一个自定义 profile
      profile_3:
        name: "Profile 3"
        cipher_suites: [4865, 4866, 4867, 49199, 49200]
        curves: [29, 23, 24, 25]
```

**Profile 字段**：

| 字段 | 类型 | 说明 |
|------|------|------|
| `name` | string | 显示名（必需） |
| `cipher_suites` | []uint16 | 十进制 cipher suites，空 = 默认 |
| `curves` | []uint16 | 十进制椭圆曲线，空 = 默认 |
| `point_formats` | []uint8 | EC point formats，空 = 默认 |

**常用值参考**：

- Cipher Suites（TLS 1.3）：`4865`（AES_128_GCM）、`4866`（AES_256_GCM）、`4867`（CHACHA20）
- Cipher Suites（TLS 1.2）：`49195`、`49196`、`49199`、`49200`（ECDHE 变体）
- Curves：`29`（X25519）、`23`（P-256）、`24`（P-384）、`25`（P-521）

---

### 故障排查

#### Docker

**本地目录版本**：

```bash
# 容器状态
docker compose -f docker-compose.local.yml ps

# 详细日志
docker compose -f docker-compose.local.yml logs --tail=100 sub2api

# 数据库连接
docker compose -f docker-compose.local.yml exec postgres pg_isready

# Redis 连接
docker compose -f docker-compose.local.yml exec redis redis-cli ping

# 重启全部服务
docker compose -f docker-compose.local.yml restart

# 检查数据目录
ls -la data/ postgres_data/ redis_data/
```

**命名卷版本**：

```bash
docker compose ps
docker compose logs --tail=100 sub2api
docker compose exec postgres pg_isready
docker compose exec redis redis-cli ping
docker compose restart
```

#### 二进制安装

```bash
sudo systemctl status sub2api          # 服务状态
sudo journalctl -u sub2api -n 50       # 近期日志
sudo cat /etc/sub2api/config.yaml      # 配置文件
sudo systemctl status postgresql       # PostgreSQL
sudo systemctl status redis            # Redis
```

#### 常见问题

1. **端口被占用**：修改 `.env` 或 systemd 配置中的 `SERVER_PORT`
2. **数据库连接失败**：检查 PostgreSQL 是否运行、凭据是否正确
3. **Redis 连接失败**：检查 Redis 是否运行、密码是否正确
4. **Permission denied**：二进制安装需确保文件属主正确

---

## 数据管理守护进程 datamanagementd

在宿主机部署 `datamanagementd`，并与主进程联动开启“数据管理”功能。（详见 `deploy/DATAMANAGEMENTD_CN.md`）

### 1. 关键约束

- 主进程固定探测路径：`/tmp/sub2api-datamanagement.sock`
- 仅当该 Unix Socket 可连通且 `Health` 成功时，后台“数据管理”才会启用
- `datamanagementd` 使用 SQLite 持久化元数据，不依赖主库

### 2. 宿主机构建与运行

```bash
cd /opt/sub2api-src/datamanagement
go build -o /opt/sub2api/datamanagementd ./cmd/datamanagementd

mkdir -p /var/lib/sub2api/datamanagement
chown -R sub2api:sub2api /var/lib/sub2api/datamanagement
```

手动启动示例：

```bash
/opt/sub2api/datamanagementd \
  -socket-path /tmp/sub2api-datamanagement.sock \
  -sqlite-path /var/lib/sub2api/datamanagement/datamanagementd.db \
  -version 1.0.0
```

### 3. systemd 托管（推荐）

仓库已提供示例服务文件：`deploy/sub2api-datamanagementd.service`

```bash
sudo cp deploy/sub2api-datamanagementd.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now sub2api-datamanagementd
sudo systemctl status sub2api-datamanagementd
```

查看日志：

```bash
sudo journalctl -u sub2api-datamanagementd -f
```

也可使用一键安装脚本（自动安装二进制 + 注册 systemd）：

```bash
# 方式一：使用现成二进制
sudo ./deploy/install-datamanagementd.sh --binary /path/to/datamanagementd

# 方式二：从源码构建后安装
sudo ./deploy/install-datamanagementd.sh --source /path/to/sub2api
```

### 4. Docker 部署联动

若 `sub2api` 运行在 Docker 容器中，需要将宿主机 Socket 挂载到容器同路径：

```yaml
services:
  sub2api:
    volumes:
      - /tmp/sub2api-datamanagement.sock:/tmp/sub2api-datamanagement.sock
```

建议在 `docker-compose.override.yml` 中维护该挂载，避免覆盖主 compose 文件。

### 5. 依赖检查

`datamanagementd` 执行备份时依赖以下工具：

- `pg_dump`
- `redis-cli`
- `docker`（仅 `source_mode=docker_exec` 时）

缺失依赖会导致对应任务失败，并在任务详情中体现错误信息。

---

## CDN 缓存规则

适用场景：国内用户访问慢，不想迁服务器，希望通过 Cloudflare / EdgeOne 等 CDN 做缓存加速。**不改代码，只配面板规则。**

### 项目结构回顾

- 后端：Go（go-zero + Gin），默认端口 8080
- 前端：Vue 3 + Vite，构建产物输出到 `backend/internal/web/dist`，嵌入 Go 二进制
- 前端产物带内容 hash（`vendor-vue.[hash].js` 等），天然适合长缓存
- 对外路由前缀：`/`（SPA）、`/api/v1/*`、`/v1/*`、`/v1beta/*`、`/antigravity/*`、`/setup/*`、`/payment/webhook/*`、`/health`

### 规则顺序（重要）

面板里从上往下按顺序命中，**第一条命中即停**。顺序错了会出事故。

#### ① 必须 Bypass（最高优先级）

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

#### ② 短缓存（Edge 5min / Browser 60s，GET only）

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

#### ③ 强缓存（1 年 immutable）

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

#### ④ 首页 HTML —— 不缓存

```
/
/index.html
```

**配置**：Bypass 或 TTL=0。

**理由**：`index.html` 是运行时动态注入站点配置的，缓存了会出现设置不同步。

### 面板配置注意事项

**Cloudflare**：

- 用 **Cache Rules**（不是老的 Page Rules）
- 按上面顺序创建，Bypass 放最上面
- **橙云必须开**（代理模式），不然 CDN 根本没接上
- 开启 **Tiered Cache**（免费档也可用）
- 开启 **Brotli / Gzip**、**HTTP/3** / 0-RTT

**EdgeOne（腾讯）**：

- 规则里显式勾选“仅缓存 GET/HEAD”，POST 永远不能进缓存
- 免费个人套餐：国内节点需域名备案；未备案则只能用海外节点
- 回源 Host 记得设成部署平台分配的域名

### 验证

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

### 预期效果与局限

**会明显变快**：

- 前端打开速度（静态资源边缘命中）
- 未登录态访问首页的后续资源加载
- 公开元数据接口（`/v1/models` 等）

**收益有限**：

- 登录后的用户态 API（本来就 bypass）
- LLM 推理请求（POST、流式，无法缓存，只能靠链路优化）

**真正想让推理也快**，要么：

1. 给国内用户加一层就近反向代理入口（仍不迁主服务）
2. 换更靠近大陆的部署区域（香港/东京/新加坡）
3. 对推理做分区域路由

---

## 相关

- 发布与上线流程见 [[workflow]]（dev → publish → tag）。
- 外链：[GitHub 仓库](https://github.com/Wei-Shaw/sub2api)、[Docker Hub](https://hub.docker.com/r/weishaw/sub2api)、[TLS 指纹工具 tls.sub2api.org](https://tls.sub2api.org/)。
- **平台说明**：CDN 缓存规则方案原文以 Zeabur 为部署平台撰写，但当前服务实际平台可能已变更（历史文档为 Zeabur，以实际平台为准）。部署或配置 CDN 回源前请先确认当前平台与分配域名。
