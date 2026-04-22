<script setup lang="ts">
import { useRouter } from 'vue-router'
const router = useRouter()
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-hero-gradient text-center">
    <div>
      <h1 class="brand-serif text-5xl font-semibold text-ink-900">404</h1>
      <p class="mt-3 text-ink-500 ui-sans">Page not found</p>
      <button class="btn-brand mt-6" @click="router.replace('/dashboard')">
        回到仪表盘
      </button>
    </div>
  </div>
</template>
