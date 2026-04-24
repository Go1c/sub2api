<template>
  <div class="min-h-screen bg-[#fafafa] text-gray-900 dark:bg-dark-950 dark:text-white">
    <header class="sticky top-0 z-20 border-b border-gray-200 bg-white/85 backdrop-blur-md dark:border-dark-800 dark:bg-dark-950/85">
      <div class="mx-auto flex h-16 max-w-5xl items-center justify-between px-6">
        <button class="flex items-center gap-3 text-left" @click="goHome">
          <span class="relative inline-flex h-10 w-10 items-center justify-center">
            <span class="absolute inset-0 rounded-xl bg-gradient-to-br from-blue-600 via-indigo-500 to-purple-600 shadow-[0_6px_18px_rgba(99,102,241,0.3)]"></span>
            <img
              v-if="siteLogo"
              :src="siteLogo"
              :alt="siteName"
              class="relative h-7 w-7 rounded-lg object-contain"
            />
            <Icon v-else name="book" size="lg" class="relative text-white" />
          </span>
          <span class="text-lg font-semibold tracking-tight">{{ siteName }}</span>
        </button>

        <button
          class="rounded-full border border-gray-200 bg-white px-4 py-2 text-sm font-semibold text-gray-700 shadow-sm transition-colors hover:border-gray-300 hover:text-gray-900 dark:border-dark-700 dark:bg-dark-900 dark:text-dark-200 dark:hover:text-white"
          @click="goConsole"
        >
          {{ consoleLabel }}
        </button>
      </div>
    </header>

    <main class="mx-auto max-w-5xl px-6 py-12">
      <section
        v-if="page"
        class="rounded-2xl border border-gray-200 bg-white p-6 shadow-sm dark:border-dark-800 dark:bg-dark-900 md:p-10"
      >
        <div class="mb-8 border-b border-gray-100 pb-6 dark:border-dark-800">
          <button
            class="mb-5 inline-flex items-center gap-2 text-sm font-medium text-gray-500 transition-colors hover:text-gray-900 dark:text-dark-400 dark:hover:text-white"
            @click="goHome"
          >
            <Icon name="arrowLeft" size="sm" />
            {{ backLabel }}
          </button>
          <h1 class="text-3xl font-semibold tracking-tight text-gray-900 dark:text-white md:text-4xl">
            {{ page.title }}
          </h1>
        </div>

        <article
          v-if="renderedContent"
          class="markdown-body"
          v-html="renderedContent"
        ></article>
        <div v-else class="rounded-xl border border-dashed border-gray-300 p-8 text-center text-sm text-gray-500 dark:border-dark-700 dark:text-dark-400">
          {{ emptyLabel }}
        </div>
      </section>

      <section
        v-else
        class="rounded-2xl border border-gray-200 bg-white p-10 text-center shadow-sm dark:border-dark-800 dark:bg-dark-900"
      >
        <Icon name="document" size="xl" class="mx-auto mb-4 text-gray-400" />
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
          {{ notFoundTitle }}
        </h1>
        <p class="mx-auto mt-3 max-w-md text-sm text-gray-500 dark:text-dark-400">
          {{ notFoundDescription }}
        </p>
        <button class="btn btn-primary mt-6" @click="goHome">
          {{ backLabel }}
        </button>
      </section>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, watchEffect } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore, useAuthStore } from '@/stores'
import type { SitePage } from '@/types'

marked.setOptions({
  breaks: true,
  gfm: true
})

const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
const authStore = useAuthStore()
const { locale } = useI18n()

const isZh = computed(() => locale.value.startsWith('zh'))
const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API')
const siteLogo = computed(() => appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '')
const sitePages = computed(() => appStore.cachedPublicSettings?.site_pages || [])

const consoleLabel = computed(() => (isZh.value ? '控制台' : 'Console'))
const backLabel = computed(() => (isZh.value ? '返回首页' : 'Back home'))
const emptyLabel = computed(() => (isZh.value ? '内容尚未配置' : 'Content has not been configured yet'))
const notFoundTitle = computed(() => (isZh.value ? '页面未配置' : 'Page not configured'))
const notFoundDescription = computed(() =>
  isZh.value
    ? '该页面可能未启用，或后台配置的访问路径与当前地址不一致。'
    : 'This page may be disabled, or its configured path does not match the current URL.'
)

