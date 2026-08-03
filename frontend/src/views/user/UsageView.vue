<template>
  <AppLayout>
    <TablePageLayout>
      <template #actions>
        <div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
          <!-- Total Requests -->
          <div class="card p-4">
          <div class="flex items-center gap-3">
            <div class="rounded-lg bg-blue-100 p-2 dark:bg-blue-900/30">
              <Icon name="document" size="md" class="text-blue-600 dark:text-blue-400" />
            </div>
            <div>
              <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
                {{ t('usage.totalRequests') }}
              </p>
              <p class="text-xl font-bold text-gray-900 dark:text-white">
                {{ usageStats?.total_requests?.toLocaleString() || '0' }}
              </p>
              <p class="text-xs text-gray-500 dark:text-gray-400">
                {{ t('usage.inSelectedRange') }}
              </p>
            </div>
          </div>
        </div>

        <!-- Total Tokens -->
        <div class="card p-4">
          <div class="flex items-center gap-3">
            <div class="rounded-lg bg-amber-100 p-2 dark:bg-amber-900/30">
              <Icon name="cube" size="md" class="text-amber-600 dark:text-amber-400" />
            </div>
            <div>
              <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
                {{ t('usage.totalTokens') }}
              </p>
              <p class="text-xl font-bold text-gray-900 dark:text-white">
                {{ formatTokens(usageStats?.total_tokens || 0) }}
              </p>
              <p class="text-xs text-gray-500 dark:text-gray-400">
                {{ t('usage.in') }}: {{ formatTokens(usageStats?.total_input_tokens || 0) }} /
                {{ t('usage.out') }}: {{ formatTokens(usageStats?.total_output_tokens || 0) }}
              </p>
            </div>
          </div>
        </div>

        <!-- Total Cost -->
        <div class="card p-4">
          <div class="flex items-center gap-3">
            <div class="rounded-lg bg-green-100 p-2 dark:bg-green-900/30">
              <Icon name="dollar" size="md" class="text-green-600 dark:text-green-400" />
            </div>
            <div class="min-w-0 flex-1">
              <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
                {{ t('usage.totalCost') }}
              </p>
              <p class="text-xl font-bold text-green-600 dark:text-green-400">
                ${{ (usageStats?.total_actual_cost || 0).toFixed(4) }}
              </p>
              <p class="text-xs text-gray-500 dark:text-gray-400">
                {{ t('usage.actualCost') }} /
                <span class="line-through">${{ (usageStats?.total_cost || 0).toFixed(4) }}</span>
                {{ t('usage.standardCost') }}
              </p>
            </div>
          </div>
        </div>

        <!-- Average Duration -->
        <div class="card p-4">
          <div class="flex items-center gap-3">
            <div class="rounded-lg bg-purple-100 p-2 dark:bg-purple-900/30">
              <Icon name="clock" size="md" class="text-purple-600 dark:text-purple-400" />
            </div>
            <div>
              <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
                {{ t('usage.avgDuration') }}
              </p>
              <p class="text-xl font-bold text-gray-900 dark:text-white">
                {{ formatDuration(usageStats?.average_duration_ms || 0) }}
              </p>
              <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('usage.perRequest') }}</p>
            </div>
          </div>
        </div>
        </div>
      </template>

      <template #filters>
        <div class="card">
          <div class="px-6 py-4">
          <div class="flex flex-wrap items-end gap-4">
            <!-- Error tab filters -->
            <template v-if="activeTab === 'errors'">
              <div class="min-w-[180px]">
                <label class="input-label">{{ t('usage.errors.keyName') }}</label>
                <Select
                  v-model="errorFilter.api_key_id"
                  :options="errorKeyOptions"
                  @change="applyErrorFilters"
                />
              </div>
              <div class="min-w-[180px]">
                <label class="input-label">{{ t('usage.errors.model') }}</label>
                <Select
                  v-model="errorFilter.model"
                  :options="errorModelOptions"
                  searchable
                  creatable
                  clearable
                  :placeholder="t('usage.errors.modelPlaceholder')"
                  @change="applyErrorFilters"
                />
              </div>
              <div class="min-w-[160px]">
                <label class="input-label">{{ t('usage.errors.category') }}</label>
                <Select
                  v-model="errorFilter.category"
                  :options="errorCategoryOptions"
                  @change="applyErrorFilters"
                />
              </div>
              <div class="min-w-[140px]">
                <label class="input-label">{{ t('usage.errors.status') }}</label>
                <Select
                  v-model="errorFilter.status_code"
                  :options="errorStatusOptions"
                  @change="applyErrorFilters"
                />
              </div>
            </template>

            <!-- Usage tab API key filter -->
            <div v-else class="min-w-[180px]">
              <label class="input-label">{{ t('usage.apiKeyFilter') }}</label>
              <Select
                v-model="filters.api_key_id"
                :options="apiKeyOptions"
                :placeholder="t('usage.allApiKeys')"
                @change="applyFilters"
              />
            </div>

            <!-- Date Range Filter (shared by usage + errors) -->
            <div>
              <label class="input-label">{{ t('usage.timeRange') }}</label>
              <DateRangePicker
                v-model:start-date="startDate"
                v-model:end-date="endDate"
                @change="onDateRangeChange"
              />
            </div>

            <!-- Actions -->
            <div class="ml-auto flex items-center gap-3">
              <button
                @click="refreshData"
                :disabled="activeTab === 'errors' ? errorLoading : loading"
                class="btn btn-secondary"
              >
                {{ t('common.refresh') }}
              </button>
              <button @click="resetFilters" class="btn btn-secondary">
                {{ t('common.reset') }}
              </button>
              <div v-if="activeTab === 'errors'" class="relative" ref="errorColumnDropdownRef">
                <button
                  type="button"
                  @click="showErrorColumnDropdown = !showErrorColumnDropdown"
                  class="btn btn-secondary px-2 md:px-3"
                  :title="t('admin.users.columnSettings')"
                >
                  <Icon name="grid" size="sm" class="md:mr-1.5" />
                  <span class="hidden md:inline">{{ t('admin.users.columnSettings') }}</span>
                </button>
                <div
                  v-if="showErrorColumnDropdown"
                  class="absolute right-0 top-full z-50 mt-1 max-h-80 w-48 overflow-y-auto rounded-lg border border-gray-200 bg-white py-1 shadow-lg dark:border-dark-600 dark:bg-dark-800"
                >
                  <button
                    v-for="col in errToggleableColumns"
                    :key="col.key"
                    type="button"
                    @click="toggleErrColumn(col.key)"
                    class="flex w-full items-center justify-between px-4 py-2 text-left text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700"
                  >
                    <span>{{ col.label }}</span>
                    <Icon
                      v-if="isErrColumnVisible(col.key)"
                      name="check"
                      size="sm"
                      class="text-primary-500"
                    />
                  </button>
                </div>
              </div>
              <button
                v-if="activeTab !== 'errors'"
                @click="exportToCSV"
                :disabled="exporting"
                class="btn btn-primary"
              >
                <svg
                  v-if="exporting"
                  class="-ml-1 mr-2 h-4 w-4 animate-spin"
                  fill="none"
                  viewBox="0 0 24 24"
                >
                  <circle
                    class="opacity-25"
                    cx="12"
                    cy="12"
                    r="10"
                    stroke="currentColor"
                    stroke-width="4"
                  ></circle>
                  <path
                    class="opacity-75"
                    fill="currentColor"
                    d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                  ></path>
                </svg>
                {{ exporting ? t('usage.exporting') : t('usage.exportCsv') }}
              </button>
            </div>
          </div>
        </div>
        </div>
      </template>

      <template #table>
        <div
          v-if="errorViewEnabled"
          class="mb-4 flex gap-2 border-b border-gray-200 dark:border-dark-700"
        >
          <button
            type="button"
            class="-mb-px border-b-2 px-3 py-2 text-sm font-medium transition-colors"
            :class="activeTab === 'usage'
              ? 'border-primary-500 text-primary-600 dark:text-primary-400'
              : 'border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200'"
            @click="switchToUsage"
          >
            {{ t('usage.tabs.usage') }}
          </button>
          <button
            type="button"
            class="-mb-px border-b-2 px-3 py-2 text-sm font-medium transition-colors"
            :class="activeTab === 'errors'
              ? 'border-primary-500 text-primary-600 dark:text-primary-400'
              : 'border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200'"
            @click="switchToErrors"
          >
            {{ t('usage.tabs.errors') }}
          </button>
        </div>

        <UserErrorRequestsTable
          v-if="errorViewEnabled && activeTab === 'errors'"
          :rows="errorRows"
          :total="errorTotal"
          :loading="errorLoading"
          :page="errorPage"
          :page-size="errorPageSize"
          :visible-column-keys="errVisibleColumnKeys"
          @sort="onErrorSort"
          @update:page="onErrorPage"
          @update:pageSize="onErrorPageSize"
          @ipGeoBatchFailed="handleIpGeoBatchFailed"
        />

        <DataTable
          v-else
          :columns="columns"
          :data="usageLogs"
          :loading="loading"
          :server-side-sort="true"
          :estimate-row-height="88"
          :overscan="12"
          default-sort-key="created_at"
          default-sort-order="desc"
          @sort="handleSort"
        >
          <template #cell-api_key="{ row }">
            <span class="text-sm text-gray-900 dark:text-white">{{
              row.api_key?.name || '-'
            }}</span>
          </template>

          <template #cell-model="{ row, value }">
            <div
              v-if="row.model_mapping_chain && row.model_mapping_chain.includes('→')"
              class="space-y-0.5 text-xs"
            >
              <div
                v-for="(step, i) in row.model_mapping_chain.split('→')"
                :key="i"
                class="break-all"
                :class="
                  i === 0
                    ? 'font-medium text-gray-900 dark:text-white'
                    : 'text-gray-500 dark:text-gray-400'
                "
                :style="i > 0 ? `padding-left: ${i * 0.75}rem` : ''"
              >
                <span v-if="i > 0" class="mr-0.5">↳</span>{{ step }}
              </div>
            </div>
            <div
              v-else-if="row.upstream_model && row.upstream_model !== row.model"
              class="space-y-0.5 text-xs"
            >
              <div class="break-all font-medium text-gray-900 dark:text-white">
                {{ row.model }}
              </div>
              <div class="break-all text-gray-500 dark:text-gray-400">
                <span class="mr-0.5">↳</span>{{ row.upstream_model }}
              </div>
            </div>
            <span v-else class="font-medium text-gray-900 dark:text-white">{{ value }}</span>
          </template>

          <template #cell-reasoning_effort="{ row }">
            <span class="text-sm text-gray-900 dark:text-white">
              {{ formatReasoningEffort(row.reasoning_effort) }}
            </span>
          </template>

          <template #cell-endpoint="{ row }">
            <span class="text-sm text-gray-600 dark:text-gray-300 block max-w-[320px] whitespace-normal break-all">
              {{ formatUsageEndpoints(row) }}
            </span>
          </template>

          <template #cell-stream="{ row }">
            <span
              class="inline-flex items-center rounded px-2 py-0.5 text-xs font-medium"
              :class="getRequestTypeBadgeClass(row)"
            >
              {{ getRequestTypeLabel(row) }}
            </span>
          </template>

          <template #cell-billing_mode="{ row }">
            <span class="inline-flex items-center rounded px-1.5 py-0.5 text-xs font-medium"
                  :class="getBillingModeBadgeClass(row.billing_mode)">
              {{ getBillingModeLabel(row.billing_mode, t) }}
            </span>
          </template>

          <template #cell-deduction_mode="{ row }">
            <span class="inline-flex items-center rounded px-1.5 py-0.5 text-xs font-medium"
                  :class="getDeductionModeBadgeClass(row.billing_type)">
              {{ getDeductionModeLabel(row.billing_type) }}
            </span>
          </template>

          <template #cell-tokens="{ row }">
            <!-- 图片生成请求（仅按次计费时显示图片格式） -->
            <div v-if="row.image_count > 0 && row.billing_mode === 'image'" class="flex items-center gap-1.5">
              <svg
                class="h-4 w-4 text-indigo-500"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"
                />
              </svg>
              <span class="font-medium text-gray-900 dark:text-white">{{ row.image_count }}{{ $t('usage.imageUnit') }}</span>
              <span class="text-gray-400">({{ row.image_size || '2K' }})</span>
            </div>
            <!-- Token 请求 -->
            <div v-else class="flex items-center gap-1.5">
              <div class="space-y-1.5 text-sm">
                <!-- Input / Output Tokens -->
                <div class="flex items-center gap-2">
                  <!-- Input -->
                  <div class="inline-flex items-center gap-1">
                    <Icon name="arrowDown" size="sm" class="text-emerald-500" />
                    <span class="font-medium text-gray-900 dark:text-white">{{
                      (row.input_tokens ?? 0).toLocaleString()
                    }}</span>
                  </div>
                  <!-- Output -->
                  <div class="inline-flex items-center gap-1">
                    <Icon name="arrowUp" size="sm" class="text-violet-500" />
                    <span class="font-medium text-gray-900 dark:text-white">{{
                      (row.output_tokens ?? 0).toLocaleString()
                    }}</span>
                  </div>
                </div>
                <!-- Cache Tokens (Read + Write) -->
                <div
                  v-if="row.cache_read_tokens > 0 || row.cache_creation_tokens > 0"
                  class="flex items-center gap-2"
                >
                  <!-- Cache Read -->
                  <div v-if="row.cache_read_tokens > 0" class="inline-flex items-center gap-1">
                    <Icon name="inbox" size="sm" class="text-sky-500" />
                    <span class="font-medium text-sky-600 dark:text-sky-400">{{
                      formatCacheTokens(row.cache_read_tokens)
                    }}</span>
                  </div>
                  <!-- Cache Write -->
                  <div v-if="row.cache_creation_tokens > 0" class="inline-flex items-center gap-1">
                    <Icon name="edit" size="sm" class="text-amber-500" />
                    <span class="font-medium text-amber-600 dark:text-amber-400">{{
                      formatCacheTokens(row.cache_creation_tokens)
                    }}</span>
                    <span v-if="row.cache_creation_1h_tokens > 0" class="inline-flex items-center rounded px-1 py-px text-[10px] font-medium leading-tight bg-orange-100 text-orange-600 ring-1 ring-inset ring-orange-200 dark:bg-orange-500/20 dark:text-orange-400 dark:ring-orange-500/30">1h</span>
                    <span v-if="row.cache_ttl_overridden" :title="t('usage.cacheTtlOverriddenHint')" class="inline-flex items-center rounded px-1 py-px text-[10px] font-medium leading-tight bg-rose-100 text-rose-600 ring-1 ring-inset ring-rose-200 dark:bg-rose-500/20 dark:text-rose-400 dark:ring-rose-500/30 cursor-help">R</span>
                  </div>
                </div>
              </div>
              <!-- Token Detail Tooltip -->
              <div
                class="group relative"
                @mouseenter="showTokenTooltip($event, row)"
                @mouseleave="hideTokenTooltip"
              >
                <div
                  class="flex h-4 w-4 cursor-help items-center justify-center rounded-full bg-gray-100 transition-colors group-hover:bg-blue-100 dark:bg-gray-700 dark:group-hover:bg-blue-900/50"
                >
                  <Icon
                    name="infoCircle"
                    size="xs"
                    class="text-gray-400 group-hover:text-blue-500 dark:text-gray-500 dark:group-hover:text-blue-400"
                  />
                </div>
              </div>
            </div>
          </template>

          <template #cell-cost="{ row }">
            <div class="flex items-center gap-1.5 text-sm">
              <span class="font-medium text-green-600 dark:text-green-400">
                ${{ (row.actual_cost ?? 0).toFixed(6) }}
              </span>
              <!-- Cost Detail Tooltip -->
              <div
                class="group relative"
                @mouseenter="showTooltip($event, row)"
                @mouseleave="hideTooltip"
              >
                <div
                  class="flex h-4 w-4 cursor-help items-center justify-center rounded-full bg-gray-100 transition-colors group-hover:bg-blue-100 dark:bg-gray-700 dark:group-hover:bg-blue-900/50"
                >
                  <Icon
                    name="infoCircle"
                    size="xs"
                    class="text-gray-400 group-hover:text-blue-500 dark:text-gray-500 dark:group-hover:text-blue-400"
                  />
                </div>
              </div>
            </div>
          </template>

          <!-- 合并首字/总耗时的健康度列：左侧色条上端随首字档、下端随总耗时档 -->
          <template #cell-latency="{ row }">
            <div class="flex items-stretch gap-2">
              <span
                class="w-1 shrink-0 rounded-full"
                :class="row.first_token_ms != null
                  ? ['bg-gradient-to-b from-40% to-60%', LATENCY_BAR_FROM_CLASSES[firstTokenSeverity(row.first_token_ms)], LATENCY_BAR_TO_CLASSES[durationSeverity(row.duration_ms ?? 0)]]
                  : LATENCY_BAR_CLASSES[durationSeverity(row.duration_ms ?? 0)]"
                aria-hidden="true"
              ></span>
              <div class="grid grid-cols-[max-content_max-content] items-baseline gap-x-2 gap-y-0.5 text-xs">
                <span class="text-gray-400 dark:text-gray-500">{{ t('usage.latencyFirstToken') }}</span>
                <span
                  v-if="row.first_token_ms != null"
                  class="font-medium tabular-nums"
                  :class="LATENCY_TEXT_CLASSES[firstTokenSeverity(row.first_token_ms)]"
                >{{ formatDuration(row.first_token_ms) }}</span>
                <span v-else class="text-gray-400 dark:text-gray-500">-</span>
                <span class="text-gray-400 dark:text-gray-500">{{ t('usage.latencyDuration') }}</span>
                <span
                  class="font-medium tabular-nums"
                  :class="LATENCY_TEXT_CLASSES[durationSeverity(row.duration_ms ?? 0)]"
                >{{ formatDuration(row.duration_ms) }}</span>
              </div>
            </div>
          </template>

          <template #cell-created_at="{ value }">
            <span class="text-sm text-gray-600 dark:text-gray-400">{{
              formatDateTime(value)
            }}</span>
          </template>

          <template #cell-user_agent="{ row }">
            <span v-if="row.user_agent" class="text-sm text-gray-600 dark:text-gray-400 block max-w-[320px] whitespace-normal break-all" :title="row.user_agent">{{ formatUserAgent(row.user_agent) }}</span>
            <span v-else class="text-sm text-gray-400 dark:text-gray-500">-</span>
          </template>

          <template #empty>
            <EmptyState :message="t('usage.noRecords')" />
          </template>
        </DataTable>
      </template>

      <template #pagination>
        <Pagination
          v-if="activeTab === 'usage' && pagination.total > 0"
          :page="pagination.page"
          :total="pagination.total"
          :page-size="pagination.page_size"
          @update:page="handlePageChange"
          @update:pageSize="handlePageSizeChange"
        />
      </template>
    </TablePageLayout>
  </AppLayout>

  <!-- Token Tooltip Portal -->
  <Teleport to="body">
    <div
      v-if="tokenTooltipVisible"
      class="fixed z-[9999] pointer-events-none -translate-y-1/2"
      :style="{
        left: tokenTooltipPosition.x + 'px',
        top: tokenTooltipPosition.y + 'px'
      }"
    >
      <div
        class="whitespace-nowrap rounded-lg border border-gray-700 bg-gray-900 px-3 py-2.5 text-xs text-white shadow-xl dark:border-gray-600 dark:bg-gray-800"
      >
        <div class="space-y-1.5">
          <!-- Token Breakdown -->
          <div>
            <div class="text-xs font-semibold text-gray-300 mb-1">{{ t('usage.tokenDetails') }}</div>
            <div v-if="tokenTooltipData && tokenTooltipData.input_tokens > 0" class="flex items-center justify-between gap-4">
              <span class="text-gray-400">{{ t('admin.usage.inputTokens') }}</span>
              <span class="font-medium text-white">{{ tokenTooltipData.input_tokens.toLocaleString() }}</span>
            </div>
            <div v-if="tokenTooltipData && tokenTooltipData.output_tokens > 0" class="flex items-center justify-between gap-4">
              <span class="text-gray-400">{{ t('admin.usage.outputTokens') }}</span>
              <span class="font-medium text-white">{{ tokenTooltipData.output_tokens.toLocaleString() }}</span>
            </div>
            <div v-if="tokenTooltipData && tokenTooltipData.cache_creation_tokens > 0">
              <!-- 有 5m/1h 明细时，展开显示 -->
              <template v-if="tokenTooltipData.cache_creation_5m_tokens > 0 || tokenTooltipData.cache_creation_1h_tokens > 0">
                <div v-if="tokenTooltipData.cache_creation_5m_tokens > 0" class="flex items-center justify-between gap-4">
                  <span class="text-gray-400 flex items-center gap-1.5">
                    {{ t('admin.usage.cacheCreation5mTokens') }}
                    <span class="inline-flex items-center rounded px-1 py-px text-[10px] font-medium leading-tight bg-amber-500/20 text-amber-400 ring-1 ring-inset ring-amber-500/30">5m</span>
                  </span>
                  <span class="font-medium text-white">{{ tokenTooltipData.cache_creation_5m_tokens.toLocaleString() }}</span>
                </div>
                <div v-if="tokenTooltipData.cache_creation_1h_tokens > 0" class="flex items-center justify-between gap-4">
                  <span class="text-gray-400 flex items-center gap-1.5">
                    {{ t('admin.usage.cacheCreation1hTokens') }}
                    <span class="inline-flex items-center rounded px-1 py-px text-[10px] font-medium leading-tight bg-orange-500/20 text-orange-400 ring-1 ring-inset ring-orange-500/30">1h</span>
                  </span>
                  <span class="font-medium text-white">{{ tokenTooltipData.cache_creation_1h_tokens.toLocaleString() }}</span>
                </div>
              </template>
              <!-- 无明细时，只显示聚合值 -->
              <div v-else class="flex items-center justify-between gap-4">
                <span class="text-gray-400">{{ t('admin.usage.cacheCreationTokens') }}</span>
                <span class="font-medium text-white">{{ tokenTooltipData.cache_creation_tokens.toLocaleString() }}</span>
              </div>
            </div>
            <div v-if="tokenTooltipData && tokenTooltipData.cache_ttl_overridden" class="flex items-center justify-between gap-4">
              <span class="text-gray-400 flex items-center gap-1.5">
                {{ t('usage.cacheTtlOverriddenLabel') }}
                <span class="inline-flex items-center rounded px-1 py-px text-[10px] font-medium leading-tight bg-rose-500/20 text-rose-400 ring-1 ring-inset ring-rose-500/30">R-{{ tokenTooltipData.cache_creation_1h_tokens > 0 ? '5m' : '1H' }}</span>
              </span>
              <span class="font-medium text-rose-400">{{ tokenTooltipData.cache_creation_1h_tokens > 0 ? t('usage.cacheTtlOverridden1h') : t('usage.cacheTtlOverridden5m') }}</span>
            </div>
            <div v-if="tokenTooltipData && tokenTooltipData.cache_read_tokens > 0" class="flex items-center justify-between gap-4">
              <span class="text-gray-400">{{ t('admin.usage.cacheReadTokens') }}</span>
              <span class="font-medium text-white">{{ tokenTooltipData.cache_read_tokens.toLocaleString() }}</span>
            </div>
          </div>
          <!-- Total -->
          <div class="flex items-center justify-between gap-6 border-t border-gray-700 pt-1.5">
            <span class="text-gray-400">{{ t('usage.totalTokens') }}</span>
            <span class="font-semibold text-blue-400">{{ ((tokenTooltipData?.input_tokens || 0) + (tokenTooltipData?.output_tokens || 0) + (tokenTooltipData?.cache_creation_tokens || 0) + (tokenTooltipData?.cache_read_tokens || 0)).toLocaleString() }}</span>
          </div>
        </div>
        <!-- Tooltip Arrow (left side) -->
        <div
          class="absolute right-full top-1/2 h-0 w-0 -translate-y-1/2 border-b-[6px] border-r-[6px] border-t-[6px] border-b-transparent border-r-gray-900 border-t-transparent dark:border-r-gray-800"
        ></div>
      </div>
    </div>
  </Teleport>

  <!-- Tooltip Portal -->
  <Teleport to="body">
    <div
      v-if="tooltipVisible"
      class="fixed z-[9999] pointer-events-none -translate-y-1/2"
      :style="{
        left: tooltipPosition.x + 'px',
        top: tooltipPosition.y + 'px'
      }"
    >
      <div
        class="whitespace-nowrap rounded-lg border border-gray-700 bg-gray-900 px-3 py-2.5 text-xs text-white shadow-xl dark:border-gray-600 dark:bg-gray-800"
      >
        <div class="space-y-1.5">
          <!-- Cost Breakdown -->
          <div class="mb-2 border-b border-gray-700 pb-1.5">
            <div class="text-xs font-semibold text-gray-300 mb-1">{{ t('usage.costDetails') }}</div>
            <div v-if="tooltipData && tooltipData.input_cost > 0" class="flex items-center justify-between gap-4">
              <span class="text-gray-400">{{ t('admin.usage.inputCost') }}</span>
              <span class="font-medium text-white">${{ tooltipData.input_cost.toFixed(6) }}</span>
            </div>
            <div v-if="tooltipData && tooltipData.output_cost > 0" class="flex items-center justify-between gap-4">
              <span class="text-gray-400">{{ t('admin.usage.outputCost') }}</span>
              <span class="font-medium text-white">${{ tooltipData.output_cost.toFixed(6) }}</span>
            </div>
            <!-- Token billing: show unit prices per 1M tokens -->
            <template v-if="tooltipData && !isImageUsage(tooltipData) && (!tooltipData.billing_mode || tooltipData.billing_mode === BILLING_MODE_TOKEN)">
              <div v-if="tooltipData && tooltipData.input_tokens > 0" class="flex items-center justify-between gap-4">
                <span class="text-gray-400">{{ t('usage.inputTokenPrice') }}</span>
                <span class="font-medium text-sky-300">{{ formatTokenPricePerMillion(tooltipData.input_cost, tooltipData.input_tokens) }} {{ t('usage.perMillionTokens') }}</span>
              </div>
              <div v-if="tooltipData && tooltipData.output_tokens > 0" class="flex items-center justify-between gap-4">
                <span class="text-gray-400">{{ t('usage.outputTokenPrice') }}</span>
                <span class="font-medium text-violet-300">{{ formatTokenPricePerMillion(tooltipData.output_cost, tooltipData.output_tokens) }} {{ t('usage.perMillionTokens') }}</span>
              </div>
            </template>
            <!-- Per-request / image billing: show unit price -->
            <template v-else-if="tooltipData?.billing_mode === 'image'">
              <div class="flex items-center justify-between gap-4">
                <span class="text-gray-400">{{ t('usage.imageCount') }}</span>
                <span class="font-medium text-white">{{ tooltipData.image_count }}{{ t('usage.imageUnit') }} ({{ tooltipData.image_size || '2K' }})</span>
              </div>
              <div class="flex items-center justify-between gap-4">
                <span class="text-gray-400">{{ t('usage.imageUnitPrice') }}</span>
                <span class="font-medium text-sky-300">${{ imageUnitPrice(tooltipData).toFixed(6) }}</span>
              </div>
              <div class="flex items-center justify-between gap-4">
                <span class="text-gray-400">{{ t('usage.imageTotalPrice') }}</span>
                <span class="font-medium text-white">${{ tooltipData.total_cost?.toFixed(6) || '0.000000' }}</span>
              </div>
            </template>
            <div v-else class="flex items-center justify-between gap-4">
              <span class="text-gray-400">{{ t('usage.unitPrice') }}</span>
              <span class="font-medium text-sky-300">${{ tooltipData?.total_cost?.toFixed(6) || '0.000000' }}</span>
            </div>
            <div v-if="tooltipData && tooltipData.cache_creation_cost > 0" class="flex items-center justify-between gap-4">
              <span class="text-gray-400">{{ t('admin.usage.cacheCreationCost') }}</span>
              <span class="font-medium text-white">${{ tooltipData.cache_creation_cost.toFixed(6) }}</span>
            </div>
            <div v-if="tooltipData && tooltipData.cache_read_cost > 0" class="flex items-center justify-between gap-4">
              <span class="text-gray-400">{{ t('admin.usage.cacheReadCost') }}</span>
              <span class="font-medium text-white">${{ tooltipData.cache_read_cost.toFixed(6) }}</span>
            </div>
          </div>
          <!-- Rate and Summary -->
          <div class="flex items-center justify-between gap-6">
            <span class="text-gray-400">{{ t('usage.serviceTier') }}</span>
            <span class="font-semibold text-cyan-300">{{ getUsageServiceTierLabel(tooltipData?.service_tier, t) }}</span>
          </div>
          <div class="flex items-center justify-between gap-6">
            <span class="text-gray-400">{{ t('usage.rate') }}</span>
            <span class="font-semibold text-blue-400"
              >{{ formatMultiplier(tooltipData?.rate_multiplier || 1) }}x</span
            >
          </div>
          <div class="flex items-center justify-between gap-6">
            <span class="text-gray-400">{{ t('usage.original') }}</span>
            <span class="font-medium text-white">${{ tooltipData?.total_cost.toFixed(6) }}</span>
          </div>
          <div class="flex items-center justify-between gap-6 border-t border-gray-700 pt-1.5">
            <span class="text-gray-400">{{ t('usage.billed') }}</span>
            <span class="font-semibold text-green-400"
              >${{ tooltipData?.actual_cost.toFixed(6) }}</span
            >
          </div>
        </div>
        <!-- Tooltip Arrow (left side) -->
        <div
          class="absolute right-full top-1/2 h-0 w-0 -translate-y-1/2 border-b-[6px] border-r-[6px] border-t-[6px] border-b-transparent border-r-gray-900 border-t-transparent dark:border-r-gray-800"
        ></div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, reactive, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { usageAPI, keysAPI } from '@/api'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Select from '@/components/common/Select.vue'
