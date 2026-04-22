<script setup lang="ts">
import Sidebar from './Sidebar.vue'
import TopBar from './TopBar.vue'
</script>

<template>
  <div class="h-screen flex bg-[var(--color-bg)]">
    <Sidebar />
    <div class="flex-1 flex flex-col min-w-0">
      <TopBar />
      <main class="flex-1 overflow-auto px-6 py-5">
        <RouterView />
      </main>
    </div>
  </div>
</template>
