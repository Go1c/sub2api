<template>
  <div v-if="homeContent" class="min-h-screen">
    <iframe
      v-if="isHomeContentUrl"
      :src="homeContent.trim()"
      class="h-screen w-full border-0"
      allowfullscreen
    ></iframe>
    <div v-else v-html="homeContent"></div>
  </div>

  <div
    v-else
    class="flex min-h-screen flex-col bg-[#fafafa] text-gray-900 selection:bg-blue-200 selection:text-blue-900 dark:bg-dark-950 dark:text-white"
  >
    <div class="relative order-1 overflow-hidden">
      <nav class="relative z-20 mx-auto flex max-w-7xl items-center justify-between px-6 py-5">
        <button class="group flex items-center gap-3 text-left" @click="scrollTo('body')">
          <span class="relative inline-flex h-11 w-11 items-center justify-center">
            <span class="absolute inset-0 rounded-2xl bg-gradient-to-br from-blue-600 via-indigo-500 to-purple-600 shadow-[0_8px_24px_rgba(99,102,241,0.35)] transition-shadow group-hover:shadow-[0_10px_30px_rgba(99,102,241,0.5)]"></span>
            <span class="absolute inset-0 rounded-2xl bg-gradient-to-b from-white/30 to-transparent"></span>
            <img
              v-if="siteLogo"
              :src="siteLogo"
              :alt="siteName"
              class="relative h-7 w-7 rounded-lg object-contain"
            />
            <svg
              v-else
              class="relative"
              width="26"
              height="26"
              viewBox="0 0 32 32"
              fill="none"
              xmlns="http://www.w3.org/2000/svg"
            >
              <path d="M16 4L27 10V22L16 28L5 22V10L16 4Z" stroke="white" stroke-width="1.6" stroke-linejoin="round" stroke-opacity="0.55"></path>
              <path d="M16 10L22 13.5V20.5L16 24L10 20.5V13.5L16 10Z" fill="white" fill-opacity="0.95"></path>
              <circle cx="16" cy="17" r="2" fill="url(#hero-logo-core)"></circle>
              <defs>
                <linearGradient id="hero-logo-core" x1="14" y1="15" x2="18" y2="19" gradientUnits="userSpaceOnUse">
                  <stop stop-color="#2563EB"></stop>
                  <stop offset="1" stop-color="#9333EA"></stop>
                </linearGradient>
              </defs>
            </svg>
          </span>
          <span class="text-2xl font-semibold tracking-tight text-gray-900 dark:text-white">
            {{ siteName }}
          </span>
        </button>

        <div
          class="hidden items-center gap-1 rounded-full border border-gray-200 bg-white/70 px-2 py-1.5 shadow-[0_4px_20px_rgba(99,102,241,0.06)] backdrop-blur-md dark:border-dark-700 dark:bg-dark-900/70 md:flex"
        >
          <button
            v-for="item in navItems"
            :key="item.key"
            class="rounded-full px-3.5 py-2 text-[14px] font-medium text-gray-800 transition-all hover:bg-blue-50/70 hover:text-blue-700 dark:text-dark-200 dark:hover:bg-dark-800 dark:hover:text-white"
            :class="item.dim ? 'text-gray-600 dark:text-dark-400' : ''"
            @click="onNav(item)"
          >
            {{ item.label }}
          </button>
        </div>

        <div class="flex items-center gap-2">
          <button
            class="rounded-lg p-2 text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-700 dark:text-dark-400 dark:hover:bg-dark-800 dark:hover:text-white"
            :title="t('home.viewDocs')"
            @click="openDocs"
          >
            <Icon name="book" size="md" />
          </button>

          <LocaleSwitcher />

          <button
            class="rounded-lg p-2 text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-700 dark:text-dark-400 dark:hover:bg-dark-800 dark:hover:text-white"
            :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
            @click="toggleTheme"
          >
            <Icon v-if="isDark" name="sun" size="md" />
            <Icon v-else name="moon" size="md" />
          </button>

          <button
            class="rounded-full border border-white/20 bg-gradient-to-r from-blue-600 to-purple-600 px-5 py-2.5 text-[15px] font-semibold text-white shadow-[0_4px_14px_rgba(99,102,241,0.3)] transition-all hover:-translate-y-0.5 hover:from-blue-700 hover:to-purple-700 hover:shadow-[0_6px_20px_rgba(99,102,241,0.4)]"
            @click="goConsole"
          >
            {{ copy.headerCta }}
          </button>
        </div>
      </nav>

      <div class="pointer-events-none absolute inset-0 -z-10 hero-grid"></div>
      <div class="pointer-events-none absolute inset-0 -z-10 overflow-hidden">
        <div class="animate-blob absolute -top-[20%] left-[20%] h-[40rem] w-[40rem] rounded-full bg-blue-100 opacity-50 mix-blend-multiply blur-3xl dark:bg-blue-900/30"></div>
        <div class="animate-blob animation-delay-2000 absolute -top-[20%] right-[20%] h-[40rem] w-[40rem] rounded-full bg-purple-100 opacity-50 mix-blend-multiply blur-3xl dark:bg-purple-900/30"></div>
      </div>

      <div class="relative z-10 mx-auto max-w-6xl px-6 pb-32 pt-20 text-center">
        <div class="pointer-events-none absolute inset-0 -z-[1] flex items-center justify-center" aria-hidden="true">
          <div class="relative orbital-wrap">
            <div class="absolute -inset-32 bg-[radial-gradient(circle_at_center,rgba(99,102,241,0.18),rgba(168,85,247,0.10)_40%,transparent_70%)] blur-2xl"></div>
            <div class="orbital-shell h-[560px] w-[560px] rounded-full border border-blue-200/60 dark:border-blue-900/30"></div>
            <div class="orbital-ring absolute inset-10 rounded-full border border-purple-200/50 dark:border-purple-900/30"></div>
            <div class="absolute inset-24 rounded-full border border-gray-200/70 dark:border-dark-700/70"></div>
            <div class="orbital-nodes absolute inset-0">
              <span class="absolute left-1/2 top-0 h-2.5 w-2.5 -translate-x-1/2 rounded-full bg-gradient-to-br from-blue-500 to-purple-500 shadow-[0_0_12px_rgba(99,102,241,0.7)]"></span>
              <span class="absolute bottom-0 left-1/2 h-1.5 w-1.5 -translate-x-1/2 rounded-full bg-purple-400 shadow-[0_0_8px_rgba(168,85,247,0.6)]"></span>
            </div>
          </div>
        </div>

        <div
          v-for="(item, index) in copy.floatingItems"
          :key="item.text"
          :class="[
            'float-pill absolute hidden items-center gap-1.5 rounded-full border border-white bg-white/70 px-3 py-1.5 text-xs font-medium text-gray-700 shadow-[0_4px_20px_rgba(99,102,241,0.08)] backdrop-blur-md dark:border-dark-700 dark:bg-dark-900/70 dark:text-dark-200 lg:flex',
            floatingSlots[index] || floatingSlots[0]
          ]"
          :style="{ animationDelay: `${index * 600}ms` }"
        >
          <span class="inline-flex h-5 w-5 items-center justify-center rounded-full border border-white bg-gradient-to-br from-blue-50 to-purple-50 shadow-sm dark:border-dark-700 dark:from-dark-800 dark:to-dark-700">
            <Icon :name="item.icon" size="sm" :class="floatingIconTones[index] || floatingIconTones[0]" />
          </span>
          {{ item.text }}
        </div>

        <div
          class="fade-rise mb-8 inline-flex items-center gap-2 rounded-full border border-gray-200 bg-white/80 px-4 py-1.5 text-sm font-medium text-gray-800 shadow-sm backdrop-blur-md dark:border-dark-700 dark:bg-dark-900/80 dark:text-dark-100"
          style="--delay: 40ms;"
        >
          <span class="relative flex h-2 w-2">
            <span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-green-400 opacity-75"></span>
            <span class="relative inline-flex h-2 w-2 rounded-full bg-green-500"></span>
          </span>
          {{ copy.announcement }}
          <span class="ml-1 rounded-full bg-gradient-to-r from-blue-600 to-purple-600 px-1.5 py-0.5 text-[10px] font-semibold text-white">
            {{ copy.announcementBadge }}
          </span>
        </div>

        <h1
          class="fade-rise mb-6 text-6xl font-semibold leading-[0.95] tracking-tight text-gray-900 dark:text-white md:text-[7.5rem]"
          style="--delay: 120ms;"
        >
          <span class="block">{{ copy.titleLead }}</span>
          <span class="block bg-gradient-to-r from-blue-600 via-indigo-500 to-purple-600 bg-clip-text text-transparent drop-shadow-[0_4px_30px_rgba(99,102,241,0.25)]">
            {{ copy.titleAccent }}
          </span>
        </h1>

        <p
          class="fade-rise mx-auto mb-4 max-w-3xl text-lg leading-relaxed text-gray-600 dark:text-dark-300 md:text-xl"
          style="--delay: 200ms;"
        >
          {{ copy.subtitle }}
        </p>

        <p
          class="fade-rise mx-auto mb-12 max-w-2xl text-base text-gray-500 dark:text-dark-400 md:text-lg"
          style="--delay: 280ms;"
        >
          {{ copy.description }}
        </p>

        <div class="fade-rise flex flex-col items-center justify-center gap-4 sm:flex-row" style="--delay: 360ms;">
          <button
            class="group relative inline-flex w-full items-center justify-center gap-2 px-8 py-4 transition-all duration-300 hover:scale-[1.02] sm:w-auto"
            @click="goPrimary"
          >
            <div class="absolute -inset-0.5 rounded-full bg-gradient-to-r from-cyan-400 via-blue-500 to-purple-600 opacity-50 blur transition duration-500 group-hover:opacity-100 group-hover:duration-200"></div>
            <div class="absolute inset-0 rounded-full border border-white/20 bg-gradient-to-r from-blue-600 to-purple-600 shadow-[inset_0_1px_1px_rgba(255,255,255,0.4)]"></div>
            <span class="relative flex items-center gap-2 text-lg font-medium text-white">
              {{ copy.primaryCta }}
              <Icon name="arrowRight" size="md" class="transition-transform duration-300 group-hover:translate-x-1" />
            </span>
          </button>

          <button
            class="flex w-full items-center justify-center gap-2 rounded-full border border-gray-200 bg-white/80 px-8 py-4 text-lg font-medium text-gray-700 shadow-sm backdrop-blur-md transition-all hover:border-gray-300 hover:bg-white dark:border-dark-700 dark:bg-dark-900/80 dark:text-dark-100 dark:hover:bg-dark-900 sm:w-auto"
            @click="openDocs"
          >
            <Icon name="terminal" size="md" />
            {{ copy.secondaryCta }}
          </button>
        </div>

        <div class="fade-rise mt-20 flex flex-wrap items-center justify-center gap-3" style="--delay: 440ms;">
          <div
            v-for="(item, index) in copy.heroBadges"
            :key="item.text"
            class="group flex items-center gap-2 rounded-full border border-white bg-white/70 px-5 py-2.5 text-sm font-medium text-gray-700 shadow-[0_4px_20px_rgba(99,102,241,0.06)] backdrop-blur-md transition-all hover:-translate-y-0.5 hover:shadow-[0_8px_30px_rgba(99,102,241,0.15)] dark:border-dark-700 dark:bg-dark-900/70 dark:text-dark-200"
          >
            <span class="inline-flex h-6 w-6 items-center justify-center rounded-full border border-white bg-gradient-to-br from-blue-50 to-purple-50 transition-transform group-hover:scale-110 dark:border-dark-700 dark:from-dark-800 dark:to-dark-700">
              <Icon :name="item.icon" size="sm" :class="badgeIconTones[index] || badgeIconTones[0]" />
            </span>
            {{ item.text }}
          </div>
        </div>
      </div>
    </div>

    <section id="features" class="relative order-3 mx-auto max-w-7xl px-6 py-16">
      <div class="fade-rise mb-16 text-center">
        <div class="inline-flex items-center gap-2 rounded-full border border-blue-100 bg-blue-50 px-3 py-1 text-xs font-medium text-blue-700 dark:border-blue-900/40 dark:bg-blue-950/40 dark:text-blue-300">
          <span class="h-1.5 w-1.5 animate-pulse rounded-full bg-blue-500"></span>
          {{ copy.featuresKicker }}
        </div>
        <h2 class="mt-5 text-5xl font-semibold tracking-tight text-gray-900 dark:text-white md:text-6xl">
          {{ copy.featuresTitleLead }}
          <span class="bg-gradient-to-r from-blue-600 via-indigo-500 to-purple-600 bg-clip-text text-transparent">
            {{ copy.featuresTitleAccent }}
          </span>
        </h2>
        <p class="mx-auto mt-5 max-w-2xl text-lg text-gray-600 dark:text-dark-300">
          {{ copy.featuresSubtitle }}
        </p>
      </div>

      <div class="mb-8 grid gap-5 md:grid-cols-2 lg:grid-cols-4">
        <div
          v-for="(card, index) in copy.featureCards"
          :key="card.title"
          class="fade-rise group rounded-3xl border border-gray-200 bg-white/80 p-7 backdrop-blur-md transition-all duration-300 hover:-translate-y-1 hover:border-gray-300 hover:shadow-[0_10px_30px_rgba(99,102,241,0.08)] dark:border-dark-700 dark:bg-dark-900/70 dark:hover:border-dark-600"
          :style="{ '--delay': `${120 + index * 80}ms` }"
        >
          <div class="mb-5 inline-flex h-11 w-11 items-center justify-center rounded-2xl border border-white bg-gradient-to-br from-blue-50 to-purple-50 shadow-sm transition-transform group-hover:scale-110 dark:border-dark-700 dark:from-dark-800 dark:to-dark-700">
            <Icon :name="card.icon" size="lg" class="text-blue-600 dark:text-blue-300" />
          </div>
          <h3 class="mb-2 text-xl font-semibold tracking-tight text-gray-900 dark:text-white">{{ card.title }}</h3>
          <p class="leading-relaxed text-gray-600 dark:text-dark-400">{{ card.desc }}</p>
        </div>
      </div>

      <div class="grid gap-5 lg:grid-cols-5">
        <div class="fade-rise rounded-3xl border border-gray-200 bg-white/80 p-7 backdrop-blur-md dark:border-dark-700 dark:bg-dark-900/70 lg:col-span-3" style="--delay: 200ms;">
          <div class="mb-6 flex items-center justify-between">
            <h3 class="text-lg font-semibold tracking-tight text-gray-900 dark:text-white">{{ copy.enginesTitle }}</h3>
            <span class="text-xs text-gray-500 dark:text-dark-400">{{ copy.enginesNote }}</span>
          </div>

          <div class="grid gap-4 sm:grid-cols-3">
            <div
              v-for="card in copy.engineCards"
              :key="card.name"
              class="group relative rounded-2xl border border-gray-100 bg-gray-50/60 p-5 transition-all hover:border-gray-200 hover:bg-white hover:shadow-sm dark:border-dark-800 dark:bg-dark-950/60 dark:hover:border-dark-700 dark:hover:bg-dark-900/70"
            >
              <div class="mb-4 flex items-center justify-between">
                <div class="flex h-9 w-9 items-center justify-center rounded-xl border border-gray-100 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900">
                  <Icon :name="card.icon" size="lg" :class="engineTone(card.tone).icon" />
                </div>
                <span :class="['rounded-full bg-gradient-to-r px-2 py-0.5 text-[10px] font-bold text-white', engineTone(card.tone).pill]">
                  {{ card.pill }}
                </span>
              </div>
              <div class="mb-1 text-xl font-semibold tracking-tight text-gray-900 dark:text-white">{{ card.name }}</div>
              <div class="text-xs leading-relaxed text-gray-500 dark:text-dark-400">{{ card.models }}</div>
            </div>
          </div>
        </div>

        <div id="status" class="fade-rise relative overflow-hidden rounded-3xl border border-gray-200 bg-gradient-to-br from-blue-50 via-white to-purple-50 p-7 dark:border-dark-700 dark:from-dark-900 dark:via-dark-900 dark:to-purple-950/30 lg:col-span-2" style="--delay: 260ms;">
          <div class="pointer-events-none absolute -right-16 -top-16 h-48 w-48 rounded-full bg-blue-200/40 blur-3xl dark:bg-blue-900/20"></div>
          <div class="relative">
            <div class="mb-5 inline-flex items-center gap-2 rounded-full border border-emerald-200 bg-white px-3 py-1 text-xs font-medium text-emerald-700 dark:border-emerald-900/40 dark:bg-dark-900 dark:text-emerald-300">
              <span class="h-1.5 w-1.5 animate-pulse rounded-full bg-emerald-500"></span>
              {{ copy.statusEyebrow }}
            </div>

            <div class="space-y-3">
              <div
                v-for="(item, index) in copy.statusItems"
                :key="item.label"
                class="flex items-center justify-between border-b border-gray-100 py-2 last:border-0 dark:border-dark-800"
              >
                <div class="flex items-center gap-2.5">
                  <span class="inline-flex h-7 w-7 items-center justify-center rounded-lg border border-gray-100 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900">
                    <Icon :name="item.icon" size="sm" :class="statusIconTones[index] || statusIconTones[0]" />
                  </span>
                  <span class="text-sm text-gray-600 dark:text-dark-300">{{ item.label }}</span>
                </div>
                <span class="text-2xl font-bold tracking-tight text-gray-900 dark:text-white">{{ item.value }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>

    <section id="pricing" class="relative order-2 mx-auto max-w-7xl px-6 py-12">
      <div class="pointer-events-none absolute inset-x-0 top-0 -z-10 h-[40rem] bg-[radial-gradient(ellipse_50%_40%_at_50%_0%,rgba(99,102,241,0.10),transparent_70%)]"></div>

      <h2 class="text-center text-5xl font-semibold tracking-tight text-gray-900 dark:text-white">
        {{ copy.pricingTitleLead }}
        <span class="bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent">
          {{ copy.pricingTitleAccent }}
        </span>
      </h2>

      <p class="mt-4 text-center text-gray-600 dark:text-dark-300">{{ pricingNote }}</p>

      <div class="mx-auto mb-12 mt-5 flex max-w-3xl flex-wrap items-center justify-center gap-2 text-sm">
        <span class="rounded-full border border-emerald-200 bg-emerald-50 px-3 py-1.5 font-semibold text-emerald-700 dark:border-emerald-400/30 dark:bg-emerald-500/10 dark:text-emerald-200">
          {{ copy.pricingRechargeNote }}
        </span>
        <span class="rounded-full border border-purple-200 bg-purple-50 px-3 py-1.5 font-semibold text-purple-700 dark:border-purple-400/30 dark:bg-purple-500/10 dark:text-purple-200">
          {{ copy.pricingDiscountFormula }}
        </span>
      </div>

      <div class="overflow-x-auto">
        <div class="inline-block min-w-full align-middle">
          <div class="overflow-hidden rounded-2xl border border-gray-200 shadow-sm dark:border-dark-700">
            <table class="min-w-full divide-y divide-gray-200 dark:divide-dark-700">
              <thead class="bg-gray-50 dark:bg-dark-900">
                <tr>
                  <th class="px-6 py-4 text-left text-sm font-semibold text-gray-900 dark:text-white">{{ copy.pricingCols.model }}</th>
                  <th class="px-6 py-4 text-left text-sm font-semibold text-gray-900 dark:text-white">{{ copy.pricingCols.group }}</th>
                  <th class="px-6 py-4 text-center text-sm font-semibold text-gray-900 dark:text-white">
                    <div class="flex flex-col items-center gap-1.5">
                      <span>{{ copy.pricingCols.officialInput }}</span>
                      <span class="rounded-full border border-amber-200 bg-amber-50 px-2 py-0.5 text-[11px] font-semibold text-amber-700 dark:border-amber-400/30 dark:bg-amber-500/10 dark:text-amber-200">
                        {{ copy.pricingCurrency.usd }}
                      </span>
                    </div>
                  </th>
                  <th class="px-6 py-4 text-center text-sm font-semibold text-gray-900 dark:text-white">
                    <div class="flex flex-col items-center gap-1.5">
                      <span>{{ copy.pricingCols.officialOutput }}</span>
                      <span class="rounded-full border border-amber-200 bg-amber-50 px-2 py-0.5 text-[11px] font-semibold text-amber-700 dark:border-amber-400/30 dark:bg-amber-500/10 dark:text-amber-200">
                        {{ copy.pricingCurrency.usd }}
                      </span>
                    </div>
                  </th>
                  <th class="px-6 py-4 text-center text-sm font-semibold text-gray-900 dark:text-white">
                    <div class="flex flex-col items-center gap-1.5">
                      <span>{{ copy.pricingCols.convertedInput }}</span>
                      <span class="rounded-full border border-emerald-200 bg-emerald-50 px-2 py-0.5 text-[11px] font-semibold text-emerald-700 dark:border-emerald-400/30 dark:bg-emerald-500/10 dark:text-emerald-200">
                        {{ copy.pricingCurrency.cny }}
                      </span>
                    </div>
                  </th>
                  <th class="px-6 py-4 text-center text-sm font-semibold text-gray-900 dark:text-white">
                    <div class="flex flex-col items-center gap-1.5">
                      <span>{{ copy.pricingCols.convertedOutput }}</span>
                      <span class="rounded-full border border-emerald-200 bg-emerald-50 px-2 py-0.5 text-[11px] font-semibold text-emerald-700 dark:border-emerald-400/30 dark:bg-emerald-500/10 dark:text-emerald-200">
                        {{ copy.pricingCurrency.cny }}
                      </span>
                    </div>
                  </th>
                  <th class="px-6 py-4 text-center text-sm font-semibold text-blue-700 dark:text-blue-300">
                    <div class="flex flex-col items-center gap-1.5">
                      <span>{{ copy.pricingCols.input }}</span>
                      <span class="rounded-full border border-emerald-200 bg-emerald-50 px-2 py-0.5 text-[11px] font-semibold text-emerald-700 dark:border-emerald-400/30 dark:bg-emerald-500/10 dark:text-emerald-200">
                        {{ copy.pricingCurrency.cny }}
                      </span>
                    </div>
                  </th>
                  <th class="px-6 py-4 text-center text-sm font-semibold text-blue-700 dark:text-blue-300">
                    <div class="flex flex-col items-center gap-1.5">
                      <span>{{ copy.pricingCols.output }}</span>
                      <span class="rounded-full border border-emerald-200 bg-emerald-50 px-2 py-0.5 text-[11px] font-semibold text-emerald-700 dark:border-emerald-400/30 dark:bg-emerald-500/10 dark:text-emerald-200">
                        {{ copy.pricingCurrency.cny }}
                      </span>
                    </div>
                  </th>
                  <th class="px-6 py-4 text-center text-sm font-semibold text-gray-900 dark:text-white">
                    <div class="flex flex-col items-center gap-1.5">
                      <span>{{ copy.pricingCols.discount }}</span>
                      <span class="rounded-full border border-purple-200 bg-purple-50 px-2 py-0.5 text-[11px] font-semibold text-purple-700 dark:border-purple-400/30 dark:bg-purple-500/10 dark:text-purple-200">
                        {{ copy.pricingDiscountBasis }}
                      </span>
                    </div>
                  </th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-200 bg-white dark:divide-dark-700 dark:bg-dark-900">
                <tr
                  v-for="row in pricingRows"
                  :key="row.model"
                  :class="['transition-all duration-200', pricingTone(row).row]"
                >
                  <td class="px-6 py-4 text-sm font-medium text-gray-900 dark:text-white">
                    <div class="flex items-center gap-2">{{ row.model }}</div>
                  </td>
                  <td class="px-6 py-4 text-sm text-gray-600 dark:text-dark-300">{{ row.group }}</td>
                  <td class="px-6 py-4 text-center text-sm">
                    <span class="inline-flex min-w-[4.75rem] items-center justify-center rounded-md border border-amber-200/70 bg-amber-50/80 px-2.5 py-1 font-semibold text-amber-700 shadow-sm dark:border-amber-400/20 dark:bg-amber-500/10 dark:text-amber-200">
                      {{ formatUsd(row.officialInput) }}
                    </span>
                  </td>
                  <td class="px-6 py-4 text-center text-sm">
                    <span class="inline-flex min-w-[4.75rem] items-center justify-center rounded-md border border-amber-200/70 bg-amber-50/80 px-2.5 py-1 font-semibold text-amber-700 shadow-sm dark:border-amber-400/20 dark:bg-amber-500/10 dark:text-amber-200">
                      {{ formatUsd(row.officialOutput) }}
                    </span>
                  </td>
                  <td class="px-6 py-4 text-center text-sm">
                    <span class="inline-flex min-w-[4.75rem] items-center justify-center rounded-md border border-emerald-200/50 bg-emerald-50/50 px-2.5 py-1 font-semibold text-emerald-700 line-through decoration-2 decoration-red-400/70 shadow-sm dark:border-emerald-400/15 dark:bg-emerald-500/10 dark:text-emerald-300/70">
                      {{ formatConverted(row.officialInput) }}
                    </span>
                  </td>
                  <td class="px-6 py-4 text-center text-sm">
                    <span class="inline-flex min-w-[4.75rem] items-center justify-center rounded-md border border-emerald-200/50 bg-emerald-50/50 px-2.5 py-1 font-semibold text-emerald-700 line-through decoration-2 decoration-red-400/70 shadow-sm dark:border-emerald-400/15 dark:bg-emerald-500/10 dark:text-emerald-300/70">
                      {{ formatConverted(row.officialOutput) }}
                    </span>
                  </td>
                  <td class="px-6 py-4 text-center">
                    <div :class="['inline-flex min-w-[6.25rem] items-baseline justify-center gap-1 rounded-lg border px-3 py-1.5 font-bold shadow-sm', pricingTone(row).price]">
                      <span class="text-[10px] font-semibold opacity-70">CNY</span>
                      <span class="text-base">{{ formatCny(row.inputPrice) }}</span>
                    </div>
                  </td>
                  <td class="px-6 py-4 text-center">
                    <div :class="['inline-flex min-w-[6.25rem] items-baseline justify-center gap-1 rounded-lg border px-3 py-1.5 font-bold shadow-sm', pricingTone(row).price]">
                      <span class="text-[10px] font-semibold opacity-70">CNY</span>
                      <span class="text-base">{{ formatCny(row.outputPrice) }}</span>
                    </div>
                  </td>
                  <td class="px-6 py-4 text-center">
                    <div class="flex items-center justify-center gap-2">
                      <span v-if="pricingTone(row).flame" class="animate-bounce text-xl drop-shadow-md">🔥</span>
                      <span
                        :class="[
                          'inline-flex min-w-[7rem] flex-col items-center justify-center rounded-full bg-gradient-to-r px-5 py-2 text-white shadow-xl ring-2 ring-white/25 dark:ring-white/10',
                          pricingTone(row).discount,
                          pricingTone(row).discountText
                        ]"
                      >
                        <span class="text-[10px] font-semibold leading-none text-white/80">{{ copy.pricingDiscountTag }}</span>
                        <span class="mt-0.5 text-xl font-extrabold leading-none">{{ row.discount }}</span>
                      </span>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </section>

    <section id="vision" class="relative order-4 mx-auto max-w-7xl px-6 py-16">
      <div class="pointer-events-none absolute inset-0 -z-10 overflow-hidden">
        <div class="absolute left-1/2 top-1/2 h-[70rem] w-[70rem] -translate-x-1/2 -translate-y-1/2 bg-[radial-gradient(circle_at_center,rgba(99,102,241,0.12),rgba(168,85,247,0.06)_40%,transparent_70%)]"></div>
        <div class="absolute inset-0 bg-[linear-gradient(to_right,#f0f0f0_1px,transparent_1px),linear-gradient(to_bottom,#f0f0f0_1px,transparent_1px)] bg-[size:4rem_4rem] opacity-60 [mask-image:radial-gradient(ellipse_50%_40%_at_50%_50%,#000_40%,transparent_80%)] dark:bg-none"></div>
      </div>

      <div class="fade-rise relative overflow-hidden rounded-[2rem] border border-gray-200 bg-white/70 p-12 text-center shadow-[0_12px_40px_rgba(99,102,241,0.08)] backdrop-blur-md dark:border-dark-700 dark:bg-dark-900/70 md:p-16">
        <div class="absolute inset-0 -z-10 bg-[radial-gradient(ellipse_60%_50%_at_50%_0%,rgba(99,102,241,0.10),transparent_70%)]"></div>

        <p class="mx-auto max-w-4xl text-2xl font-semibold leading-snug tracking-tight text-gray-900 dark:text-white md:text-4xl">
          "<span>{{ copy.visionLead }}</span><span class="bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent">{{ copy.visionHighlightOne }}</span><span>{{ copy.visionMiddle }}</span><span class="bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent">{{ copy.visionHighlightTwo }}</span><span>{{ copy.visionEnd }}</span>"
        </p>

        <p class="mt-6 text-sm text-gray-500 dark:text-dark-400">— {{ copy.visionAuthor }}</p>

        <button class="group relative mt-10 inline-flex items-center gap-2 px-8 py-3.5 transition-all duration-300 hover:scale-[1.02]" @click="goPrimary">
          <div class="absolute -inset-0.5 rounded-full bg-gradient-to-r from-cyan-400 via-blue-500 to-purple-600 opacity-50 blur transition duration-500 group-hover:opacity-100"></div>
          <div class="absolute inset-0 rounded-full border border-white/20 bg-gradient-to-r from-blue-600 to-purple-600 shadow-[inset_0_1px_1px_rgba(255,255,255,0.4)]"></div>
          <span class="relative flex items-center gap-2 font-medium text-white">
            {{ copy.visionCta }}
            <Icon name="arrowRight" size="sm" class="transition-transform duration-300 group-hover:translate-x-1" />
          </span>
        </button>
      </div>
    </section>

    <section id="support" class="order-5 border-t border-gray-200 bg-white px-6 py-14 dark:border-dark-800 dark:bg-dark-950">
      <div class="mx-auto grid max-w-7xl gap-8 lg:grid-cols-[0.9fr_1.1fr] lg:items-center">
        <div class="min-w-0">
          <div class="mb-4 inline-flex h-11 w-11 items-center justify-center rounded-xl border border-blue-100 bg-blue-50 text-blue-600 dark:border-blue-900/40 dark:bg-blue-950/40 dark:text-blue-300">
            <Icon name="chatBubble" size="lg" />
          </div>
          <h2 class="text-3xl font-semibold tracking-tight text-gray-900 dark:text-white md:text-4xl">
            {{ copy.supportTitle }}
          </h2>
          <p class="mt-4 max-w-xl text-base leading-relaxed text-gray-600 dark:text-dark-300">
            {{ copy.supportSubtitle }}
          </p>
        </div>

        <div v-if="contactChannels.length > 0" class="grid min-w-0 gap-3 sm:grid-cols-2">
          <button
            v-for="channel in contactChannels"
            :key="`${channel.label}-${channel.url}`"
            class="group flex min-w-0 items-center justify-between gap-3 rounded-xl border border-gray-200 bg-gray-50 px-5 py-4 text-left transition-all hover:-translate-y-0.5 hover:border-blue-200 hover:bg-blue-50 dark:border-dark-700 dark:bg-dark-900 dark:hover:border-blue-900/60 dark:hover:bg-blue-950/30"
            @click="openContact(channel.url)"
          >
            <span class="min-w-0">
              <span class="block text-sm font-semibold text-gray-900 dark:text-white">{{ channel.label }}</span>
              <span class="mt-1 block break-all text-xs leading-relaxed text-gray-500 dark:text-dark-400">{{ channel.url }}</span>
            </span>
            <Icon name="externalLink" size="sm" class="shrink-0 text-gray-400 transition-colors group-hover:text-blue-600 dark:group-hover:text-blue-300" />
          </button>
        </div>

        <div
          v-else-if="fallbackContactInfo"
          class="min-w-0 rounded-xl border border-gray-200 bg-gray-50 px-5 py-4 text-sm font-medium text-gray-700 dark:border-dark-700 dark:bg-dark-900 dark:text-dark-200"
        >
          <span class="block text-xs font-semibold uppercase text-gray-500 dark:text-dark-400">{{ copy.supportFallbackLabel }}</span>
          <span class="mt-2 block whitespace-pre-wrap break-words leading-relaxed [overflow-wrap:anywhere]">
            {{ fallbackContactInfo }}
          </span>
        </div>
      </div>
    </section>

    <footer id="footer" class="order-6 border-t border-gray-200 bg-white px-6 py-8 dark:border-dark-800 dark:bg-dark-950">
      <div class="mx-auto flex max-w-7xl flex-col items-center justify-between gap-4 md:flex-row">
        <div class="flex items-center gap-3">
          <span class="relative inline-flex h-9 w-9 items-center justify-center">
            <span class="absolute inset-0 rounded-xl bg-gradient-to-br from-blue-600 via-indigo-500 to-purple-600 shadow-[0_6px_18px_rgba(99,102,241,0.3)]"></span>
            <span class="absolute inset-0 rounded-xl bg-gradient-to-b from-white/30 to-transparent"></span>
            <img
              v-if="siteLogo"
              :src="siteLogo"
              :alt="siteName"
              class="relative h-6 w-6 rounded-md object-contain"
            />
            <svg
              v-else
              class="relative"
              width="22"
              height="22"
              viewBox="0 0 32 32"
              fill="none"
              xmlns="http://www.w3.org/2000/svg"
            >
              <path d="M16 4L27 10V22L16 28L5 22V10L16 4Z" stroke="white" stroke-width="1.6" stroke-linejoin="round" stroke-opacity="0.55"></path>
              <path d="M16 10L22 13.5V20.5L16 24L10 20.5V13.5L16 10Z" fill="white" fill-opacity="0.95"></path>
              <circle cx="16" cy="17" r="2" fill="url(#footer-logo-core)"></circle>
              <defs>
                <linearGradient id="footer-logo-core" x1="14" y1="15" x2="18" y2="19" gradientUnits="userSpaceOnUse">
                  <stop stop-color="#2563EB"></stop>
                  <stop offset="1" stop-color="#9333EA"></stop>
                </linearGradient>
              </defs>
            </svg>
          </span>

          <div>
            <div class="text-lg font-semibold tracking-tight text-gray-900 dark:text-white">{{ siteName }}</div>
            <div class="text-sm text-gray-500 dark:text-dark-400">{{ siteSubtitle }}</div>
          </div>
        </div>

        <div class="flex items-center gap-4 text-sm text-gray-500 dark:text-dark-400">
          <button class="transition-colors hover:text-gray-800 dark:hover:text-white" @click="openDocs">
            {{ copy.docsLabel }}
          </button>
          <a
            :href="githubUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="transition-colors hover:text-gray-800 dark:hover:text-white"
          >
            GitHub
          </a>
          <span>© {{ currentYear }} {{ siteName }}.</span>
        </div>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useAuthStore, useAppStore } from '@/stores'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'
