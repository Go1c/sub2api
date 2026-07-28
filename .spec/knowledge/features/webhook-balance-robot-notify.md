---
name: webhook-notify
description: 个人资料 Webhook 通知：余额/站内信/公告经 HTTPS POST 推送；无浏览器 WebSocket
metadata:
  type: doc
  level: L2
  status: 实现中
---

# Webhook 通知

## 一句话

非管理员在个人资料配置 **Webhook**（仅此名称）。启用后，余额首次跌破阈值、收到站内信、收到公告时，服务端向配置的 **https** URL POST 通知。

## 非目标

- 浏览器 WebSocket 实时推送（已移除）
- 产品文案不绑定某一厂商品牌

## 配置

- 启用（默认关）
- Webhook URL
- 余额阈值（默认 $10）
- 站内信通知 / 公告通知子开关（默认开，受总开关约束）
- 发送测试

## 通道关系

- 邮件余额提醒：独立
- Webhook：本功能
