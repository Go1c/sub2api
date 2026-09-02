---
name: admin-invoice-export
description: 管理员发票记录页的 Excel 导出（全部 / 正在开票）
metadata:
  type: doc
  level: L2
  status: 已交付
---

# 管理员发票记录导出
简介：在管理员发票记录页新增 Excel 导出动作，支持导出「全部开票信息」或仅「正在开票」的发票请求。

## 背景 / 目标
管理员需要把发票记录导出为 `.xlsx`，便于线下处理与归档。导出范围分两类：

- 全部发票信息。
- 正在开票的发票请求（`status === "processing"`，UI 标签 `正在开票`）。

目标页面为 `frontend/src/views/admin/InvoicesView.vue`，复用现有管理员发票列表 API，**无需任何后端改动**。

## 设计
后端保持不变，复用现有 `/admin/invoices` 列表端点。前端新增一个聚焦的导出工具负责 Excel 行映射与工作簿写入，再接入管理员发票页，配合本地化文案和用户反馈。

技术栈：Vue 3、TypeScript、Vue i18n、Vitest、xlsx。

**页面职责（拥有 API 分页与用户反馈）**：在现有刷新按钮旁新增导出控件，暴露两个显式动作：

- `导出全部开票信息`
- `导出正在开票`

页面以较大 page size、携带当前搜索 / 用户筛选，从 `/admin/invoices` 翻页拉取所有页。正在开票导出额外传 `status=processing`；全部导出则省略 `status`。这样保持服务端权限、搜索行为和分页契约不变。`runExport(scope)` 通过 `adminInvoicesAPI.list` 翻页累积所有行，再调用 `downloadAdminInvoiceWorkbook`。

**工具职责（行格式化、工作簿创建、文件下载）**：Excel 生成隔离在 `frontend/src/utils/adminInvoiceExport.ts`，导出 `buildAdminInvoiceExportRows`、`buildAdminInvoiceExportFileName`、`downloadAdminInvoiceWorkbook`。它把 `InvoiceRequest` 记录转换为稳定的中文表头，并用现有 `xlsx` 依赖写出工作簿。导出列为：

`申请单号`、`用户邮箱`、`用户ID`、`发票抬头`、`税号`、`开票金额`、`历史开票金额`、`历史充值金额`、`税点扣除`、`状态`、`接收邮箱`、`申请时间`、`完成时间`、`失败原因`。

行映射规则：`processing` 映射为 `正在开票`；金额值保持数值类型；缺失的可选字段变为空字符串。

**错误处理**：任一页拉取或文件生成失败时，页面弹出本地化错误提示，并保持当前表格状态不变。导出进行期间禁用导出按钮。

## 已决策
- 不改后端，完全复用现有列表 API 与其分页 / 权限 / 搜索契约。
- 导出格式固定 `.xlsx`，复用现有 `xlsx` 依赖。
- 中文列表头固定且稳定；金额保持数值。
- 页面负责分页与反馈，工具负责格式化 / 写盘 / 下载，职责分离。

## 实现
按 TDD 分三个任务：

1. **导出工具**（`adminInvoiceExport.ts` + `__tests__/adminInvoiceExport.spec.ts`）：先写失败测试断言记录映射到中文表头、`processing`→`正在开票`、金额保持数值、缺失字段为空串；再实现三个导出函数。
2. **页面接入**（`InvoicesView.vue`、`InvoicesView.spec.ts` 及 `zh.ts` / `zh-Hant.ts` / `en.ts`）：源契约测试期望导出标签、`downloadAdminInvoiceWorkbook` 调用、以及 `status: 'processing'` 路径；在刷新旁加导出按钮并实现 `runExport(scope)`；新增导出全部 / 导出正在开票 / 导出成功 / 空导出 / 导出失败的本地化文案。
3. **聚焦验证**：跑 `adminInvoiceExport.spec.ts`、`InvoicesView.spec.ts`、`invoices.spec.ts` 三个测试，再跑 `pnpm --dir frontend typecheck` 与 `pnpm --dir frontend build`。

## 相关
- 交叉链接：[[payment]]、[[recharge-invoice-balance-gate]]
- 代码路径：
  - `frontend/src/views/admin/InvoicesView.vue`
  - `frontend/src/utils/adminInvoiceExport.ts`
  - `frontend/src/api`（管理员发票列表 API 包装，测试在 `frontend/src/api/__tests__/invoices.spec.ts`）
  - i18n：`frontend/src/i18n/locales/{zh,zh-Hant,en}.ts`