import { getPublicPricing, type PublicPricingConfig, type PublicPricingRow } from '@/api/pricing'
import type { ContactChannel, SitePage } from '@/types'
import { normalizeSitePages, resolveSitePageNavigationTarget } from '@/utils/sitePages'

type HomeIconName =
  | 'sparkles'
  | 'shield'
  | 'chartBar'
  | 'cpu'
  | 'bolt'
  | 'server'
  | 'terminal'
  | 'brain'
  | 'users'
  | 'clock'

interface NavItem {
  key: string
  label: string
  target: string
  external?: boolean
  dim?: boolean
}

interface IconTextItem {
  icon: HomeIconName
  text: string
}

interface FeatureCard {
  icon: HomeIconName
  title: string
  desc: string
}

interface EngineCard {
  icon: HomeIconName
  pill: string
  name: string
  models: string
  tone: 'orange' | 'green' | 'blue'
}

interface StatusItem {
  icon: HomeIconName
  label: string
  value: string
}

interface PricingRow {
  model: string
  group: string
  multiplier?: string
  officialInput: number
  officialOutput: number
  inputPrice: number
  outputPrice: number
  discount: string
  tone: 'purple' | 'blue' | 'green'
}

interface HomeCopy {
  nav: Omit<NavItem, 'target' | 'external' | 'dim'>[]
  dimNav: Omit<NavItem, 'target' | 'external' | 'dim'>[]
  headerCta: string
  announcement: string
  announcementBadge: string
  titleLead: string
  titleAccent: string
  subtitle: string
  description: string
  primaryCta: string
  secondaryCta: string
  floatingItems: IconTextItem[]
  heroBadges: IconTextItem[]
  featuresKicker: string
  featuresTitleLead: string
  featuresTitleAccent: string
  featuresSubtitle: string
  featureCards: FeatureCard[]
  enginesTitle: string
  enginesNote: string
  engineCards: EngineCard[]
  statusEyebrow: string
  statusItems: StatusItem[]
  pricingTitleLead: string
  pricingTitleAccent: string
  pricingNote: string
  pricingCurrency: {
    usd: string
    cny: string
  }
  pricingRechargeNote: string
  pricingDiscountFormula: string
  pricingDiscountBasis: string
  pricingDiscountTag: string
  pricingCols: {
    model: string
    group: string
    officialInput: string
    officialOutput: string
    convertedInput: string
    convertedOutput: string
    input: string
    output: string
    discount: string
  }
  pricingRows: PricingRow[]
  visionLead: string
  visionHighlightOne: string
  visionMiddle: string
  visionHighlightTwo: string
  visionEnd: string
  visionAuthor: string
  visionCta: string
  supportTitle: string
  supportSubtitle: string
  supportFallbackLabel: string
  docsLabel: string
}

