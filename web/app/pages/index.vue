<template>
  <div class="p-8">
    <!-- Hero Section -->
    <div class="mb-8">
      <h1 class="text-4xl font-bold font-heading mb-2">
        Welcome to Mochi
      </h1>
      <p class="text-on-surface-secondary mb-1">
        Kubernetes Resource Optimization Platform
      </p>
      <p
        v-if="data?.cluster_name"
        class="text-sm text-on-surface-muted"
      >
        Cluster: {{ data.cluster_name }}
      </p>
    </div>

    <!-- Quick Stats Cards -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
      <UiStatsCard
        title="Total Namespaces"
        :value="data?.stats.namespaces ?? 0"
        icon="lucide:layers"
        color="text-primary-light"
      />
      <UiStatsCard
        title="Total Workloads"
        :value="data?.stats.workloads ?? 0"
        icon="lucide:server"
        color="text-secondary-light"
      />
      <UiStatsCard
        title="Total Pods"
        :value="data?.stats.pods ?? 0"
        icon="lucide:rocket"
        color="text-tertiary-light"
      />
      <UiStatsCard
        title="System Health"
        :value="data?.stats.health_score ?? 0"
        trailing="%"
        icon="lucide:heart"
        :color="scoreColor(data?.stats.health_score, { midThreshold: 75, highThreshold: 100, type: 'text' })"
      />
    </div>

    <!-- Recent Activity -->
    <HomeRecentActivities :activities="data?.activities" />

    <!-- System Health Indicators -->
    <HomeHealthCard :health-checks="data?.health_checks" />
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
