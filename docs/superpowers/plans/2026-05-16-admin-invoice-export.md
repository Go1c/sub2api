# Admin Invoice Export Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add Excel export actions to the admin invoice records page for all invoice requests and processing-only invoice requests.

**Architecture:** Keep the backend unchanged and reuse the existing admin invoice list endpoint. Add a focused frontend export utility for Excel row mapping and workbook writing, then wire it into the admin invoice page with localized labels and user feedback.

**Tech Stack:** Vue 3, TypeScript, Vue i18n, Vitest, xlsx.

---

### Task 1: Export Utility

**Files:**
- Create: `frontend/src/utils/adminInvoiceExport.ts`
- Create: `frontend/src/utils/__tests__/adminInvoiceExport.spec.ts`

- [ ] **Step 1: Write the failing test**

Create `frontend/src/utils/__tests__/adminInvoiceExport.spec.ts` with tests that assert an invoice record maps to Chinese Excel headers, `processing` maps to `正在开票`, money values remain numeric, and missing optional fields become empty strings.

- [ ] **Step 2: Run test to verify it fails**

Run: `pnpm --dir frontend test:run -- src/utils/__tests__/adminInvoiceExport.spec.ts`

Expected: FAIL because `@/utils/adminInvoiceExport` does not exist.

- [ ] **Step 3: Implement the utility**

Create `frontend/src/utils/adminInvoiceExport.ts` exporting `buildAdminInvoiceExportRows`, `buildAdminInvoiceExportFileName`, and `downloadAdminInvoiceWorkbook`.

- [ ] **Step 4: Run test to verify it passes**

Run: `pnpm --dir frontend test:run -- src/utils/__tests__/adminInvoiceExport.spec.ts`

Expected: PASS.

### Task 2: Admin Invoice Page Wiring

**Files:**
- Modify: `frontend/src/views/admin/InvoicesView.vue`
- Modify: `frontend/src/views/admin/__tests__/InvoicesView.spec.ts`
- Modify: `frontend/src/i18n/locales/zh.ts`
- Modify: `frontend/src/i18n/locales/zh-Hant.ts`
- Modify: `frontend/src/i18n/locales/en.ts`

- [ ] **Step 1: Write the failing source contract**

Update `frontend/src/views/admin/__tests__/InvoicesView.spec.ts` so it expects export labels, `downloadAdminInvoiceWorkbook`, and a call path using `status: 'processing'`.

- [ ] **Step 2: Run test to verify it fails**

Run: `pnpm --dir frontend test:run -- src/views/admin/__tests__/InvoicesView.spec.ts`

Expected: FAIL because the page has no export actions.

- [ ] **Step 3: Implement page wiring**

Add export buttons next to refresh. Implement a `runExport(scope)` function that pages through `adminInvoicesAPI.list`, accumulates all rows, and calls `downloadAdminInvoiceWorkbook`.

- [ ] **Step 4: Add locale keys**

Add localized labels for exporting all invoice records, exporting processing invoice records, export success, empty export, and export failure.

- [ ] **Step 5: Run test to verify it passes**

Run: `pnpm --dir frontend test:run -- src/views/admin/__tests__/InvoicesView.spec.ts`

Expected: PASS.

### Task 3: Focused Verification

**Files:**
- Verify: `frontend/src/utils/__tests__/adminInvoiceExport.spec.ts`
- Verify: `frontend/src/views/admin/__tests__/InvoicesView.spec.ts`
- Verify: `frontend/src/api/__tests__/invoices.spec.ts`
- Verify: `frontend`

- [ ] **Step 1: Run focused tests**

Run: `pnpm --dir frontend test:run -- src/utils/__tests__/adminInvoiceExport.spec.ts src/views/admin/__tests__/InvoicesView.spec.ts src/api/__tests__/invoices.spec.ts`

Expected: PASS.

- [ ] **Step 2: Run typecheck**

Run: `pnpm --dir frontend typecheck`

Expected: PASS.

- [ ] **Step 3: Run build**

Run: `pnpm --dir frontend build`

Expected: PASS.
