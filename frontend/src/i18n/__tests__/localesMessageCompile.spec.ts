import { createRequire } from 'node:module'
import { describe, expect, it } from 'vitest'

import en from '../locales/en'
import zh from '../locales/zh'
import zhHant from '../locales/zh-Hant'
import enSettings from '../locales/en/admin/settings'
import zhSettings from '../locales/zh/admin/settings'
import enDashboard from '../locales/en/dashboard'
import zhDashboard from '../locales/zh/dashboard'

// pnpm nests @intlify/message-compiler under vue-i18n; resolve via that graph
// instead of requiring a direct dependency.
const require = createRequire(import.meta.url)
const { baseCompile } = require(
  require.resolve('@intlify/message-compiler', {
    paths: [require.resolve('vue-i18n')],
  }),
) as {
  baseCompile: (
    source: string,
    options?: { onError?: (err: { message: string }) => void },
  ) => unknown
}

// vue-i18n compiles messages at runtime: unescaped braces in copy (e.g. embedded
// JSON examples) throw "Invalid token in placeholder" and can take down the tree.
// Pre-compile every string so those failures become explicit test failures.
// Escape literal braces as {'{'} / {'}'} or keep language-neutral samples out of i18n.
function collectCompileErrors(node: unknown, path: string, out: string[]): void {
  if (typeof node === 'string') {
    baseCompile(node, {
      onError: (err) => {
        out.push(`${path}: ${err.message}`)
      },
    })
    return
  }
  if (Array.isArray(node)) {
    node.forEach((item, index) => collectCompileErrors(item, `${path}[${index}]`, out))
    return
  }
  if (node && typeof node === 'object') {
    for (const [key, value] of Object.entries(node as Record<string, unknown>)) {
      collectCompileErrors(value, path ? `${path}.${key}` : key, out)
    }
  }
}

describe('locale messages compile', () => {
  it.each([
    ['zh monothithic', zh],
    ['en monothithic', en],
    ['zh-Hant monothithic', zhHant],
    ['en modular settings', enSettings],
    ['zh modular settings', zhSettings],
    ['en modular dashboard', enDashboard],
    ['zh modular dashboard', zhDashboard],
  ] as const)('%s messages all compile without placeholder errors', (_label, messages) => {
    const errors: string[] = []
    collectCompileErrors(messages, _label, errors)
    expect(errors).toEqual([])
  })
})
