---
status: completed
title: 站内信历史累计补偿按批次合计展示，不再乘码数
---

# 站内信历史「累计补偿」金额被码数乘大

## 口径（已锁）

`site_message_compensation_batches.amount` = 本批成功发出的补偿合计（不是每人单价）。

- 无补偿 / 全失败：amount = 0
- 管理后台统一面值：amount = round(compensation_amount × success_count, 2)
- 运维脚本按人不同面值：已经是 sum(value)，前端直接加，禁止再乘 code_count
- 「累计补偿」= 非 cancelled 批次的 amount 之和
- 顶栏、列表、详情「批次总额 / 补偿额度」必须同一数字
- 请求字段 `compensation_amount` 仍是每人面值，不要改含义
- 站内信正文里的「补偿额度 {amount}」继续用每人面值
- 不要 UPDATE 生产/本地批次表

## 要做

1. 后端 `AdminSendCompensationBatch` 落库改成批次合计
2. 前端 `totalCompensation` 不再 `amount * codeCount`
3. `loadHistoryAsDraft` 用 amount / codeCount 还原每人额度
4. 补前后端测试
5. 更新 `.spec/knowledge/features/site-messages.md` 记录 amount 语义

## 不要做

- 不要改兑换码校验、SMTP、全员筛选、优惠码复用
- 不要改运维脚本
- 不要做按人不同面值的管理后台重发
- 不要重构站内信页、不要改 i18n（除非断言所需 key 真缺）
