<script setup lang="ts">
import SiteHeader from '@/components/marketing/SiteHeader.vue'
import SiteFooter from '@/components/marketing/SiteFooter.vue'
</script>

<template>
  <div class="min-h-screen bg-white flex flex-col">
    <SiteHeader />
    <main class="flex-1">
      <RouterView />
    </main>
    <SiteFooter />
  </div>
</template>
