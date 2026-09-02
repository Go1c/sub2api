---
status: completed
---

# Lumio Desktop 支付交接启用与知识同步

## 做什么

在后端与前端链路均完成后开启安全默认的 `payment_handoff`，同步桌面契约知识和 API 期望，并执行全量收口验证。

## 涉及范围

- `backend/internal/service/lumio_desktop_config.go` 与测试
- `backend/internal/server/api_contract_test.go`
- `.spec/knowledge/features/lumio-desktop-client.md`
- `docs/specs/`、`docs/plans/`

## 接口

- Consumes: backend/前端两张支付交接卡
- Produces: effective `feature_flags.payment_handoff=true` when global payment is enabled

## 验收标准

- [ ] 内置 payment_handoff 默认开启，但全局支付关闭仍压制为 false
- [ ] 公开配置/API contract 测试与新默认一致
- [ ] 桌面知识文档记录端点、Cookie、安全边界与剩余范围
- [ ] backend unit/integration/vet/增量 lint 通过
- [ ] frontend typecheck/test/build 通过
- [ ] 不推送、不部署、不创建公开 PR

## 依赖

依赖 `lumio-desktop-payment-handoff-backend` 与 `lumio-desktop-payment-handoff-frontend`。
