<template>
  <div class="p-8">
    <!-- Hero Section -->
    <div class="mb-6">
      <h1 class="text-4xl font-bold font-heading mb-2">
        Welcome to Mochi
      </h1>
      <p class="text-sm text-on-surface-muted">
        Cluster: {{ home.cluster_name }}
      </p>
      <HomeHealthStatus :health-checks="home.health_checks" />
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-6">
      <UiStatsCard
        title="Total Nodes"
        :value="home.stats.nodes"
        icon="lucide:cpu"
      />
      <UiStatsCard
        title="Total Namespaces"
        :value="home.stats.namespaces"
        icon="lucide:layers"
      />
      <UiStatsCard
        title="Total Workloads"
        :value="home.stats.workloads"
        icon="lucide:server"
      />
      <UiStatsCard
        title="Total Pods"
        :value="home.stats.pods"
        icon="lucide:rocket"
      />
    </div>

    <HomeRecentActivities :activities="home.activities" />
  </div>
</template>

<script setup lang="ts">
import type { HomeResponse } from '#shared/types/home'

const { data: homeData, error } = await useApiData<HomeResponse>('/api/v1/home')

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

if (!homeData.value) {
  throw createError({
    statusCode: 404,
    message: 'Home data not found',
    fatal: true,
  })
}
const home = homeData.value
</script>