import type { SelectOption } from '@/components/common/Select.vue'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import Icon from '@/components/icons/Icon.vue'
import UserErrorRequestsTable from '@/components/user/UserErrorRequestsTable.vue'
import type { UsageLog, ApiKey, UsageQueryParams, UsageStatsResponse, UserErrorRequest } from '@/types'
import type { Column } from '@/components/common/types'
import { formatDateTime, formatReasoningEffort } from '@/utils/format'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'
import { formatCacheTokens, formatMultiplier } from '@/utils/formatters'
import { formatTokenPricePerMillion } from '@/utils/usagePricing'
import { getUsageServiceTierLabel } from '@/utils/usageServiceTier'
import { resolveUsageRequestType } from '@/utils/usageRequestType'
import { BILLING_MODE_TOKEN, getBillingModeLabel, getBillingModeBadgeClass, isImageUsage } from '@/utils/billingMode'
import {
  LATENCY_BAR_CLASSES,
  LATENCY_BAR_FROM_CLASSES,
  LATENCY_BAR_TO_CLASSES,
  LATENCY_TEXT_CLASSES,
  durationSeverity,
  firstTokenSeverity,
} from '@/utils/latencyHealth'
import { COMMON_ERROR_STATUS_CODES } from '@/utils/errorBadges'