const { t, locale } = useI18n()
const router = useRouter()
const authStore = useAuthStore()
const appStore = useAppStore()

const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API')
const siteLogo = computed(() => appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '')
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || 'AI API Gateway Platform')
const docUrl = computed(() => appStore.cachedPublicSettings?.doc_url || appStore.docUrl || '')
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')

const isHomeContentUrl = computed(() => {
  const content = homeContent.value.trim()
  return content.startsWith('http://') || content.startsWith('https://')
})

const isDark = ref(document.documentElement.classList.contains('dark'))
const githubUrl = 'https://github.com/Wei-Shaw/sub2api'
const isAuthenticated = computed(() => authStore.isAuthenticated)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => (isAdmin.value ? '/admin/dashboard' : '/dashboard'))
const currentYear = computed(() => new Date().getFullYear())
const contactChannels = computed<ContactChannel[]>(() =>
  (appStore.cachedPublicSettings?.contact_channels || []).filter((channel) => {
    return channel.label?.trim() && channel.url?.trim()
  })
)
const fallbackContactInfo = computed(() => appStore.cachedPublicSettings?.contact_info || appStore.contactInfo || '')
const sitePages = computed<SitePage[]>(() => normalizeSitePages(appStore.cachedPublicSettings?.site_pages || []))
const docsTarget = computed(() => resolveSitePageNavigationTarget(sitePages.value, 'docs'))
const termsTarget = computed(() => resolveSitePageNavigationTarget(sitePages.value, 'terms'))
const privacyTarget = computed(() => resolveSitePageNavigationTarget(sitePages.value, 'privacy'))
const docsHref = computed(() => docUrl.value || githubUrl)
const usdToCnyRate = 7

