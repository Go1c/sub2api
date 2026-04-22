<script setup lang="ts"></script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-hero-gradient bg-ink-50 p-6">
    <div class="w-full max-w-md">
      <div class="flex items-center gap-3 justify-center mb-8">
        <img src="/logo.svg" alt="logo" class="w-10 h-10 rounded-md" />
        <div class="brand-serif">
          <div class="font-semibold text-2xl text-ink-900">Dragon Code</div>
          <div class="text-xs text-ink-500">{{ $t('site.tagline') }}</div>
        </div>
      </div>
      <div class="bg-white rounded-lg shadow-card p-8">
        <RouterView />
      </div>
      <div class="mt-6 text-center text-[11px] text-ink-400 ui-sans">
        {{ $t('site.footer', { year: new Date().getFullYear() }) }}
      </div>
    </div>
  </div>
</template>
