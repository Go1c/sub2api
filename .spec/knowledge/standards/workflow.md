---
name: workflow
description: 开发工作流——分支角色 / 提交 / PR / 上游同步 / 发布；动手改代码、开 PR、发版前查
metadata:
  type: doc
  level: L1
  status: 已交付
---

# 开发工作流(分支 / 提交 / 合并 / 同步)

> 本文是"开发这件事**怎么做**"的手册。Agent 之间**怎么协作**(planner → coder → 收口)在 [`AGENTS.md`](../../AGENTS.md) 的「调度核心」。
> "禁止碰什么"的硬护栏(禁止直推 `main` / `publish`、PR base、上游同步纪律)在 [`rules/system.md`](../../rules/system.md);本文只描述流程,遇护栏处**引用**它。

## 远端配置(必须)

```bash
git remote -v
# origin    https://github.com/Go1c/sub2api          ← 你的 fork
# upstream  https://github.com/Wei-Shaw/sub2api.git   ← 原始项目(必须存在)
```

首次:`git remote add upstream https://github.com/Wei-Shaw/sub2api.git && git fetch upstream`。

## 分支角色

| 分支 | 作用 | 允许的操作 |
|------|------|-----------|
| `main` | 仅与 `upstream/main` 同步,不做业务开发 | `merge upstream/main`、`push origin main` |
| `dev` | 日常开发,所有 PR 合入点 | 所有 `feat/*` / `fix/*` / `docs/*` PR → `dev` |
| `publish` | 源码 tag 语义的稳定发布分支 | 从 `dev` 定期合并,`git tag vX.Y.Z`,`push --tags` |

`main` 保护:`lock_branch` / `enforce_admins` / 禁 force push / 允许 fork syncing——只承担 fork sync 基线,不是默认开发入口。

## 日常开发

```bash
git checkout dev && git pull
git checkout -b feat/your-change
# ... 写代码,本地验证通过(见 testing.md)...
git push -u origin feat/your-change
gh pr create --repo Go1c/sub2api --base dev --title "feat: xxx"
```

## 提交规范

- 格式 `type(scope): subject`,如 `feat(geo): block mainland`、`fix(coder): 修 TDD 步骤`。
- 常用 type:`feat` / `fix` / `refactor` / `chore` / `docs`;scope 可省。
- 一次提交解决一件事;提交前测试通过、无调试残留、知识已同步(见下)。

## 同步上游(不整包合并)

经 merge-upstream API 同步 `main`,再分主题带进 `dev`:

```bash
git checkout main && git fetch upstream && git merge --ff-only upstream/main && git push origin main
git checkout dev && git merge main && git push origin dev   # 冲突常集中在 backend/ frontend/
```

> 大体量上游变更分 T1 / T2 / T3 topic 分支按主题带入,不一次性 wholesale merge。

## 发布到 publish

`publish` 是**受保护分支**:不能直推,只能通过 PR 更新(`gh pr create --base publish`)。
核心不变式 —— **publish ⊆ dev**:promotion 到 publish 的每一处改动都必须**先存在于 dev**。
publish 可以落后 dev(有未发布的工作),但**绝不能有 dev 没有的内容**。

```bash
# 1) 确保要发的内容已经全部在 dev 上、且本地验证通过(见 testing.md)
git fetch origin
# 2) 从 dev 当前提交切发布分支(发布分支 = dev 的精确快照,不在其上新增任何 commit)
git switch -c release/dev-to-publish-$(date +%Y%m%d) origin/dev
git push origin release/dev-to-publish-$(date +%Y%m%d)
# 3) 开 PR 合入 publish(release/* → publish 是 promotion 机制,不属于「hotfix 直接 --base publish」红线)
gh pr create --repo Go1c/sub2api --base publish --head release/dev-to-publish-$(date +%Y%m%d) --title "release: dev → publish YYYY-MM-DD"
gh pr merge  --repo Go1c/sub2api --merge <PR#>
# 4) 发版打 tag(release.yml 由 tag 触发);部署由 push 到 publish 触发(lumio-production.yml)
git tag -a vX.Y.Z -m "release X.Y.Z" && git push origin vX.Y.Z
# 5) 收口校验:两分支内容必须一致(只看内容,不看 commit 数)
git diff --stat origin/dev origin/publish    # 必须为空
```

> **绝不在 release 分支(或 publish)上现补修复。** 发布过程中发现要改:**先把修复提交到 `dev`**(走 `fix/*` PR),再**从 dev 重切** release 分支。在 release 分支上直接补的 commit 不会回到 dev → 两分支内容漂移、dev 上对应测试失败。`.github/workflows/publish-sync-guard.yml` 会在「PR → publish」时校验 PR head 必须是 dev 的祖先,拦下这种现补提交(建议在 branch protection 里把它设为 required check)。
>
> **判断是否漂移只看 `git diff origin/dev origin/publish`(内容),不看 `git rev-list` 的 commit 数。** 每次 promotion 都会在 publish 上产生 merge commit、内容相同的修复也可能各自落成不同 SHA,所以 commit 数差异是正常的图结构假象;**内容 diff 为空**才是两分支一致的唯一标准。
>
> 部署平台:历史为 Zeabur,**当前以实际平台为准**(已迁移,部署前确认)。见 [`operations/deployment.md`](../operations/deployment.md)。

## 改动完成 = 知识已同步

一处改动只有在**知识沉淀完成**后才算 Done:用 `spec-steward` 更新对应 `knowledge/` 文档、`status` 和 `knowledge/README.md` 导航。未同步的改动不得提交 / 合并。

## 相关

- 验收与测试:[`testing.md`](./testing.md) · 环境与坑点:[`dev-environment.md`](./dev-environment.md)
- 风格与视觉:[`code-style.md`](./code-style.md) · 护栏:[`rules/system.md`](../../rules/system.md)
- 同步记录:[`records/`](../records/) · 沉淀方法:`skills/spec-steward`
