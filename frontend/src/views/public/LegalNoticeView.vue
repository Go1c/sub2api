<template>
  <div class="min-h-screen bg-gray-50 text-gray-900 dark:bg-dark-950 dark:text-white">
    <header class="border-b border-gray-200 bg-white/95 dark:border-dark-800 dark:bg-dark-900/95">
      <div class="mx-auto flex max-w-5xl items-center justify-between gap-4 px-4 py-4 sm:px-6">
        <RouterLink to="/home" class="flex min-w-0 items-center gap-3">
          <span class="flex h-10 w-10 flex-shrink-0 items-center justify-center overflow-hidden rounded-xl bg-white shadow-sm ring-1 ring-gray-200 dark:bg-dark-800 dark:ring-dark-700">
            <img :src="siteLogo || '/logo.png'" :alt="siteName" class="h-full w-full object-contain" />
          </span>
          <span class="truncate text-base font-semibold text-gray-950 dark:text-white">
            {{ siteName }}
          </span>
        </RouterLink>
        <RouterLink
          to="/login"
          class="inline-flex flex-shrink-0 items-center justify-center rounded-lg bg-primary-600 px-4 py-2 text-sm font-semibold text-white shadow-sm shadow-primary-600/20 transition hover:bg-primary-700"
        >
          {{ copy.loginLabel }}
        </RouterLink>
      </div>
    </header>

    <main class="mx-auto max-w-3xl px-4 py-8 sm:px-6 lg:py-12">
      <article class="rounded-2xl border border-gray-200 bg-white p-8 shadow-sm dark:border-dark-700 dark:bg-dark-900 sm:p-10">
        <span class="inline-flex items-center rounded-full border border-primary-200 bg-primary-50 px-3 py-1 text-xs font-medium text-primary-700 dark:border-primary-500/30 dark:bg-primary-500/10 dark:text-primary-300">
          {{ copy.eyebrow }}
        </span>

        <h1 class="mt-4 text-2xl font-bold tracking-tight text-gray-950 dark:text-white sm:text-3xl">
          {{ copy.title }}
        </h1>
        <p class="mt-1.5 text-sm font-medium text-gray-500 dark:text-dark-400">
          {{ copy.titleEn }}
        </p>

        <div class="mt-8 space-y-5">
          <p class="text-[15px] leading-relaxed text-gray-700 dark:text-dark-200">
            {{ copy.bodyOne }}
          </p>
          <p class="text-sm leading-relaxed text-gray-500 dark:text-dark-400">
            {{ copy.bodyOneEn }}
          </p>
          <p class="text-[15px] leading-relaxed text-gray-700 dark:text-dark-200">
            {{ copy.bodyTwo }}
          </p>
          <p class="text-sm leading-relaxed text-gray-500 dark:text-dark-400">
            {{ copy.bodyTwoEn }}
          </p>
        </div>

        <dl class="mt-8 space-y-3 border-t border-gray-200 pt-6 text-sm dark:border-dark-700">
          <div v-for="item in infoRows" :key="item.label" class="flex flex-col gap-0.5 sm:flex-row sm:gap-3">
            <dt class="font-semibold text-gray-900 dark:text-white sm:w-56 sm:flex-shrink-0">{{ item.label }}</dt>
            <dd class="text-gray-600 dark:text-dark-300">
              <a
                v-if="item.email"
                :href="`mailto:${item.value}`"
                class="text-primary-600 underline underline-offset-4 hover:text-primary-700 dark:text-primary-300 dark:hover:text-primary-200"
              >
                {{ item.value }}
              </a>
              <span v-else>{{ item.value }}</span>
            </dd>
          </div>
        </dl>
      </article>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'

const { locale } = useI18n()
const appStore = useAppStore()

const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'LumioAPI')
const siteLogo = computed(() => appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '')

const isZh = computed(() => locale.value.startsWith('zh'))

const copy = computed(() => {
  if (isZh.value) {
    return {
      loginLabel: '登录',
      eyebrow: '公司信息与法律声明 · Company & Legal Notice',
      title: '公司信息与法律声明',
      titleEn: 'Company & Legal Notice',
      bodyOne:
        '本服务由 Lumio Games LLC 运营 —— 一家依据美国科罗拉多州法律注册成立的有限责任公司（Limited Liability Company）。我们的服务器位于美国境内，本服务受美国法律管辖并受其保护。',
      bodyOneEn:
        'This service is operated by Lumio Games LLC, a limited liability company registered under the laws of the State of Colorado, United States. Our servers are located within the United States, and the service is governed by and protected under U.S. law.',
      bodyTwo:
        '本服务面向全球，包括中国大陆。大陆用户和主体可以注册、充值并调用 API，具体以服务条款和适用法律为准。',
      bodyTwoEn:
        'This service is available worldwide, including Mainland China. Users and entities in Mainland China may register, add credit, and call the API, subject to the Terms of Service and applicable law.',
      info: {
        company: '公司 / Company',
        jurisdiction: '注册地 / Jurisdiction',
        address: '营业地址 / Address',
        servers: '服务器位置 / Servers',
        contact: '合规联系 / Contact'
      }
    }
  }
  return {
    loginLabel: 'Log in',
    eyebrow: 'Company & Legal Notice',
    title: 'Company & Legal Notice',
    titleEn: 'Company & Legal Notice',
    bodyOne:
      'This service is operated by Lumio Games LLC, a limited liability company registered under the laws of the State of Colorado, United States. Our servers are located within the United States, and the service is governed by and protected under U.S. law.',
    bodyOneEn:
      'Lumio Games LLC is registered under the laws of the State of Colorado, United States, and operates this service from within the United States under U.S. law.',
    bodyTwo:
      'This service is available worldwide, including Mainland China. Users and entities in Mainland China may register, add credit, and call the API, subject to the Terms of Service and applicable law.',
    bodyTwoEn:
      'Registration, billing, and API access are offered to users and entities in Mainland China, subject to the Terms of Service and applicable law.',
    info: {
      company: 'Company',
      jurisdiction: 'Jurisdiction',
      address: 'Address',
      servers: 'Servers',
      contact: 'Contact'
    }
  }
})

const infoRows = computed(() => [
  { label: copy.value.info.company, value: 'Lumio Games LLC' },
  { label: copy.value.info.jurisdiction, value: 'State of Colorado, United States' },
  { label: copy.value.info.address, value: '2020 N Academy Blvd, Ste 261 #4733, Colorado Springs, CO 80909, USA' },
  { label: copy.value.info.servers, value: 'United States' },
  { label: copy.value.info.contact, value: 'admin@lumio.games', email: true }
])
</script>