const requestedSlug = computed(() => {
  const raw = route.params.slug
  const suffix = Array.isArray(raw) ? raw.join('/') : String(raw || '')
  return normalizeSlug(`doc/${suffix}`)
})

const page = computed<SitePage | null>(() => {
  return sitePages.value.find((item) => {
    return item.enabled !== false && normalizeSlug(item.slug) === requestedSlug.value
  }) || null
})

const renderedContent = computed(() => {
  const content = page.value?.content?.trim() || ''
  if (!content) return ''
  return DOMPurify.sanitize(marked.parse(content) as string)
})

function normalizeSlug(slug: string) {
  return slug.trim().replace(/^\/+|\/+$/g, '')
}

function goHome() {
  router.push('/home')
}

function goConsole() {
  router.push(authStore.isAuthenticated ? (authStore.isAdmin ? '/admin/dashboard' : '/dashboard') : '/login')
}

onMounted(() => {
  authStore.checkAuth()
  if (!appStore.publicSettingsLoaded) {
    void appStore.fetchPublicSettings()
  }
})

watchEffect(() => {
  if (page.value?.title) {
    document.title = `${page.value.title} - ${siteName.value}`
  }
})
</script>

<style scoped>
.markdown-body {
  color: rgb(55 65 81);
  font-size: 0.975rem;
  line-height: 1.8;
}

.dark .markdown-body {
  color: rgb(209 213 219);
}

.markdown-body :deep(h1),
.markdown-body :deep(h2),
.markdown-body :deep(h3) {
  margin: 1.5em 0 0.65em;
  color: rgb(17 24 39);
  font-weight: 700;
  line-height: 1.25;
}

.dark .markdown-body :deep(h1),
.dark .markdown-body :deep(h2),
.dark .markdown-body :deep(h3) {
  color: white;
}

.markdown-body :deep(h1) {
  font-size: 1.875rem;
}

.markdown-body :deep(h2) {
  font-size: 1.5rem;
}

.markdown-body :deep(h3) {
  font-size: 1.25rem;
}

.markdown-body :deep(p),
.markdown-body :deep(ul),
.markdown-body :deep(ol),
.markdown-body :deep(blockquote),
.markdown-body :deep(pre),
.markdown-body :deep(table) {
  margin: 1em 0;
}

.markdown-body :deep(a) {
  color: rgb(37 99 235);
  font-weight: 600;
  text-decoration: none;
}

.markdown-body :deep(a:hover) {
  text-decoration: underline;
}

.markdown-body :deep(ul),
.markdown-body :deep(ol) {
  padding-left: 1.5rem;
}

.markdown-body :deep(ul) {
  list-style: disc;
}

.markdown-body :deep(ol) {
  list-style: decimal;
}

.markdown-body :deep(blockquote) {
  border-left: 4px solid rgb(191 219 254);
  padding-left: 1rem;
  color: rgb(75 85 99);
}

.dark .markdown-body :deep(blockquote) {
  border-left-color: rgb(30 64 175);
  color: rgb(156 163 175);
}

.markdown-body :deep(code) {
  border-radius: 0.375rem;
  background: rgb(243 244 246);
  padding: 0.125rem 0.375rem;
  font-size: 0.875em;
}

.dark .markdown-body :deep(code) {
  background: rgb(31 41 55);
}

.markdown-body :deep(pre) {
  overflow-x: auto;
  border-radius: 0.75rem;
  background: rgb(17 24 39);
  padding: 1rem;
  color: rgb(243 244 246);
}

.markdown-body :deep(pre code) {
  background: transparent;
  padding: 0;
  color: inherit;
}

.markdown-body :deep(table) {
  width: 100%;
  border-collapse: collapse;
}

.markdown-body :deep(th),
.markdown-body :deep(td) {
  border: 1px solid rgb(229 231 235);
  padding: 0.625rem 0.75rem;
}

.dark .markdown-body :deep(th),
.dark .markdown-body :deep(td) {
  border-color: rgb(55 65 81);
}

.markdown-body :deep(img) {
  max-width: 100%;
  border-radius: 0.75rem;
}
</style>
