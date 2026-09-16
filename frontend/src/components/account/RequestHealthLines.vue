<script setup lang="ts">
import { computed } from 'vue'
import RequestHealthBar from './RequestHealthBar.vue'
import RequestHealthStatus from './RequestHealthStatus.vue'
import { resolveBadge, type AccountHealthRow } from './requestHealth'

const props = defineProps<{
  row: AccountHealthRow
  windowSize: number
  now: number
}>()

const isGroup = computed(() => props.row.mode === 'ip_group')
const isScrollList = computed(() => isGroup.value && props.row.lines.length > 2)

function dotClass(health: AccountHealthRow['lines'][number]['health']): string {
  const badge = resolveBadge(health, props.now)
  if (badge === 'cooling' || badge === 'rate_limited') return 'bg-amber-400'
  if (badge === 'overloaded') return 'bg-red-500'
  if (badge === 'executing') return 'bg-blue-500'
  if (health.outcomes.length === 0) return 'bg-gray-500'
  return 'bg-emerald-400'
}
</script>

<template>
  <div :class="isGroup ? 'w-[28rem]' : 'min-w-[18rem]'">
    <div
      v-if="isGroup"
      class="overflow-hidden"
      :class="isScrollList ? 'max-h-[5rem] overflow-y-auto overscroll-contain' : ''"
      @wheel.stop
    >
      <div class="flex flex-col">
        <div
          v-for="item in row.lines"
          :key="item.ip"
          class="grid grid-cols-[9rem_1fr_7.5rem] items-center gap-3 px-1 py-2.5"
        >
          <div class="flex items-center gap-2">
            <span class="h-2 w-2 flex-shrink-0 rounded-full" :class="dotClass(item.health)" />
            <code class="font-mono text-[13px] text-gray-700 dark:text-gray-200">{{ item.ip }}</code>
          </div>
          <RequestHealthBar
            :outcomes="item.health.outcomes"
            :window-size="windowSize"
            :source-label="item.ip"
          />
          <RequestHealthStatus :health="item.health" :now="now" />
        </div>
      </div>
    </div>

    <div v-else class="flex items-center gap-4">
      <span class="h-2 w-2 flex-shrink-0 rounded-full" :class="dotClass(row.lines[0]?.health ?? { outcomes: [], current: 0, max: 1 })" />
      <RequestHealthBar
        :outcomes="row.lines[0]?.health.outcomes ?? []"
        :window-size="windowSize"
        :source-label="row.lines[0]?.ip"
      />
      <RequestHealthStatus :health="row.lines[0]?.health ?? { outcomes: [], current: 0, max: 1 }" :now="now" />
    </div>
  </div>
</template>