const { t } = useI18n()
const appStore = useAppStore()

let abortController: AbortController | null = null
let errorAbortController: AbortController | null = null

// Usage / errors detail tabs (errors gated by public setting)
type DetailTab = 'usage' | 'errors'
const activeTab = ref<DetailTab>('usage')
const errorViewEnabled = computed(
  () => appStore.cachedPublicSettings?.allow_user_view_error_requests ?? false
)

const errorRows = ref<UserErrorRequest[]>([])
const errorLoading = ref(false)
const errorPage = ref(1)
const errorPageSize = ref(20)
const errorTotal = ref(0)
const errorSortBy = ref('created_at')
const errorSortOrder = ref<'asc' | 'desc'>('desc')
const errorFilter = ref<{
  model: string | null
  category: string
  api_key_id: number | null
  status_code: number | null
}>({
  model: '',
  category: '',
  api_key_id: null,
  status_code: null,
})

const errorCategoryCodes = [
  'auth',
  'rate_limit',
  'quota',
  'invalid_request',
  'service_unavailable',
  'upstream',
  'internal',
  'cyber',
]

const errorKeyOptions = computed<SelectOption[]>(() => [
  { value: null, label: t('usage.errors.allKeys') },
  ...apiKeys.value.map((key) => ({ value: key.id, label: key.name })),
])

