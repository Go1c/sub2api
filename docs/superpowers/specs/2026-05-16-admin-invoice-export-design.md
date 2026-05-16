# Admin Invoice Export Design

## Goal

Add an Excel export action to the admin invoice records page so administrators can export either all invoice requests or only requests that are currently being invoiced.

## Scope

- Target page: `frontend/src/views/admin/InvoicesView.vue`.
- Export format: `.xlsx`.
- Export scopes:
  - All invoice information.
  - Processing invoice requests, where `status === "processing"` and the UI label is `正在开票`.
- Reuse the existing admin invoice list API. No backend API change is required.

## Design

The admin invoice page adds an export control next to the existing refresh button. The control exposes two explicit actions:

- `导出全部开票信息`
- `导出正在开票`

The page fetches all pages from `/admin/invoices` with a large page size and current search/user filters. For the processing export it also passes `status=processing`; for the all export it omits `status`. This keeps existing server-side permissions, search behavior, and pagination contracts intact.

Excel generation is isolated in `frontend/src/utils/adminInvoiceExport.ts`. The utility converts `InvoiceRequest` records into stable Chinese column headers and writes a workbook using the existing `xlsx` dependency. Exported columns are:

`申请单号`, `用户邮箱`, `用户ID`, `发票抬头`, `税号`, `开票金额`, `历史开票金额`, `历史充值金额`, `税点扣除`, `状态`, `接收邮箱`, `申请时间`, `完成时间`, `失败原因`.

The page owns API pagination and user feedback. The utility owns row formatting, workbook creation, and file download.

## Error Handling

If any page fetch or file generation fails, the page shows a localized error toast and leaves the current table state unchanged. The export button is disabled while an export is running.

## Testing

- Unit test the API wrapper to ensure admin invoice list requests preserve pagination and `status=processing`.
- Unit test the export utility with sample invoice data and inspect generated row objects.
- Source-contract test the admin invoice page for both export actions and the processing export path.
