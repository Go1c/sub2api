/// <reference types="vite/client" />

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<{}, {}, any>
  export default component
}

interface ImportMetaEnv {
  readonly VITE_API_BASE_URL?: string
  readonly VITE_DEV_PORT?: string
  readonly VITE_PREVIEW_PORT?: string
  readonly VITE_USE_MOCK?: string
  readonly VITE_SITE_NAME?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
