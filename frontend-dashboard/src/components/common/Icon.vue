<script setup lang="ts">
defineProps<{ name: IconName }>()

export type IconName =
  | 'dashboard'
  | 'key'
  | 'chart'
  | 'bolt'
  | 'gift'
  | 'doc'
  | 'user'
  | 'dollar'
  | 'cube'
  | 'database'
  | 'clock'
  | 'gauge'
  | 'bell'
  | 'announce'
  | 'globe'
  | 'sun'
  | 'moon'
  | 'chevron-left'
  | 'chevron-down'
  | 'copy'
  | 'plus'
  | 'check'
</script>

<template>
  <svg
    xmlns="http://www.w3.org/2000/svg"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    stroke-width="1.8"
    stroke-linecap="round"
    stroke-linejoin="round"
  >
    <template v-if="name === 'dashboard'">
      <rect x="3" y="3" width="7" height="9" rx="1.5" />
      <rect x="14" y="3" width="7" height="5" rx="1.5" />
      <rect x="14" y="12" width="7" height="9" rx="1.5" />
      <rect x="3" y="16" width="7" height="5" rx="1.5" />
    </template>
    <template v-else-if="name === 'key'">
      <circle cx="7.5" cy="15.5" r="3.5" />
      <path d="M10 13l9-9M15 4l4 4M13.5 5.5l4 4" />
    </template>
    <template v-else-if="name === 'chart'">
      <path d="M3 3v18h18" />
      <path d="M7 14l4-4 4 3 5-7" />
    </template>
    <template v-else-if="name === 'bolt'">
      <path d="M13 2L3 14h8l-1 8 10-12h-8l1-8z" />
    </template>
    <template v-else-if="name === 'gift'">
      <rect x="3" y="8" width="18" height="4" rx="1" />
      <path d="M12 8v13M5 12v8a1 1 0 001 1h12a1 1 0 001-1v-8" />
      <path d="M7.5 8a2.5 2.5 0 010-5c2 0 4.5 5 4.5 5s2.5-5 4.5-5a2.5 2.5 0 010 5" />
    </template>
    <template v-else-if="name === 'doc'">
      <path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z" />
      <path d="M14 2v6h6M8 13h8M8 17h6" />
    </template>
    <template v-else-if="name === 'user'">
      <circle cx="12" cy="8" r="4" />
      <path d="M4 21c0-4 4-7 8-7s8 3 8 7" />
    </template>
    <template v-else-if="name === 'dollar'">
      <path d="M12 2v20M17 6H9.5a3 3 0 100 6h5a3 3 0 110 6H6" />
    </template>
    <template v-else-if="name === 'cube'">
      <path d="M21 8l-9-5-9 5 9 5 9-5z" />
      <path d="M3 8v8l9 5 9-5V8" />
      <path d="M12 13v8" />
    </template>
    <template v-else-if="name === 'database'">
      <ellipse cx="12" cy="5" rx="8" ry="2.5" />
      <path d="M4 5v6c0 1.4 3.6 2.5 8 2.5s8-1.1 8-2.5V5" />
      <path d="M4 11v6c0 1.4 3.6 2.5 8 2.5s8-1.1 8-2.5v-6" />
    </template>
    <template v-else-if="name === 'clock'">
      <circle cx="12" cy="12" r="9" />
      <path d="M12 7v5l3 2" />
    </template>
    <template v-else-if="name === 'gauge'">
      <path d="M12 14l4-4" />
      <path d="M3 17a9 9 0 1118 0" />
    </template>
    <template v-else-if="name === 'bell'">
      <path d="M18 8a6 6 0 10-12 0c0 7-3 9-3 9h18s-3-2-3-9" />
      <path d="M13.7 21a2 2 0 01-3.4 0" />
    </template>
    <template v-else-if="name === 'announce'">
      <path d="M3 11v2a3 3 0 003 3h3l6 4V4l-6 4H6a3 3 0 00-3 3z" />
    </template>
    <template v-else-if="name === 'globe'">
      <circle cx="12" cy="12" r="9" />
      <path d="M3 12h18M12 3c3 3.5 3 14 0 18M12 3c-3 3.5-3 14 0 18" />
    </template>
    <template v-else-if="name === 'sun'">
      <circle cx="12" cy="12" r="4" />
      <path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4" />
    </template>
    <template v-else-if="name === 'moon'">
      <path d="M21 12.8A9 9 0 1111.2 3a7 7 0 009.8 9.8z" />
    </template>
    <template v-else-if="name === 'chevron-left'">
      <path d="M15 18l-6-6 6-6" />
    </template>
    <template v-else-if="name === 'chevron-down'">
      <path d="M6 9l6 6 6-6" />
    </template>
    <template v-else-if="name === 'copy'">
      <rect x="9" y="9" width="12" height="12" rx="2" />
      <path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1" />
    </template>
    <template v-else-if="name === 'plus'">
      <path d="M12 5v14M5 12h14" />
    </template>
    <template v-else-if="name === 'check'">
      <path d="M5 12l5 5L20 7" />
    </template>
  </svg>
</template>
