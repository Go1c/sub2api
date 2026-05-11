# Support Chat Gateway Fix and Brand Refresh Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix the authenticated support-chat 400 failure and align the widget styling with the homepage blue-purple brand palette.

**Architecture:** Keep the fix entirely inside `frontend` by normalizing gateway request data at the support-chat API boundary, then update the widget classes to use homepage-aligned gradients and indigo focus states. Validation stays at the unit-test level plus type-check and production build verification.

**Tech Stack:** Vue 3, TypeScript, Vitest, Tailwind CSS, Vite

---

### Task 1: Lock the gateway payload contract with tests

**Files:**
- Modify: `frontend/src/api/supportChat.spec.ts`
- Modify: `frontend/src/components/support/SupportChatWidget.spec.ts`

- [ ] **Step 1: Write the failing API test for numeric user IDs**

```ts
await streamSupportChat(
  {
    message: 'How do I recharge?',
    locale: 'en-US',
    user: { id: 123, email: 'u@example.com' }
  },
  {},
  { gatewayUrl: 'https://gateway.example.com', fetcher }
)

expect(JSON.parse(String(init?.body))).toEqual({
  message: 'How do I recharge?',
  locale: 'en-US',
  user: { id: '123', email: 'u@example.com' }
})
```

- [ ] **Step 2: Run the targeted API test to confirm it fails before the fix**

Run: `cmd /c corepack pnpm vitest run src/api/supportChat.spec.ts`
Expected: FAIL because the current request body still contains `id: 123`.

- [ ] **Step 3: Update the widget spec expectation for logged-in users**

```ts
expect(streamSupportChat).toHaveBeenCalledWith(
  {
    message: 'How do I recharge?',
    locale: 'en-US',
    user: { id: '1', email: 'u@example.com' }
  },
  expect.any(Object),
  { gatewayUrl: 'https://gateway.example.com' }
)
```

- [ ] **Step 4: Run the widget spec to confirm it fails before the fix**

Run: `cmd /c corepack pnpm vitest run src/components/support/SupportChatWidget.spec.ts`
Expected: FAIL because the component currently passes numeric user IDs through unchanged.

### Task 2: Implement the minimal request fix and brand refresh

**Files:**
- Modify: `frontend/src/api/supportChat.ts`
- Modify: `frontend/src/components/support/SupportChatWidget.vue`

- [ ] **Step 1: Normalize gateway user IDs at the API boundary**

```ts
function normalizeSupportChatRequest(request: SupportChatRequest): SupportChatRequest {
  if (!request.user || request.user.id === undefined || request.user.id === null) return request
  return {
    ...request,
    user: {
      ...request.user,
      id: String(request.user.id)
    }
  }
}
```

- [ ] **Step 2: Send the normalized payload in `streamSupportChat`**

```ts
body: JSON.stringify(normalizeSupportChatRequest(request))
```

- [ ] **Step 3: Replace teal action accents with homepage-aligned brand styling**

```vue
:class="message.role === 'user'
  ? 'bg-gradient-to-br from-blue-600 via-indigo-500 to-purple-600 text-white shadow-[0_12px_28px_rgba(99,102,241,0.28)]'
  : 'border border-gray-200 bg-white text-gray-800 dark:border-dark-700 dark:bg-dark-800 dark:text-dark-100'"
```

- [ ] **Step 4: Apply the same palette to the toggle, send button, chips, and focus states**

Run through `SupportChatWidget.vue` and replace the current `primary-*` action classes with blue/indigo/purple gradients or indigo-tinted surfaces while keeping layout and accessibility unchanged.

### Task 3: Verify, build, and publish the branch

**Files:**
- Modify: `frontend/src/components/support/SupportChatWidget.spec.ts`
- Modify: `frontend/src/api/supportChat.spec.ts`

- [ ] **Step 1: Re-run the focused tests after implementation**

Run: `cmd /c corepack pnpm vitest run src/api/supportChat.spec.ts src/components/support/SupportChatWidget.spec.ts`
Expected: PASS.

- [ ] **Step 2: Run type-check and production build**

Run: `cmd /c corepack pnpm vue-tsc --noEmit`
Expected: PASS.

Run: `cmd /c corepack pnpm build`
Expected: PASS and emits the frontend build output.

- [ ] **Step 3: Review git diff and commit the change**

```bash
git status --short
git diff -- frontend/src/api/supportChat.ts frontend/src/components/support/SupportChatWidget.vue frontend/src/api/supportChat.spec.ts frontend/src/components/support/SupportChatWidget.spec.ts
```

- [ ] **Step 4: Push the new branch to origin**

Run: `git push -u origin fix/support-chat-user-id-theme`
Expected: remote branch created and tracking set.
