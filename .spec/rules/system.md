# System Rules(系统规则 · 每次 Agent 初始化强制加载)

本文件经 `CLAUDE.md` 的 `@import` **在每次 Agent 初始化时强制载入上下文**——始终在场的硬红线,不走渐进式披露。
只写「**必须 / 只能 / 不得**」:什么必须做、什么禁止做。不写「怎么做 / 是什么」(那在 `knowledge/standards/` 或 [`AGENTS.md`](../AGENTS.md))。
新增系统级硬规则,直接在本文件加一节;若日后出现「非启动必载」的情境化规则,再另立文件并相应决定是否 `@import`。

## 协作 / 调度

- **子 Agent 不得再派生别的子 Agent。** 调度权只在主 loop;被调用的子 Agent 只执行、不再派活(也符合宿主限制:subagent 不能再 spawn subagent)。
- **子 Agent 的 frontmatter 只用 `name` + `description`。** 其余(追求什么、用哪些技能、不做什么)写进正文;`role` / `goal` / `tools` 等不被宿主据以调度,写了不生效。
- **改动若影响调度关系,必须同步更新 [`AGENTS.md`](../AGENTS.md) 的「调度核心」(名册 / 流程)。** 漏更新 = 主 loop 调度依据失真。

## fork 红线(sub2api / LumioAPI 专属)

- **不得直推 `main` / `publish`。** 这两个分支只接受合并 / tag 操作;日常 commit 全部走 `dev`。
- **`gh pr create` / `gh pr merge` 必须带 `--repo Go1c/sub2api`。** 缺省会指向 upstream;hotfix 也先进 `dev`,再由 dev 提升到 `publish`,不得用 `--base publish` 开 PR。
- **代码改动 commit 前必须本地验证。** 前端改动:`cd frontend && pnpm typecheck && pnpm build`;涉及 UI 必须 dev server 起来 curl 过或请求用户浏览器确认。只凭类型检查不得报"完成"。
- **改 `backend/` / `frontend/` 必须控制范围。** 优先最小必要改动,不顺手重构,避免增加 upstream 同步负担。
- **同步上游不得整包合并。** 走 merge-upstream API 同步 `main`,再经 `dev` 的 T1 / T2 / T3 topic 分支分主题带入。
- **后端集成测试必须用 build tag 校验:** `go vet -tags integration ./...`(否则本地 build / test 漏检)。
