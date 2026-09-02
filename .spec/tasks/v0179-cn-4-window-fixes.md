---
status: completed
---

# 任务：国模第四刀（窗口内 CN 修复）

在底座+网关+前端之后，按依赖叠 origin 窗口 PR（缺 API 就适配）：

| PR | 内容 | 备注 |
|---|---|---|
| #5730 | CN 分组入口与正确性 | 若 2 刀已含网关部分，这里收 groups UI / handler |
| #5773 | 渠道定价 CN 平台 | |
| #5782 | CN 账号测试路由 | |
| #5842 | 自适应 API 协议（测试服务侧） | UI 已在 3 刀 |
| #5847 | CN header overrides | UI 已在 3 刀则只收后端缺口 |
| #5919 | CN 原生 Anthropic reasoning effort | 若 2 刀已含则跳过 |
| #5911 | DeepSeek relay 余额校验 | 若 1 刀已含则跳过 |
| #5913 | DeepSeek Responses 账号测试 | |
| #6009 | CN quota 刷新入口 | 若 3 刀已含则跳过 |
| #6011 | CN Anthropic 协议账号测试 | |
| #5906 | CN 额度探测测试加锁 | 若 1 刀已含则跳过 |
| #5875 | 平台筛选 catalog 收 CN（composite 可等 5 刀一起） | |

## 验收标准

- [ ] 上表每条：合入或注明已在前刀覆盖
- [ ] 改动包 unit + vet integration；前端 typecheck/build
- [ ] 不碰 GROUPING SETS #5762、xai/models 全量、grok_free_quota_gate、publish

## 依赖

v0179-cn-1、建议在 2/3 之后
