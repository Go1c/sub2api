<script setup lang="ts"></script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-ink-50 p-6">
    <div class="w-full max-w-md">
      <div class="flex items-center gap-2 justify-center mb-8">
        <div class="w-10 h-10 rounded-lg bg-gradient-to-br from-brand-500 to-brand-900 flex items-center justify-center text-white text-lg font-semibold">
          L
        </div>
        <div>
          <div class="font-semibold text-xl text-ink-900 leading-tight">LumioAPI</div>
          <div class="text-xs text-ink-500">{{ $t('site.tagline') }}</div>
        </div>
      </div>
      <div class="bg-white rounded-xl border border-ink-100 p-8">
        <RouterView />
      </div>
      <div class="mt-6 text-center text-xs text-ink-400">
        {{ $t('site.footer', { year: new Date().getFullYear() }) }}
      </div>
    </div>
  </div>
</template>