const errorModelOptions = computed<SelectOption[]>(() => {
  const seen = new Set<string>()
  const opts: SelectOption[] = []
  for (const row of errorRows.value) {
    if (row.model && !seen.has(row.model)) {
      seen.add(row.model)
      opts.push({ value: row.model, label: row.model })
    }
  }
  const current = (errorFilter.value.model ?? '').trim()
  if (current && !seen.has(current)) {
    opts.unshift({ value: current, label: current })
  }
  return opts
})

const errorCategoryOptions = computed<SelectOption[]>(() => [
  { value: '', label: t('usage.errors.allCategories') },
  ...errorCategoryCodes.map((c) => ({ value: c, label: t('usage.errors.categories.' + c) })),
])

const errorStatusOptions = computed<SelectOption[]>(() => [
  { value: null, label: t('usage.errors.allStatuses') },
  ...COMMON_ERROR_STATUS_CODES.map((c) => ({ value: c, label: String(c) })),
])

// 错误请求 tab 独立列设置
const ERR_ALWAYS_VISIBLE = ['status', 'created_at']
const ERR_DEFAULT_HIDDEN_COLUMNS = ['user_agent']
const ERR_HIDDEN_COLUMNS_KEY = 'user-usage-error-hidden-columns'

const errAllColumns = computed<Column[]>(() => [
  { key: 'key_name', label: t('usage.errors.keyName') },
  { key: 'model', label: t('usage.errors.model') },
  { key: 'endpoint', label: t('usage.errors.endpoint') },
  { key: 'client_ip', label: 'IP' },
  { key: 'group', label: t('admin.usage.group') },
  { key: 'type', label: t('usage.type') },
  { key: 'platform', label: t('usage.errors.platform') },
  { key: 'category', label: t('usage.errors.category') },
  { key: 'status', label: t('usage.errors.status') },
  { key: 'message', label: t('usage.errors.message') },
  { key: 'created_at', label: t('usage.errors.time') },
  { key: 'user_agent', label: t('usage.userAgent') },
])

