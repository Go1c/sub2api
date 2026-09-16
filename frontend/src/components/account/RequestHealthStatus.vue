<script setup lang="ts">
import { computed } from 'vue'
import Icon from '@/components/icons/Icon.vue'
import { formatCountdown, resolveBadge, type RequestHealthSnapshot } from './requestHealth'

const props = defineProps<{
  health: RequestHealthSnapshot
  now: number
}>()

const badge = computed(() => resolveBadge(props.health, props.now))
const countdown = computed(() =>
  props.health.cooldownUntil ? formatCountdown(props.health.cooldownUntil, props.now) : '00:00'
)
</script>

<template>
  <div class="flex items-center whitespace-nowrap">
    <span
      v-if="badge === 'cooling'"
      class="inline-flex items-center gap-1 rounded-full bg-amber-500/15 px-2 py-0.5 text-sm text-amber-400"
    >
      <Icon name="clock" size="xs" class="text-amber-400" />
      {{ $t('admin.requestHealth.cooling', { time: countdown }) }}
    </span>
    <span
      v-else-if="badge === 'rate_limited'"
      class="inline-flex items-center gap-1 rounded-full bg-amber-500/15 px-2 py-0.5 text-sm text-amber-400"
    >
      {{ $t('admin.requestHealth.rateLimited') }}
    </span>
    <span
      v-else-if="badge === 'overloaded'"
      class="inline-flex items-center gap-1 rounded-full bg-red-500/15 px-2 py-0.5 text-sm text-red-400"
    >
      {{ $t('admin.requestHealth.overloaded') }}
    </span>
    <span
      v-else-if="badge === 'executing'"
      class="text-sm text-[#3b82f6]"
    >
      {{ $t('admin.requestHealth.executing', { n: health.current }) }}
    </span>
    <span v-else class="text-sm text-gray-400 dark:text-gray-500">{{ $t('admin.requestHealth.idle') }}</span>
  </div>
</template>
