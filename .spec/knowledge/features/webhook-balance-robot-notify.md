---
name: webhook-balance-robot-notify
description: 个人资料配置企业微信/外部机器人 Webhook，余额低于阈值时服务端 POST 推送（与邮件、浏览器 WebSocket 解耦）
metadata:
  type: doc
  level: L2
  status: 实现中
---

# 企业微信 / 机器人余额告警（Webhook）

## 一句话

非管理员在个人资料配置 **Webhook URL**（企业微信群机器人优先），启用后当余额**首次跌破**阈值时，服务端向该 URL 发送提醒；**不依赖浏览器打开**。

## 与其它通道关系

| 通道 | 机制 | 是否本功能 |
|------|------|------------|
| 邮件余额提醒 | SMTP | 否（独立，已有） |
| 浏览器 WebSocket | 站内实时 toast | 否（次要；可并存） |
| **机器人 Webhook** | HTTPS POST | **是（主推）** |

## 用户配置

- 入口：个人资料 `/profile` → **企业微信 / 机器人通知**
- 启用（默认关）
- Webhook URL（`https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=...`）
- 独立阈值（默认 $10）
- 发送测试

## 后端

- 字段：`webhook_balance_notify_enabled` / `url` / `threshold`
- 服务：`WebhookBalanceNotifyService`
- 企微 payload：`{"msgtype":"text","text":{"content":"..."}}`
- 扣费钩子：与邮件、WebSocket 并行，独立判断
- 测试：`POST /api/v1/user/webhook-balance-notify/test`（30s 限流）

## 安全

- 仅 `https`
- 拒绝 localhost / `.local`
- 测试限流；发送失败打日志，不阻断扣费
