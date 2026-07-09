<template>
  <div class="p-8">
    <!-- Hero Section -->
    <div class="mb-6">
      <h1 class="text-4xl font-bold font-heading mb-2">
        Welcome to Mochi
      </h1>
      <p
        v-if="data?.cluster_name"
        class="text-sm text-on-surface-muted"
      >
        Cluster: {{ data.cluster_name }}
      </p>
      <HomeHealthStatus :health-checks="data?.health_checks" />
    </div>

    <!-- Quick Stats Cards -->
    <div class="grid grid-cols-1 md:grid-cols-3 gap-6 mb-6">
      <UiStatsCard
        title="Total Namespaces"
        :value="data?.stats.namespaces ?? 0"
        icon="lucide:layers"
      />
      <UiStatsCard
        title="Total Workloads"
        :value="data?.stats.workloads ?? 0"
        icon="lucide:server"
      />
      <UiStatsCard
        title="Total Pods"
        :value="data?.stats.pods ?? 0"
        icon="lucide:rocket"
      />
    </div>

    <HomeRecentActivities :activities="data?.activities" />
  </div>
</template>

<script setup lang="ts">
import type { HomeResponse } from '#shared/types/home'

const { data, error } = await useApiData<HomeResponse>('/api/v1/home')

const { parseError } = useApiError()
if (error.value) {
  const errorInfo = parseError(error.value, 'Failed to load home data')
  throw createError({
    status: errorInfo.status,
    statusText: errorInfo.statusText,
    message: errorInfo.message,
    fatal: true,
  })
}
</script>
