# 分支与同步策略

本仓库是 `Wei-Shaw/sub2api` 的 fork，在此基础上做了 **Dragon Code 风格的独立前端** (`frontend-dashboard/`)。分支职责按如下分工，请**不要直接在 `main` 或 `publish` 上提交**。

## 远端

```bash
git remote -v
# origin    https://github.com/Go1c/sub2api        (fetch/push)   ← 你的 fork
# upstream  https://github.com/Wei-Shaw/sub2api    (fetch/push)   ← 原始项目
```

首次配置：

```bash
git remote add upstream https://github.com/Wei-Shaw/sub2api.git
git fetch upstream
```

## 分支角色

| 分支      | 作用 | 允许的操作 |
|-----------|------|-----------|
| `main`    | 仅与 `upstream/main` 同步，不做业务开发 | `merge upstream/main`、`push origin main` |
| `dev`     | 日常开发，所有功能 PR 合入点 | 所有 `feat/*`、`fix/*`、`docs/*` PR → `dev` |
| `publish` | 源码 tag 语义的稳定发布分支，Zeabur 部署源 | 从 `dev` 定期合并，`git tag vX.Y.Z`，`push --tags` |

**推荐的保护规则**（在 GitHub 上设置）：
- `main`：禁止直推，禁止 force push，合并前要求通过 CI。
- `publish`：同上；建议要求 PR 评审。

## 同步上游

定期（建议每周或有新 upstream release 时）：

```bash
git checkout main
git fetch upstream
git merge --ff-only upstream/main    # 失败说明 main 有本地分叉，检查后再处理
git push origin main

# 把 upstream 新变更带进 dev
git checkout dev
git merge main                       # 可能有冲突，解决后提交
git push origin dev
```

> ⚠️ 本仓库新增了 `frontend-dashboard/` 和 `doc/`，upstream 没有这些；与 upstream 合并时冲突应集中在 `backend/` 或 `frontend/`。

## 日常开发

```bash
git checkout dev
git pull
git checkout -b feat/your-change
# ... 写代码，本地验证通过（见 doc/frontend-dashboard.md） ...
git push -u origin feat/your-change
gh pr create --base dev --title "feat: xxx"
```

## 发布到 publish

当 `dev` 稳定且准备部署到 Zeabur：

```bash
git checkout publish
git pull
git merge --no-ff dev -m "release: vX.Y.Z"
git tag -a vX.Y.Z -m "release X.Y.Z"
git push origin publish --tags
```

Zeabur 监听 `publish` 分支（或 tag），自动触发构建/部署。详见 `doc/deploy-zeabur.md`。