const errHiddenColumns = reactive<Set<string>>(new Set())
const errToggleableColumns = computed(() =>
  errAllColumns.value.filter((col) => !ERR_ALWAYS_VISIBLE.includes(col.key))
)
const errVisibleColumnKeys = computed(() =>
  errAllColumns.value
    .filter((col) => ERR_ALWAYS_VISIBLE.includes(col.key) || !errHiddenColumns.has(col.key))
    .map((col) => col.key)
)
const isErrColumnVisible = (key: string) => !errHiddenColumns.has(key)
const toggleErrColumn = (key: string) => {
  if (errHiddenColumns.has(key)) errHiddenColumns.delete(key)
  else errHiddenColumns.add(key)
  try {
    localStorage.setItem(ERR_HIDDEN_COLUMNS_KEY, JSON.stringify([...errHiddenColumns]))
  } catch (e) {
    console.error('Failed to save error columns:', e)
  }
}
const loadSavedErrColumns = () => {
  try {
    const saved = localStorage.getItem(ERR_HIDDEN_COLUMNS_KEY)
    const values = saved ? (JSON.parse(saved) as string[]) : ERR_DEFAULT_HIDDEN_COLUMNS
    values.forEach((key) => errHiddenColumns.add(key))
  } catch {
    ERR_DEFAULT_HIDDEN_COLUMNS.forEach((key) => errHiddenColumns.add(key))
  }
}

