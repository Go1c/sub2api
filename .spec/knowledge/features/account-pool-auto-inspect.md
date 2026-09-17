---
name: account-pool-auto-inspect
description: 号池自动巡检：定时扫描请求健康条，成功率过低则加入指定分组并去掉指定模型，401 停调度可通知 Telegram
metadata:
  type: doc
  level: L2
  status: 已实现
---

# 号池自动巡检

账号列表「自动巡检」把一键巡检做成后台配置：按间隔扫描最近请求健康条，账号成功率低于阈值且每条有流量的出口都失败时，自动并入指定分组并去掉指定模型。登录 401 导致停止调度时，可额外通知 Telegram 机器人。

## 背景 / 目标

- 号池里 IP 组账号经常整组 429/打不进去，需要定时把账号降到「降智分组」并摘掉 gpt-6-astra 一类模型。
- 不在网关热路径上改调度；只读现有健康条，再走账号分组绑定和 credentials.model_mapping 更新。
- 401 停调度已经存在，这里只补可开关的机器人通知，避免每 5 分钟重复推送。

## 设计

- **入口**：账号列表刷新按钮旁「自动巡检」。配置存 `settings.account_pool_auto_inspect_config`。
- **周期**：默认 5 分钟，可配 1–1440。关闭时后台跳过且不覆盖上次结果；弹窗「立即执行」先保存当前表单再强制跑一轮，不受 leader lock 挡住。
- **成功率**：读 Redis 请求健康条（窗口 12）。只统计已填充的 ok/fail；空槽和没有流量的 IP 不计。样本数 ≥ 最少样本（默认 4），整体成功率 < 阈值（默认 50%），且每条有样本的出口也都低于阈值，才降级。
- **动作**：`add_group_ids` 与现有分组做并集（保留 Codex，再加降智分组），只绑到平台匹配的账号；保存时校验分组存在。`remove_models` 从非空白名单 `model_mapping` 删掉匹配的 key/value。已经加过组、模型已不在白名单则本轮跳过。空映射（透传全部模型）不改模型。spark 影子账号跳过。
- **401 通知**：扫描 `temp_unschedulable` / `status=error` / 不可调度，原因含 OAuth 401。独立开关；Telegram Token 留空则复用运维「账号异常告警」机器人。冷却默认 60 分钟。
- **降噪**：定时任务用 Redis leader lock，避免多副本重复改账号。单轮最多降级 50 个账号。

## 已决策

- 分组是「加入」不是「替换」，对应截图里 Codex + 降智分组同时存在。
- 不自动恢复：成功率回来后不会自行移出分组或加回模型。
- 不改 IP 组选路和健康条写路径。

## 待解决

- 暂无。

## 相关

- 健康条：[[account-request-health]]
- Telegram 告警（凭证可复用，不要混规则）：[[account-error-alert]]
- 后端：`backend/internal/service/account_pool_auto_inspect.go`
- 后台任务：`backend/internal/service/account_pool_auto_inspect_service.go`
- 管理接口：`GET/PUT /admin/accounts/pool-auto-inspect/config`，`POST /admin/accounts/pool-auto-inspect/run`
- 前端：`frontend/src/components/admin/account/PoolAutoInspectDialog.vue`
