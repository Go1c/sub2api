---
status: completed
---

# 用户侧 API 暴露 affiliate 运行规则配置

`GET /api/v1/user/aff` 的 `AffiliateDetail` 增加内嵌 `rules`，透传冻结小时 / 有效期天数 / 单被邀人上限，以及注册赠送开关与额度。不改管理端、计提/绑定、schema。
