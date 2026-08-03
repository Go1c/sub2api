---
status: cancelled
title: 余额低告警站内信 — 后端（已作废：通道改为 WebSocket）
---

# 余额低告警站内信 — Backend

## 做什么

扩展既有余额低告警：用户可开启「站内信」通道；扣费余额首次跌破阈值时，除邮件外（或仅勾选站内信时）写入一条系统站内信。

## 设计依据

- `.spec/knowledge/features/balance-low-site-message-notify.md`

## 涉及范围

- 新 migration：`users.balance_notify_site_message_enabled boolean NOT NULL DEFAULT false`
- Ent schema `backend/ent/schema/user.go` + 生成/字段映射
- domain / repo / DTO：`User`、`handler/dto`、`user_handler` update、`user_service` update
- `backend/internal/service/balance_notify_service.go`：注入站内信写入；穿越后 dispatch
- Wire：`ProvideBalanceNotifyService` / `wire_gen` 相关
- 单测：`balance_notify_check_test.go` 等
- 若有：`api_contract_test` 中 user profile 字段

## 实现要点

1. **字段读写**
   - Create/Update user 映射新字段
   - `UpdateProfile` 接受 `balance_notify_site_message_enabled *bool`
2. **发送**
   - 窄接口例如：`SendBalanceLowSiteMessage(ctx, userID, subject, content) error`
   - 实现：`SiteMessageRepository.Create`，`SenderID=RecipientID=userID`
   - 前置：`user.BalanceNotifyEnabled && user.BalanceNotifySiteMessageEnabled`
   - 前置：全局 `balance_low_notify_enabled`
   - 前置：`site_messages_enabled == true`（读 setting）
   - 与邮件并行，互不 `return` 阻断
3. **文案**：余额、阈值、可选充值 URL；subject/content 非空
4. **错误**：Create 失败 slog，不 panic

## 验收标准

- [ ] migration 可在干净库应用；默认 `false`
- [ ] 用户 update profile 可读写 `balance_notify_site_message_enabled`
- [ ] 穿越阈值 + 用户总开关开 + 站内信勾选开 + 站内信功能开 → 恰好 1 条站内信
- [ ] 任一开关关闭或未穿越 → 0 条
- [ ] 站内信失败不影响扣费路径与邮件发送
- [ ] `go test` 覆盖上述分支（unit tag 与项目惯例一致）
- [ ] 不引入真实 WebSocket 推送

## 依赖

无（可先于前端合并；前端未上线时字段默认 false 无行为变化）
