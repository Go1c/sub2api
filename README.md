# sub2api / LumioAPI

[![Go](https://img.shields.io/badge/Go-1.25.7-00ADD8.svg)](https://golang.org/)
[![Vue](https://img.shields.io/badge/Vue-3.4+-4FC08D.svg)](https://vuejs.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15+-336791.svg)](https://www.postgresql.org/)
[![Redis](https://img.shields.io/badge/Redis-7+-DC382D.svg)](https://redis.io/)

[`Wei-Shaw/sub2api`](https://github.com/Wei-Shaw/sub2api) 的 fork(线上品牌 **LumioAPI**)。一个面向「订阅额度分发」的 AI API 网关:用户通过平台生成的 API Key 访问上游 AI 服务,平台负责鉴权、计费、负载均衡与请求转发。

技术栈:Go(Gin + Ent)后端 · Vue 3 前端 · PostgreSQL + Redis。

## 文档怎么组织(LumioAgent 规范)

本仓库的规范、知识、子 Agent 与技能统一收在 **`.spec/`**(唯一权威源);根目录入口只是指针。

> 一句话:**主 Agent 调度,子 Agent 执行,Skill 是方法,.md 是规则。**

```
.spec/
├── AGENTS.md          # ★ 中心文档:项目介绍 + Agent 调度,先读它
├── agents/            # 子 Agent 定义(planner / coder)
├── rules/system.md    # 硬性禁令 / 护栏(含 fork 红线)
├── skills/            # 技能库(before-you-code / spec-steward / task-breakdown / TDD)
└── knowledge/         # 项目知识库
    ├── README.md      # 知识导航(先看这里)
    ├── standards/     # 开发规范:工作流 / 测试 / 风格 / 环境
    ├── features/      # 功能设计与记录(支付、订阅、抽奖、站内信……)
    ├── operations/    # 部署与运维
    └── records/       # 历史决策 / 同步台账 / 故障墙
```

### 从哪开始读

1. [`.spec/AGENTS.md`](.spec/AGENTS.md) —— 项目介绍 + 调度核心 + 视觉识别。
2. [`.spec/knowledge/README.md`](.spec/knowledge/README.md) —— 知识库导航,按需下钻。
3. [`.spec/rules/system.md`](.spec/rules/system.md) —— 动手前必读的硬红线。

新增 / 修改 / 维护任何规范或知识 → 用 `spec-steward` 技能,保证放对位置、frontmatter 合规、索引同步。

## 部署

部署、Docker、CDN 缓存与运维见 [`.spec/knowledge/operations/deployment.md`](.spec/knowledge/operations/deployment.md);分支 / 发布流程见 [`.spec/knowledge/standards/workflow.md`](.spec/knowledge/standards/workflow.md)。上游原始安装说明见 [Wei-Shaw/sub2api](https://github.com/Wei-Shaw/sub2api)。

## License

[GNU LGPL v3.0](LICENSE) (or later)。贡献者协议见 [`CLA.md`](CLA.md)。