const showErrorColumnDropdown = ref(false)
const errorColumnDropdownRef = ref<HTMLElement | null>(null)
const handleErrorColumnClickOutside = (event: MouseEvent) => {
  if (
    errorColumnDropdownRef.value &&
    !errorColumnDropdownRef.value.contains(event.target as HTMLElement)
  ) {
    showErrorColumnDropdown.value = false
  }
}

// Tooltip state
const tooltipVisible = ref(false)
const tooltipPosition = ref({ x: 0, y: 0 })
const tooltipData = ref<UsageLog | null>(null)

// Token tooltip state
const tokenTooltipVisible = ref(false)
const tokenTooltipPosition = ref({ x: 0, y: 0 })
const tokenTooltipData = ref<UsageLog | null>(null)

// Usage stats from API
const usageStats = ref<UsageStatsResponse | null>(null)

const columns = computed<Column[]>(() => [
  { key: 'api_key', label: t('usage.apiKeyFilter'), sortable: false },
  { key: 'model', label: t('usage.model'), sortable: true },
  { key: 'reasoning_effort', label: t('usage.reasoningEffort'), sortable: false },
  { key: 'endpoint', label: t('usage.endpoint'), sortable: false },
  { key: 'stream', label: t('usage.type'), sortable: false },
  { key: 'billing_mode', label: t('admin.usage.billingMode'), sortable: false },
  { key: 'deduction_mode', label: t('usage.deductionMode'), sortable: false },
  { key: 'tokens', label: t('usage.tokens'), sortable: false },
  { key: 'cost', label: t('usage.cost'), sortable: false },
  { key: 'latency', label: t('usage.latency'), sortable: false },
  { key: 'created_at', label: t('usage.time'), sortable: true },
  { key: 'user_agent', label: t('usage.userAgent'), sortable: false }
])

const usageLogs = ref<UsageLog[]>([])
const apiKeys = ref<ApiKey[]>([])
const loading = ref(false)
const exporting = ref(false)

const apiKeyOptions = computed(() => {
  return [
    { value: null, label: t('usage.allApiKeys') },
    ...apiKeys.value.map((key) => ({
      value: key.id,
      label: key.name
    }))
  ]
})

