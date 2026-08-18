<template>
  <div class="p-8">
    <!-- Header -->
    <div class="mb-6">
      <UiBreadcrumb
        :items="breadcrumbs"
      />
      <h1 class="text-4xl font-bold font-heading mb-2">
        {{ workload.name }}
      </h1>
      <p class="text-sm text-on-surface-muted">
        {{ headerSubline }}
      </p>
    </div>

    <UiTabs
      v-model="activeTab"
      :tabs="tabs"
    >
      <template #overview>
        <WorkloadOverviewTab :workload="workload" />
      </template>
      <template #compute>
        <WorkloadComputeTab
          :namespace="workload.namespace"
          :workload-type="workload.type"
          :workload-name="workload.name"
          :is-active="activeTab === 'compute'"
        />
      </template>
    </UiTabs>
  </div>
</template>

<script setup lang="ts">
import { workloadTypeLabel, workloadStatusLine } from '#shared/constants/workload'
import type { WorkloadResponse } from '#shared/types/workload'

const route = useRoute()
const { namespace, type, name } = route.params

const { data: workloadData, error } = await useApiData<WorkloadResponse>(
  `/api/v1/workloads/${namespace}/${type}/${name}`,
)

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

if (!workloadData.value) {
  throw createError({
    statusCode: 404,
    message: 'Workload not found',
    fatal: true,
  })
}
const workload = workloadData.value

const breadcrumbs = [
  { label: 'Home', to: '/' },
  { label: workload.namespace, to: `/namespaces/${workload.namespace}` },
  { label: workload.name },
]

const headerSubline = [
  workloadTypeLabel(workload.type),
  workloadStatusLine(workload),
  `Created ${timeAgo(workload.created_at)}`,
].join(' · ')

const tabs = [
  { id: 'overview', label: 'Overview' },
  { id: 'compute', label: 'Compute' },
]
const activeTab = ref('overview')
</script>