const zhCopy: HomeCopy = {
  nav: [
    { key: 'features', label: '特性' },
    { key: 'pricing', label: '定价' },
    { key: 'status', label: '状态' },
    { key: 'docs', label: '文档' }
  ],
  dimNav: [
    { key: 'terms', label: '服务条款' },
    { key: 'privacy', label: '隐私保护' },
    { key: 'image2', label: 'Image2生图' }
  ],
  headerCta: '控制台',
  announcement: 'Claude 4.7 和 GPT-5.5 现已可用',
  announcementBadge: '全新升级',
  titleLead: '极致优雅的',
  titleAccent: 'AI 接口引擎',
  subtitle: '更智能，更懂你 — Smarter. More capable. Closer to you.',
  description: '一行代码，直连全网最顶尖大模型。更稳定的架构，更极简的接入，低至官方 0.1 折的颠覆性定价。',
  primaryCta: '获取 API Key',
  secondaryCta: '查看文档',
  floatingItems: [
    { icon: 'sparkles', text: '多模态全面进化' },
    { icon: 'shield', text: '更稳定的对话' },
    { icon: 'chartBar', text: '更高效的生产力' },
    { icon: 'cpu', text: '突破边界 · 超越想象' }
  ],
  heroBadges: [
    { icon: 'bolt', text: '更聪明' },
    { icon: 'sparkles', text: '更全能' },
    { icon: 'shield', text: '更可靠' },
    { icon: 'chartBar', text: '更极致' }
  ],
  featuresKicker: '核心能力',
  featuresTitleLead: '重新定义',
  featuresTitleAccent: 'API 体验',
  featuresSubtitle: '从配置到调用，每一处细节都为你精心打磨。',
  featureCards: [
    { icon: 'bolt', title: '5 分钟极速接入', desc: '统一域名与密钥格式，无缝覆盖主流 IDE 与 CLI。' },
    { icon: 'server', title: '极致性价比', desc: '低至官方原价 0.1 折，计价透明无套路。' },
    { icon: 'chartBar', title: '可视化费用分析', desc: '精确到每次调用的日志与消费明细。' },
    { icon: 'terminal', title: '为编码而生', desc: '专注最强编码模型，拒绝滥竽充数。' }
  ],
  enginesTitle: '三大 AI 引擎 · 各司其职',
  enginesNote: '动态调度最适合模型',
  engineCards: [
    { icon: 'cpu', pill: '架构规划', name: 'Claude', models: 'Opus 4.7 · Sonnet 4.6 · Haiku 4.5', tone: 'orange' },
    { icon: 'brain', pill: '编码实战', name: 'ChatGPT', models: 'GPT-5.5 · GPT-5.4 · GPT-5.3 Codex', tone: 'green' },
    { icon: 'sparkles', pill: '多模态设计', name: 'Gemini', models: '2.5 Pro · 2.0 Flash', tone: 'blue' }
  ],
  statusEyebrow: '系统全绿运行',
  statusItems: [
    { icon: 'users', label: '注册开发者', value: '2,000+' },
    { icon: 'chartBar', label: '月调用次数', value: '100万+' },
    { icon: 'clock', label: '平均响应', value: '16.3ms' }
  ],
  pricingTitleLead: '模型',
  pricingTitleAccent: '定价',
  pricingNote: '官方原价以美元（USD）标注 · 折合价与 LumioAPI 价格以人民币（CNY）计价 · 单位：百万 tokens',
  pricingCurrency: {
    usd: '美元 USD',
    cny: '人民币 CNY'
  },
  pricingRechargeNote: '充值规则：¥1 人民币 = $1 美元额度',
  pricingDiscountFormula: '先享 1.43 折，再叠加渠道倍率算最终折扣',
  pricingDiscountBasis: '最终折扣',
  pricingDiscountTag: '最终折扣',
  pricingCols: {
    model: '模型',
    group: '分组',
    officialInput: '官方输入',
    officialOutput: '官方输出',
    convertedInput: '人民币折合',
    convertedOutput: '人民币折合',
    input: 'LumioAPI 输入价',
    output: 'LumioAPI 输出价',
    discount: '最终折扣'
  },
  pricingRows: [
    { model: 'Claude Opus 4.7', group: 'Claude', multiplier: '1.4', officialInput: 5, officialOutput: 25, inputPrice: 7, outputPrice: 35, discount: '2折', tone: 'purple' },
    { model: 'Claude Sonnet 4.6', group: 'Claude', multiplier: '1.4', officialInput: 3, officialOutput: 15, inputPrice: 4.2, outputPrice: 21, discount: '2折', tone: 'purple' },
    { model: 'GPT5.5', group: 'OpenAI', multiplier: '0.2', officialInput: 5, officialOutput: 30, inputPrice: 1, outputPrice: 6, discount: '0.29折', tone: 'blue' },
    { model: 'GPT5.4', group: 'OpenAI', multiplier: '0.2', officialInput: 2.5, officialOutput: 15, inputPrice: 0.5, outputPrice: 3, discount: '0.29折', tone: 'blue' }
  ],
  visionLead: '让每一个开发者都能以',
  visionHighlightOne: '可接受的价格',
  visionMiddle: '与',
  visionHighlightTwo: '稳定的服务',
  visionEnd: '，触达世界上最聪明的 AI。',
  visionAuthor: 'Lumio · OPC 创业团队',
  visionCta: '加入我们的愿景',
  supportTitle: '技术支持',
  supportSubtitle: '选择任意联系渠道，直接加入 QQ 群、飞书群或 Telegram 群获取支持。',
  supportFallbackLabel: '客服联系方式',
  docsLabel: '文档'
}