// Helper function to format date in local timezone
const formatLocalDate = (date: Date): string => {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

// Initialize date range immediately
const now = new Date()
const weekAgo = new Date(now)
weekAgo.setDate(weekAgo.getDate() - 6)

// Date range state
const startDate = ref(formatLocalDate(weekAgo))
const endDate = ref(formatLocalDate(now))

const filters = ref<UsageQueryParams>({
  api_key_id: undefined,
  start_date: undefined,
  end_date: undefined
})

// Initialize filters with date range
filters.value.start_date = startDate.value
filters.value.end_date = endDate.value

// Handle date range change from DateRangePicker
const onDateRangeChange = (range: {
  startDate: string
  endDate: string
  preset: string | null
}) => {
  filters.value.start_date = range.startDate
  filters.value.end_date = range.endDate
  applyFilters()
}

const pagination = reactive({
  page: 1,
  page_size: getPersistedPageSize(),
  total: 0,
  pages: 0
})
const sortState = reactive({
  sort_by: 'created_at',
  sort_order: 'desc' as 'asc' | 'desc'
})

// 超过 1 分钟简化为 "Xm Ys"，免去人工换算（超过 1 小时再进位为 "Xh Ym"）
const formatDuration = (ms: number | null | undefined): string => {
  if (ms == null) return '-'
  if (ms < 1000) return `${ms}ms`
  if (ms < 60_000) return `${(ms / 1000).toFixed(2)}s`
  const totalSec = Math.round(ms / 1000)
  if (totalSec < 3600) return `${Math.floor(totalSec / 60)}m ${totalSec % 60}s`
  return `${Math.floor(totalSec / 3600)}h ${Math.floor((totalSec % 3600) / 60)}m`
}

const imageUnitPrice = (row: UsageLog | null): number => {
  if (!row || row.image_count <= 0) return 0
  const total = row.total_cost ?? 0
  const price = total / row.image_count
  return Number.isFinite(price) ? price : 0
}

const formatUserAgent = (ua: string): string => {
  return ua
}

const getRequestTypeLabel = (log: UsageLog): string => {
  const requestType = resolveUsageRequestType(log)
  if (requestType === 'cyber') return t('usage.cyber')
  if (requestType === 'ws_v2') return t('usage.ws')
  if (requestType === 'stream') return t('usage.stream')
  if (requestType === 'sync') return t('usage.sync')
  return t('usage.unknown')
}

const getRequestTypeBadgeClass = (log: UsageLog): string => {
  const requestType = resolveUsageRequestType(log)
  if (requestType === 'cyber') return 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200'
  if (requestType === 'ws_v2') return 'bg-violet-100 text-violet-800 dark:bg-violet-900 dark:text-violet-200'
  if (requestType === 'stream') return 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200'
  if (requestType === 'sync') return 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200'
  return 'bg-amber-100 text-amber-800 dark:bg-amber-900 dark:text-amber-200'
}

const getDeductionModeLabel = (billingType: number | null | undefined): string => {
  if (billingType === 0) return t('usage.deductionModeBalance')
  if (billingType === 1) return t('usage.deductionModeSubscription')
  if (billingType === 2) return t('usage.deductionModeMixed')
  return '-'
}

const getDeductionModeBadgeClass = (billingType: number | null | undefined): string => {
  if (billingType === 0) return 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900 dark:text-emerald-200'
  if (billingType === 1) return 'bg-indigo-100 text-indigo-800 dark:bg-indigo-900 dark:text-indigo-200'
  if (billingType === 2) return 'bg-amber-100 text-amber-800 dark:bg-amber-900 dark:text-amber-200'
  return 'bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-200'
}


const getRequestTypeExportText = (log: UsageLog): string => {
  const requestType = resolveUsageRequestType(log)
  if (requestType === 'cyber') return 'Cyber'
  if (requestType === 'ws_v2') return 'WS'
  if (requestType === 'stream') return 'Stream'
  if (requestType === 'sync') return 'Sync'
  return 'Unknown'
}

const formatUsageEndpoints = (log: UsageLog): string => {
  const inbound = log.inbound_endpoint?.trim()
  return inbound || '-'
}

const formatUsageModelForExport = (log: UsageLog): string => {
  if (log.model_mapping_chain?.includes('→')) {
    return log.model_mapping_chain
      .split('→')
      .map((step) => step.trim())
      .filter(Boolean)
      .join(' -> ')
  }

  if (log.upstream_model && log.upstream_model !== log.model) {
    return `${log.model} -> ${log.upstream_model}`
  }

  return log.model
}

const formatTokens = (value: number): string => {
  if (value >= 1_000_000_000) {
    return `${(value / 1_000_000_000).toFixed(2)}B`
  } else if (value >= 1_000_000) {
    return `${(value / 1_000_000).toFixed(2)}M`
  } else if (value >= 1_000) {
    return `${(value / 1_000).toFixed(2)}K`
  }
  return value.toLocaleString()
}

type UsageTableQueryParams = UsageQueryParams & {
  sort_by?: string
  sort_order?: 'asc' | 'desc'
}

const buildUsageQueryParams = (page: number, pageSize: number): UsageTableQueryParams => ({
  page,
  page_size: pageSize,
  ...filters.value,
  sort_by: sortState.sort_by,
  sort_order: sortState.sort_order
})

const loadUsageLogs = async () => {
  if (abortController) {
    abortController.abort()
  }
  const currentAbortController = new AbortController()
  abortController = currentAbortController
  const { signal } = currentAbortController
  loading.value = true
  try {
    const response = await usageAPI.query(
      buildUsageQueryParams(pagination.page, pagination.page_size),
      { signal }
    )
    if (signal.aborted) {
      return
    }
    usageLogs.value = response.items
    pagination.total = response.total
    pagination.pages = response.pages
  } catch (error) {
    if (signal.aborted) {
      return
    }
    const abortError = error as { name?: string; code?: string }
    if (abortError?.name === 'AbortError' || abortError?.code === 'ERR_CANCELED') {
      return
    }
    appStore.showError(t('usage.failedToLoad'))
  } finally {
    if (abortController === currentAbortController) {
      loading.value = false
    }
  }
}

const loadApiKeys = async () => {
  try {
    const response = await keysAPI.list(1, 100)
    apiKeys.value = response.items
  } catch (error) {
    console.error('Failed to load API keys:', error)
  }
}

const loadUsageStats = async () => {
  try {
    const apiKeyId = filters.value.api_key_id ? Number(filters.value.api_key_id) : undefined
    const stats = await usageAPI.getStatsByDateRange(
      filters.value.start_date || startDate.value,
      filters.value.end_date || endDate.value,
      apiKeyId
    )
    usageStats.value = stats
  } catch (error) {
    console.error('Failed to load usage stats:', error)
  }
}

const applyFilters = () => {
  pagination.page = 1
  loadUsageLogs()
  loadUsageStats()
  // Date/API-key changes also reset the errors list when that tab is open.
  errorPage.value = 1
  if (activeTab.value === 'errors') {
    void loadErrors()
  } else {
    errorRows.value = []
    errorTotal.value = 0
  }
}

const refreshData = () => {
  loadUsageLogs()
  loadUsageStats()
  if (activeTab.value === 'errors') void loadErrors()
}

const resetFilters = () => {
  filters.value = {
    api_key_id: undefined,
    start_date: undefined,
    end_date: undefined
  }
  // Reset date range to default (last 7 days)
  const now = new Date()
  const weekAgo = new Date(now)
  weekAgo.setDate(weekAgo.getDate() - 6)
  startDate.value = formatLocalDate(weekAgo)
  endDate.value = formatLocalDate(now)
  filters.value.start_date = startDate.value
  filters.value.end_date = endDate.value
  pagination.page = 1
  errorFilter.value = { model: '', category: '', api_key_id: null, status_code: null }
  errorPage.value = 1
  errorSortBy.value = 'created_at'
  errorSortOrder.value = 'desc'
  loadUsageLogs()
  loadUsageStats()
  if (activeTab.value === 'errors') void loadErrors()
}

const loadErrors = async () => {
  if (!errorViewEnabled.value) return
  errorAbortController?.abort()
  const current = new AbortController()
  errorAbortController = current
  errorLoading.value = true
  try {
    const resp = await usageAPI.listMyErrorRequests(
      {
        page: errorPage.value,
        page_size: errorPageSize.value,
        start_date: filters.value.start_date || startDate.value,
        end_date: filters.value.end_date || endDate.value,
        model: (errorFilter.value.model ?? '').trim() || undefined,
        category: errorFilter.value.category || undefined,
        api_key_id: errorFilter.value.api_key_id ?? undefined,
        status_code: errorFilter.value.status_code ?? undefined,
        sort_by: errorSortBy.value,
        sort_order: errorSortOrder.value,
      },
      { signal: current.signal }
    )
    if (current.signal.aborted) return
    errorRows.value = resp.items
    errorTotal.value = resp.total
  } catch (error) {
    if (current.signal.aborted) return
    const abortError = error as { name?: string; code?: string }
    if (abortError?.name === 'AbortError' || abortError?.code === 'ERR_CANCELED') return
    console.error('[UsageView] loadErrors failed:', error)
    appStore.showError(t('usage.errors.failedToLoad'))
  } finally {
    if (errorAbortController === current) errorLoading.value = false
  }
}

const applyErrorFilters = () => {
  errorPage.value = 1
  void loadErrors()
}

const onErrorSort = (sortBy: string, sortOrder: 'asc' | 'desc') => {
  errorSortBy.value = sortBy
  errorSortOrder.value = sortOrder
  errorPage.value = 1
  void loadErrors()
}

const onErrorPage = (page: number) => {
  errorPage.value = page
  void loadErrors()
}

const onErrorPageSize = (pageSize: number) => {
  errorPageSize.value = pageSize
  errorPage.value = 1
  void loadErrors()
}

const handleIpGeoBatchFailed = () => {
  appStore.showError(t('usage.ipGeo.batchFailed'))
}

const switchToUsage = () => {
  activeTab.value = 'usage'
  showErrorColumnDropdown.value = false
}

const switchToErrors = () => {
  activeTab.value = 'errors'
  if (errorRows.value.length === 0) void loadErrors()
}

const handlePageChange = (page: number) => {
  pagination.page = page
  loadUsageLogs()
}

const handlePageSizeChange = (pageSize: number) => {
  pagination.page_size = pageSize
  pagination.page = 1
  loadUsageLogs()
}

const handleSort = (key: string, order: 'asc' | 'desc') => {
  sortState.sort_by = key
  sortState.sort_order = order
  pagination.page = 1
  loadUsageLogs()
}

/**
 * Escape CSV value to prevent injection and handle special characters
 */
const escapeCSVValue = (value: unknown): string => {
  if (value == null) return ''

  const str = String(value)
  const escaped = str.replace(/"/g, '""')

  // Prevent formula injection by prefixing dangerous characters with single quote
  if (/^[=+\-@\t\r]/.test(str)) {
    return `"\'${escaped}"`
  }

  // Escape values containing comma, quote, or newline
  if (/[,"\n\r]/.test(str)) {
    return `"${escaped}"`
  }

  return str
}

const exportToCSV = async () => {
  if (pagination.total === 0) {
    appStore.showWarning(t('usage.noDataToExport'))
    return
  }

  exporting.value = true
  appStore.showInfo(t('usage.preparingExport'))

  try {
    const allLogs: UsageLog[] = []
    const pageSize = 100 // Use a larger page size for export to reduce requests
    const totalRequests = Math.ceil(pagination.total / pageSize)

    for (let page = 1; page <= totalRequests; page++) {
      const response = await usageAPI.query(buildUsageQueryParams(page, pageSize))
      allLogs.push(...response.items)
    }

    if (allLogs.length === 0) {
      appStore.showWarning(t('usage.noDataToExport'))
      return
    }

    const headers = [
      'Time',
      'API Key Name',
      'Model',
      'Reasoning Effort',
      'Inbound Endpoint',
      'Type',
      'Billing Mode',
      'Deduction Mode',
      'Input Tokens',
      'Output Tokens',
      'Cache Read Tokens',
      'Cache Creation Tokens',
      'Rate Multiplier',
      'Billed Cost',
      'Original Cost',
      'First Token (ms)',
      'Duration (ms)',
      'Latency'
    ]
    const rows = allLogs.map((log) =>
      [
        log.created_at,
        log.api_key?.name || '',
        formatUsageModelForExport(log),
        formatReasoningEffort(log.reasoning_effort),
        log.inbound_endpoint || '',
        getRequestTypeExportText(log),
        getBillingModeLabel(log.billing_mode, t),
        getDeductionModeLabel(log.billing_type),
        log.input_tokens,
        log.output_tokens,
        log.cache_read_tokens,
        log.cache_creation_tokens,
        log.rate_multiplier,
        (log.actual_cost ?? 0).toFixed(8),
        (log.total_cost ?? 0).toFixed(8),
        log.first_token_ms ?? '',
        log.duration_ms ?? '',
        // Human-readable latency for spreadsheet scanning; keep raw ms columns above.
        [log.first_token_ms != null ? formatDuration(log.first_token_ms) : '-', formatDuration(log.duration_ms)].join(' / ')
      ].map(escapeCSVValue)
    )

    const csvContent = [
      headers.map(escapeCSVValue).join(','),
      ...rows.map((row) => row.join(','))
    ].join('\n')

    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' })
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `usage_${filters.value.start_date}_to_${filters.value.end_date}.csv`
    link.click()
    window.URL.revokeObjectURL(url)

    appStore.showSuccess(t('usage.exportSuccess'))
  } catch (error) {
    appStore.showError(t('usage.exportFailed'))
    console.error('CSV Export failed:', error)
  } finally {
    exporting.value = false
  }
}

// Tooltip functions
const showTooltip = (event: MouseEvent, row: UsageLog) => {
  const target = event.currentTarget as HTMLElement
  const rect = target.getBoundingClientRect()

  tooltipData.value = row
  // Position to the right of the icon, vertically centered
  tooltipPosition.value.x = rect.right + 8
  tooltipPosition.value.y = rect.top + rect.height / 2
  tooltipVisible.value = true
}

const hideTooltip = () => {
  tooltipVisible.value = false
  tooltipData.value = null
}

// Token tooltip functions
const showTokenTooltip = (event: MouseEvent, row: UsageLog) => {
  const target = event.currentTarget as HTMLElement
  const rect = target.getBoundingClientRect()

  tokenTooltipData.value = row
  tokenTooltipPosition.value.x = rect.right + 8
  tokenTooltipPosition.value.y = rect.top + rect.height / 2
  tokenTooltipVisible.value = true
}

const hideTokenTooltip = () => {
  tokenTooltipVisible.value = false
  tokenTooltipData.value = null
}

onMounted(() => {
  loadSavedErrColumns()
  document.addEventListener('click', handleErrorColumnClickOutside)
  loadApiKeys()
  loadUsageLogs()
  loadUsageStats()
})

onUnmounted(() => {
  abortController?.abort()
  errorAbortController?.abort()
  document.removeEventListener('click', handleErrorColumnClickOutside)
})
</script>
