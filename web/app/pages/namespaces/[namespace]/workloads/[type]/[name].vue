<template>
  <div class="p-8">
    <!-- Header -->
    <div class="mb-6">
      <UiBreadcrumb :items="breadcrumbs" />
      <h1 class="text-4xl font-bold font-heading mb-2">
        {{ data?.name }}
      </h1>
      <p
        v-if="headerSubline"
        class="text-sm text-on-surface-muted"
      >
        {{ headerSubline }}
      </p>
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

const breadcrumbs = computed(() => [
  { label: 'Home', to: '/' },
  { label: String(namespace), to: `/namespaces/${namespace}` },
  { label: data.value?.name ?? String(name) },
])

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

const headerSubline = computed(() => {
  if (!data.value) {
    return undefined
  }

  const parts = [workloadTypeLabel(data.value.type)]

  if (data.value.type !== 'Pod') {
    parts.push(`${data.value.ready}/${data.value.replicas} replicas`)
  }

  parts.push(`Created ${timeAgo(data.value.created_at)}`)

  return parts.join(' · ')
})
</script>