const enCopy: HomeCopy = {
  nav: [
    { key: 'features', label: 'Features' },
    { key: 'pricing', label: 'Pricing' },
    { key: 'status', label: 'Status' },
    { key: 'docs', label: 'Docs' }
  ],
  dimNav: [
    { key: 'terms', label: 'Terms' },
    { key: 'privacy', label: 'Privacy' },
    { key: 'image2', label: 'Image2生图' }
  ],
  headerCta: 'Console',
  announcement: 'Claude 4.7 and GPT-5.5 are now live',
  announcementBadge: 'New',
  titleLead: 'An elegantly built',
  titleAccent: 'AI interface engine',
  subtitle: 'Smarter. More capable. Closer to you.',
  description: 'Connect to the best frontier models through one line of configuration, with steadier routing, simpler setup, and disruptive pricing down to 10% of list price.',
  primaryCta: 'Get API Key',
  secondaryCta: 'View docs',
  floatingItems: [
    { icon: 'sparkles', text: 'Multimodal upgrade' },
    { icon: 'shield', text: 'More stable sessions' },
    { icon: 'chartBar', text: 'Higher productivity' },
    { icon: 'cpu', text: 'Push past the limit' }
  ],
  heroBadges: [
    { icon: 'bolt', text: 'Smarter' },
    { icon: 'sparkles', text: 'More capable' },
    { icon: 'shield', text: 'More reliable' },
    { icon: 'chartBar', text: 'More refined' }
  ],
  featuresKicker: 'Core capabilities',
  featuresTitleLead: 'Redefining the',
  featuresTitleAccent: 'API experience',
  featuresSubtitle: 'From setup to daily usage, every detail is shaped to feel cleaner and faster.',
  featureCards: [
    { icon: 'bolt', title: 'Go live in five minutes', desc: 'One domain and one key format that work cleanly across mainstream IDEs and CLIs.' },
    { icon: 'server', title: 'Extreme value', desc: 'Pricing drops as low as 10% of list price, with transparent billing and no surprises.' },
    { icon: 'chartBar', title: 'Visual cost analysis', desc: 'Track every request with detailed logs, spend data, and model-level visibility.' },
    { icon: 'terminal', title: 'Built for coding', desc: 'Focused on top-tier coding models instead of trying to be everything at once.' }
  ],
  enginesTitle: 'Three AI engines, each in its lane',
  enginesNote: 'Route work to the model that fits it best',
  engineCards: [
    { icon: 'cpu', pill: 'Architecture', name: 'Claude', models: 'Opus 4.7 · Sonnet 4.6 · Haiku 4.5', tone: 'orange' },
    { icon: 'brain', pill: 'Coding', name: 'ChatGPT', models: 'GPT-5.5 · GPT-5.4 · GPT-5.3 Codex', tone: 'green' },
    { icon: 'sparkles', pill: 'Multimodal', name: 'Gemini', models: '2.5 Pro · 2.0 Flash', tone: 'blue' }
  ],
  statusEyebrow: 'System fully green',
  statusItems: [
    { icon: 'users', label: 'Registered developers', value: '2,000+' },
    { icon: 'chartBar', label: 'Monthly calls', value: '1M+' },
    { icon: 'clock', label: 'Average latency', value: '16.3ms' }
  ],
  pricingTitleLead: 'Model',
  pricingTitleAccent: 'pricing',
  pricingNote: 'Official list price is in USD · converted and LumioAPI prices are in CNY · unit: per million tokens',
  pricingCurrency: {
    usd: 'USD',
    cny: 'CNY'
  },
  pricingRechargeNote: 'Recharge rule: ¥1 CNY = $1 USD credit',
  pricingDiscountFormula: 'Start at 14.3%, then apply the channel multiplier',
  pricingDiscountBasis: 'Final discount',
  pricingDiscountTag: 'Final',
  pricingCols: {
    model: 'Model',
    group: 'Group',
    officialInput: 'Official input',
    officialOutput: 'Official output',
    convertedInput: 'Converted input',
    convertedOutput: 'Converted output',
    input: 'LumioAPI input',
    output: 'LumioAPI output',
    discount: 'Discount'
  },
  pricingRows: [
    { model: 'Claude Opus 4.7', group: 'Claude', multiplier: '1.4', officialInput: 5, officialOutput: 25, inputPrice: 7, outputPrice: 35, discount: '20%', tone: 'purple' },
    { model: 'Claude Sonnet 4.6', group: 'Claude', multiplier: '1.4', officialInput: 3, officialOutput: 15, inputPrice: 4.2, outputPrice: 21, discount: '20%', tone: 'purple' },
    { model: 'GPT5.5', group: 'OpenAI', multiplier: '0.2', officialInput: 5, officialOutput: 30, inputPrice: 1, outputPrice: 6, discount: '2.86%', tone: 'blue' },
    { model: 'GPT5.4', group: 'OpenAI', multiplier: '0.2', officialInput: 2.5, officialOutput: 15, inputPrice: 0.5, outputPrice: 3, discount: '2.86%', tone: 'blue' }
  ],
  visionLead: 'We want every developer to reach the world’s smartest AI with',
  visionHighlightOne: 'pricing they can accept',
  visionMiddle: 'and',
  visionHighlightTwo: 'service they can trust',
  visionEnd: '.',
  visionAuthor: 'Lumio · OPC founding team',
  visionCta: 'Join the vision',
  supportTitle: 'Support',
  supportSubtitle: 'Choose any contact channel to reach the support community directly.',
  supportFallbackLabel: 'Contact',
  docsLabel: 'Docs'
}

