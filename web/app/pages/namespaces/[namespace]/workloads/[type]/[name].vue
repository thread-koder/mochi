<template>
  <div class="p-8">
    <!-- Header -->
    <div class="mb-4">
      <!-- Breadcrumb -->
      <div class="flex items-center space-x-2 text-sm mb-4">
        <NuxtLink
          to="/"
          class="text-primary-light hover:text-primary transition-colors"
        >
          Home
        </NuxtLink>
        <Icon
          name="lucide:chevron-right"
          class="text-xs text-on-surface-muted"
        />
        <NuxtLink
          :to="`/namespaces/${namespace}`"
          class="text-primary-light hover:text-primary transition-colors"
        >
          {{ namespace }}
        </NuxtLink>
        <Icon
          name="lucide:chevron-right"
          class="text-xs text-on-surface-muted"
        />
        <span class="text-on-surface">{{ data?.name }}</span>
      </div>

      <div class="flex items-center justify-between mb-4">
        <div>
          <h1 class="text-4xl font-bold font-heading mb-2">
            {{ data?.name }}
          </h1>
          <div class="flex items-center space-x-2 flex-wrap">
            <span class="px-3 py-1 rounded-full text-xs font-medium bg-primary/20 text-primary-light border border-primary/30">
              {{ workloadTypeLabel(data?.type ?? '') }}
            </span>
            <span
              v-if="data?.type !== 'Pod'"
              class="px-3 py-1 rounded-full text-xs font-medium bg-primary/20 text-primary-light border border-primary/30"
            >
              {{ data?.ready }}/{{ data?.replicas }} Replicas
            </span>
            <span class="px-3 py-1 rounded-full text-xs font-medium bg-primary/20 text-primary-light border border-primary/30">
              Created {{ timeAgo(data?.created_at ?? '') }}
            </span>
          </div>
        </div>
      </div>
    </div>

    <UiTabs
      v-model="activeTab"
      :tabs="tabs"
    >
      <template #overview>
        <WorkloadOverviewTab :workload-data="data" />
      </template>
      <template #compute>
        <WorkloadComputeTab
          :namespace="data?.namespace ?? ''"
          :workload-type="data?.type ?? ''"
          :workload-name="data?.name ?? ''"
          :is-active="activeTab === 'compute'"
        />
      </template>
    </UiTabs>
  </div>
</template>

<script setup lang="ts">
import { workloadTypeLabel } from '#shared/constants/workload'
import type { WorkloadResponse } from '#shared/types/workload'

const route = useRoute()
const { namespace, type, name } = route.params

const tabs = [
  { id: 'overview', label: 'Overview' },
  { id: 'compute', label: 'Compute' },
]
const activeTab = ref('overview')

const { data, error } = await useApiData<WorkloadResponse>(`/api/v1/workloads/${namespace}/${type}/${name}`)

const { parseError } = useApiError()
if (error.value) {
  const errorInfo = parseError(error.value, 'Failed to load workload data')
  throw createError({
    status: errorInfo.status,
    statusText: errorInfo.statusText,
    message: errorInfo.message,
    fatal: true,
  })
}
</script>
