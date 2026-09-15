<template>
  <div>
    <div class="mb-1 flex items-center gap-2">
      <label class="input-label mb-0">{{ t('admin.accounts.proxy') }}</label>
      <slot name="banner" />
    </div>

    <div v-if="isOpenAIOAuth" class="mb-3 flex flex-wrap gap-4 text-sm">
      <label class="inline-flex items-center gap-2">
        <input
          type="radio"
          class="h-4 w-4 border-gray-300 text-primary-600 focus:ring-primary-500"
          :checked="mode === 'single'"
          @change="setMode('single')"
        />
        {{ t('admin.accounts.proxyModeSingle') }}
      </label>
      <label class="inline-flex items-center gap-2">
        <input
          type="radio"
          class="h-4 w-4 border-gray-300 text-primary-600 focus:ring-primary-500"
          :checked="mode === 'group'"
          @change="setMode('group')"
        />
        {{ t('admin.accounts.proxyModeGroup') }}
      </label>
    </div>

    <ProxySelector
      v-if="!isOpenAIOAuth || mode === 'single'"
      :model-value="proxyId"
      :proxies="proxies"
      @update:model-value="emit('update:proxyId', $event)"
    />

    <div v-else>
      <Select
        :model-value="proxyIpGroupId == null ? '' : String(proxyIpGroupId)"
        :options="groupOptions"
        :placeholder="t('admin.accounts.selectIpGroup')"
        @update:model-value="onGroupChange"
      />
      <p class="input-hint">{{ t('admin.accounts.ipGroupHint') }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import ProxySelector from '@/components/common/ProxySelector.vue'
import Select from '@/components/common/Select.vue'
import type { AccountPlatform, AccountType, Proxy, ProxyIPGroup } from '@/types'

const props = defineProps<{
  platform: AccountPlatform
  type: AccountType | string
  proxyId: number | null
  proxyIpGroupId: number | null
  proxies: Proxy[]
  ipGroups: ProxyIPGroup[]
}>()

const emit = defineEmits<{
  'update:proxyId': [value: number | null]
  'update:proxyIpGroupId': [value: number | null]
}>()

const { t } = useI18n()

const isOpenAIOAuth = computed(() => props.platform === 'openai' && props.type === 'oauth')

const mode = computed<'single' | 'group'>(() => {
  if (!isOpenAIOAuth.value) return 'single'
  return props.proxyIpGroupId && props.proxyIpGroupId > 0 ? 'group' : 'single'
})

const groupOptions = computed(() => [
  { value: '', label: t('admin.accounts.noIpGroup') },
  ...props.ipGroups.map((group) => ({
    value: String(group.id),
    label: `${group.name} (${t('admin.accounts.ipGroupConcurrency', { count: group.per_ip_concurrency })})`
  }))
])

const setMode = (next: 'single' | 'group') => {
  if (next === 'group') {
    emit('update:proxyId', null)
    if (!props.proxyIpGroupId && props.ipGroups[0]) {
      emit('update:proxyIpGroupId', props.ipGroups[0].id)
    }
    return
  }
  emit('update:proxyIpGroupId', null)
}

const onGroupChange = (value: string | number | boolean | null) => {
  if (value === true || value === false) return
  const raw = String(value ?? '')
  emit('update:proxyIpGroupId', raw === '' ? null : Number(raw))
}

watch(isOpenAIOAuth, (enabled) => {
  if (!enabled) {
    emit('update:proxyIpGroupId', null)
  }
})
</script>
