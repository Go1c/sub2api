---
status: completed
---

# 管理员签到流水周期统计

## 口径

- 周期（运营时区 `daily_checkin_settings.timezone`，周一为一周起点）：
  - `day`：当天业务日期
  - `week`：本周一至今天
  - `month`：本月 1 日至今天
  - `all`：全部
- `unique_users` / `checkin_count`：周期内全部流水（含奖池耗尽）
- `total_amount` / `avg_amount` / `p50_amount` / `p90_amount` / `max_amount`：仅 `status=awarded` 的 `actual_reward`
- 统计接口可叠加列表的 search / user_id / status，不叠加列表的单日 `business_date`（周期自己管日期）

## 验收

- `GET /api/v1/admin/affiliates/checkins/stats?period=`
- 管理页顶栏可切今日 / 本周 / 本月 / 累计，展示人数、次数、总额、平均、P50、P90、最大
