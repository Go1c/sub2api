---
status: in_progress
title: Lumio 桌面公开配置接口
---

# Lumio 桌面公开配置接口

## 做什么

新增 `GET /api/v1/desktop/config`，只返回桌面客户端所需的默认模型、充值入口、最低客户端版本、更新提示和功能开关；配置可由现有管理员设置 API 动态维护。

## 涉及范围

- `backend/internal/service/`：typed config、设置解析/保存、安全回退
- `backend/internal/handler/`：公开白名单 DTO、ETag 与缓存头、管理员映射
- `backend/internal/server/routes/public.go`：公开只读路由
- 对应 unit tests

## 验收标准

- [ ] 无 JWT 可读取且响应只含明确白名单字段
- [ ] 空配置返回内置安全默认值
- [ ] 非法 JSON、版本或外部支付 URL 不会进入公开响应
- [ ] 全局注册/支付关闭能压制桌面功能开关
- [ ] 管理员 GET/PUT 可读写 typed desktop config，省略字段不覆盖旧值
- [ ] ETag、304 和 `Cache-Control` 行为有测试
- [ ] 相关 unit tests 通过

## 依赖

无