const copy = computed(() => (locale.value.startsWith('zh') ? zhCopy : enCopy))
const publicPricing = ref<PublicPricingConfig | null>(null)
const pricingNote = computed(() => {
  const configuredNote = normalizePricingNote(publicPricing.value?.rateNote)
  return configuredNote || copy.value.pricingNote
})
const pricingRows = computed<PricingRow[]>(() => {
  const configuredRows = publicPricing.value?.rows
  const rows: Array<PricingRow | PublicPricingRow> =
    configuredRows && configuredRows.length > 0 ? configuredRows : copy.value.pricingRows
  return rows.map(toPricingRow)
})

const navItems = computed<NavItem[]>(() => [
  { ...copy.value.nav[0], target: '#features' },
  { ...copy.value.nav[1], target: '#pricing' },
  { ...copy.value.nav[2], key: 'status', target: '/status' },
  docsTarget.value
    ? {
        ...copy.value.nav[3],
        target: docsTarget.value.target
      }
    : { ...copy.value.nav[3], target: docsHref.value, external: true },
  {
    ...copy.value.dimNav[0],
    target: termsTarget.value?.target || '#footer',
    dim: true
  },
  {
    ...copy.value.dimNav[1],
    target: privacyTarget.value?.target || '#footer',
    dim: true
  },
  { key: 'image2', label: copy.value.dimNav[2].label, target: 'https://img.lumio.games/', external: true, dim: true }
])

