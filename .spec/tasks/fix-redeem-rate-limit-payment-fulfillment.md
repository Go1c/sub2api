---
status: completed
---

# Fix Redeem Rate Limit Payment Fulfillment

## Task

将普通兑换码失败限流窗口从 24 小时缩短为 1 小时，并让支付成功后的可信余额入账仅绕过用户兑换失败限流，继续保留兑换码校验、分布式锁、事务、幂等、订单状态与返利去重保护。

## Acceptance

- `redeem:ratelimit:<userID>` 继续按用户隔离，阈值保持 20 次，TTL 为 1 小时。
- 普通手动兑换继续检查限流并记录无效兑换失败次数，达到阈值后返回 `429 / REDEEM_RATE_LIMITED`。
- 支付余额履约不检查也不增加用户兑换失败计数，但仍执行兑换码校验、锁、事务、余额更新和已使用标记。
- 支付履约成功后订单为 `COMPLETED` 并记录 `RECHARGE_SUCCESS`，重试不重复增加余额或返利。
- 修改过的 Go 文件已 `gofmt`，后端测试、integration vet 与 lint 完成或准确记录环境阻断。

## Knowledge

更新 `.spec/knowledge/features/payment.md`，记录可信支付履约的严格绕过边界与现有 Redis Key 部署注意事项。

## Verification Results

- 基线：从最新 `origin/dev` 的 `cef36bdf6` 创建 `fix/redeem-rate-limit-payment-fulfillment-20260711`；旧基线 WIP 已完整保存在 stash `wip/old-base-redeem-payment-fulfillment-20260711`。
- 红：`go test -tags=unit ./internal/service -run '^TestExecuteBalanceFulfillmentBypassesRedeemRateLimitAndRemainsIdempotent$' -count=1` 按预期失败，错误为 `redeem balance: ... code=429 reason="REDEEM_RATE_LIMITED"`，确认已支付余额订单仍被手动兑换限流拦截。
- 绿：同一回归测试转绿；支付专用入口的 missing / used / expired 严格保护、手动兑换 429 / 失败计数、支付锁保留、订单 `COMPLETED`、`RECHARGE_SUCCESS` 与重试不重复余额的定向测试全部通过。
- `go test -tags=unit ./internal/service ./internal/repository -count=1`：通过。
- `go test -tags=integration ./internal/repository -run '^TestRedeemCacheSuite$' -count=1`：通过，确认既有 Lua 固定 1 小时窗口继续生效。
- `go test -p 1 -tags=unit ./...`：全仓通过。
- `go test -p 1 -tags=integration ./...`：全仓通过。
- `go vet -p 1 -tags integration ./...`：通过。
- CI 同版本 `go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.9.0 run ./...`：`0 issues.`（本机 `golangci-lint` 不在 PATH，故使用固定版本 `go run`）。
- 修改过的 Go 文件已 `gofmt`；最终 diff、`git diff --check` 与固定窗口回退检查在提交前完成。