const floatingSlots = [
  'top-[12%] left-[6%]',
  'top-[14%] right-[6%]',
  'top-[40%] left-[3%]',
  'top-[42%] right-[3%]'
]

const floatingIconTones = ['text-purple-500', 'text-blue-500', 'text-indigo-500', 'text-cyan-500']
const badgeIconTones = ['text-blue-500', 'text-purple-500', 'text-cyan-500', 'text-indigo-500']
const statusIconTones = ['text-blue-600', 'text-purple-600', 'text-emerald-600']

function initTheme() {
  const savedTheme = localStorage.getItem('theme')
  if (
    savedTheme === 'dark' ||
    (!savedTheme && window.matchMedia('(prefers-color-scheme: dark)').matches)
  ) {
    isDark.value = true
    document.documentElement.classList.add('dark')
  }
}

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

function scrollTo(target: string) {
  document.querySelector(target)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

function openDocs() {
  if (docsTarget.value?.kind === 'route') {
    router.push(docsTarget.value.target)
    return
  }
  window.open(docsHref.value, '_blank', 'noopener,noreferrer')
}

function onNav(item: NavItem) {
  if (item.external) {
    window.open(item.target, '_blank', 'noopener,noreferrer')
    return
  }
  if (item.target.startsWith('/')) {
    router.push(item.target)
    return
  }
  scrollTo(item.target)
}

function openContact(url: string) {
  window.open(url, '_blank', 'noopener,noreferrer')
}

function goConsole() {
  router.push(isAuthenticated.value ? dashboardPath.value : '/login')
}

function goPrimary() {
  router.push(isAuthenticated.value ? dashboardPath.value : '/register')
}

function engineTone(tone: EngineCard['tone']) {
  if (tone === 'orange') {
    return { icon: 'text-orange-500 dark:text-orange-300', pill: 'from-orange-500 to-orange-600' }
  }
  if (tone === 'green') {
    return { icon: 'text-green-500 dark:text-green-300', pill: 'from-green-500 to-green-600' }
  }
  return { icon: 'text-blue-500 dark:text-blue-300', pill: 'from-blue-500 to-blue-600' }
}

function pricingTone(row: PricingRow) {
  if (row.tone === 'purple') {
    return {
      row: 'bg-purple-50/30 hover:bg-purple-50/50 dark:bg-purple-950/10 dark:hover:bg-purple-950/20',
      price: 'border-purple-200 bg-purple-50 text-purple-700 dark:border-purple-900/40 dark:bg-purple-950/30 dark:text-purple-300',
      discount: 'from-purple-500 via-fuchsia-500 to-purple-600',
      discountText: 'shadow-purple-500/35',
      flame: false
    }
  }

  if (row.tone === 'green') {
    return {
      row: 'relative z-10 border-l-4 border-emerald-500 bg-gradient-to-r from-emerald-50 via-green-50/80 to-emerald-50 shadow-[0_0_15px_rgba(16,185,129,0.15)] dark:from-emerald-950/30 dark:via-green-950/20 dark:to-emerald-950/30',
      price: 'border-green-200 bg-green-50 text-green-700 dark:border-green-900/40 dark:bg-green-950/30 dark:text-green-300',
      discount: 'from-green-500 to-emerald-600',
      discountText: 'px-5 py-2 text-base shadow-green-500/30',
      flame: true
    }
  }

  return {
    row: 'hover:bg-gray-50 dark:hover:bg-dark-800',
    price: 'border-blue-200 bg-blue-50 text-blue-700 dark:border-blue-900/40 dark:bg-blue-950/30 dark:text-blue-300',
    discount: 'from-blue-500 via-sky-500 to-blue-600',
    discountText: 'shadow-blue-500/35',
    flame: false
  }
}

function pricingToneForRow(row: Pick<PricingRow, 'model' | 'group'>): PricingRow['tone'] {
  const text = `${row.model} ${row.group}`.toLowerCase()
  if (text.includes('claude') || text.includes('opus') || text.includes('sonnet')) {
    return 'purple'
  }
  if (text.includes('gemini') || text.includes('google')) {
    return 'green'
  }
  return 'blue'
}

function toPricingRow(row: PricingRow | PublicPricingRow): PricingRow {
  const officialInput = Number(row.officialInput) || 0
  const officialOutput = Number(row.officialOutput) || 0
  const multiplier = parsePricingMultiplier(row.multiplier)
  const derivedInputPrice = calculateLumioPrice(officialInput, multiplier)
  const derivedOutputPrice = calculateLumioPrice(officialOutput, multiplier)

  return {
    model: row.model,
    group: row.group,
    multiplier: row.multiplier,
    officialInput,
    officialOutput,
    inputPrice: derivedInputPrice ?? (Number(row.inputPrice) || 0),
    outputPrice: derivedOutputPrice ?? (Number(row.outputPrice) || 0),
    discount: multiplier > 0 ? formatFinalDiscount(multiplier) : row.discount || '',
    tone: 'tone' in row ? row.tone : pricingToneForRow(row)
  }
}

async function loadPublicPricing() {
  try {
    publicPricing.value = await getPublicPricing()
  } catch (error) {
    console.warn('Failed to load public pricing', error)
  }
}

function normalizePricingNote(note?: string) {
  const cleaned = note
    ?.trim()
    .replace(/[；;]\s*折扣展示可在管理员后台调整。?$/, '')
    .replace(/[；;]\s*价格与折扣支持管理员后台配置。?$/, '')
    .trim()

  if (!cleaned || cleaned === '价格以人民币（¥）计价，单位为百万 tokens') {
    return ''
  }
  return cleaned
}

function parsePricingMultiplier(value?: string) {
  const raw = value?.trim()
  if (!raw) return 0

  const normalized = raw
    .replace(/倍率/g, '')
    .replace(/[xX倍\s]/g, '')
    .trim()

  if (!normalized) return 0

  const numeric = Number.parseFloat(normalized)
  if (!Number.isFinite(numeric) || numeric <= 0) return 0

  if (normalized.endsWith('%')) {
    return numeric / 100
  }
  if (normalized.endsWith('折')) {
    return (numeric / 10) * usdToCnyRate
  }
  return numeric
}

function calculateLumioPrice(officialUsd: number, multiplier: number) {
  if (officialUsd <= 0 || multiplier <= 0) return null
  return Math.round(officialUsd * multiplier * 100) / 100
}

function formatFinalDiscount(multiplier: number) {
  const ratio = multiplier / usdToCnyRate
  if (locale.value.startsWith('zh')) {
    return `${formatCompactNumber(ratio * 10)}折`
  }
  return `${formatCompactNumber(ratio * 100)}%`
}

function formatCompactNumber(value: number) {
  return value
    .toFixed(value < 1 ? 2 : 1)
    .replace(/(\.\d*?[1-9])0+$/, '$1')
    .replace(/\.0$/, '')
}

function formatUsd(value: number) {
  const fixed = Number.isInteger(value) ? value.toFixed(0) : value.toFixed(value < 0.1 ? 3 : 2)
  return `$${fixed.replace(/(\.\d*?[1-9])0+$/, '$1').replace(/\.00$/, '')}`
}

function formatCny(value: number) {
  return `¥${value.toFixed(2)}`
}

function formatConverted(value: number) {
  const converted = (value * 7).toFixed(2)
  return `¥${converted.replace(/\.00$/, '')}`
}

onMounted(() => {
  initTheme()
  authStore.checkAuth()
  void loadPublicPricing()
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
})
</script>

<style scoped>
.hero-grid {
  background-image:
    linear-gradient(to right, #f0f0f0 1px, transparent 1px),
    linear-gradient(to bottom, #f0f0f0 1px, transparent 1px);
  background-size: 4rem 4rem;
  mask-image: radial-gradient(ellipse 60% 50% at 50% 0%, #000 70%, transparent 100%);
}

.fade-rise {
  animation: fade-rise 0.8s ease-out both;
  animation-delay: var(--delay, 0ms);
}

.float-pill {
  animation: float-pill 8s ease-in-out infinite;
}

.orbital-shell {
  box-shadow: inset 0 0 60px rgba(99, 102, 241, 0.08);
  animation: pulse-ring 10s ease-in-out infinite;
}

.orbital-ring {
  animation: drift-spin 18s linear infinite;
}

.orbital-nodes {
  animation: drift-spin 14s linear infinite;
}

.animate-blob {
  animation: blob 14s infinite ease-in-out;
}

.animation-delay-2000 {
  animation-delay: 2s;
}

@keyframes fade-rise {
  from {
    opacity: 0;
    transform: translateY(20px);
  }

  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@keyframes float-pill {
  0%,
  100% {
    transform: translateY(0);
  }

  50% {
    transform: translateY(-8px);
  }
}

@keyframes pulse-ring {
  0%,
  100% {
    transform: scale(1);
  }

  50% {
    transform: scale(1.03);
  }
}

@keyframes drift-spin {
  from {
    transform: rotate(0deg);
  }

  to {
    transform: rotate(360deg);
  }
}

@keyframes blob {
  0%,
  100% {
    transform: translate3d(0, 0, 0) scale(1);
  }

  33% {
    transform: translate3d(18px, -28px, 0) scale(1.04);
  }

  66% {
    transform: translate3d(-22px, 14px, 0) scale(0.98);
  }
}

@media (prefers-reduced-motion: reduce) {
  .fade-rise,
  .float-pill,
  .orbital-shell,
  .orbital-ring,
  .orbital-nodes,
  .animate-blob {
    animation: none !important;
  }
}
</style>
