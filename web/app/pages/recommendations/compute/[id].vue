<template>
  <div class="p-8">
    <!-- Header -->
    <div class="mb-8">
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
          to="/recommendations/compute"
          class="text-primary-light hover:text-primary transition-colors"
        >
          Recommendations
        </NuxtLink>
        <Icon
          name="lucide:chevron-right"
          class="text-xs text-on-surface-muted"
        />
        <NuxtLink
          to="/recommendations/compute"
          class="text-primary-light hover:text-primary transition-colors"
        >
          Compute
        </NuxtLink>
        <Icon
          name="lucide:chevron-right"
          class="text-xs text-on-surface-muted"
        />
        <span class="text-on-surface">Recommendation #{{ id }}</span>
      </div>

      <div class="flex items-center justify-between mb-4">
        <div>
          <h1 class="text-4xl font-bold font-heading mb-2">
            Recommendation Details
          </h1>
          <div class="flex items-center space-x-2 flex-wrap">
            <!-- Workload Link -->
            <NuxtLink
              v-if="data"
              :to="`/namespaces/${data.namespace}/workloads/${data.workload_type}/${data.workload_name}`"
              class="px-3 py-1 rounded-full text-xs font-medium bg-primary/20 text-primary-light border border-primary/30 hover:bg-primary/30 transition-colors"
            >
              {{ data.workload_name }}
            </NuxtLink>
            <!-- Status Badge -->
            <span
              :class="recommendationStatusBadgeClass(data?.status ?? '')"
              class="px-3 py-1 rounded-full text-xs font-medium border"
            >
              {{ recommendationStatusLabel(data?.status ?? '') }}
            </span>
            <!-- Created -->
            <span
              v-if="data?.created_at"
              class="px-3 py-1 rounded-full text-xs font-medium bg-primary/20 text-primary-light border border-primary/30"
            >
              Created {{ timeAgo(data.created_at) }}
            </span>
          </div>
        </div>
      </div>
    </div>

    <!-- Loading State -->
    <div
      v-if="pending && !error"
      class="py-12 flex flex-col items-center justify-center gap-3 text-on-surface-secondary"
    >
      <Icon
        name="lucide:loader-circle"
        class="text-3xl animate-spin"
      />
      <p class="text-sm font-medium">
        Loading recommendation...
      </p>
    </div>

    <!-- Content -->
    <div
      v-else-if="data && !error"
      class="space-y-6"
    >
      <!-- Overview Card -->
      <ComputeRecommendationsDetailOverview :recommendation="data" />

      <!-- Container Recommendations Table -->
      <ComputeRecommendationsDetailTable :recommendations="data.recommendations" />

      <!-- Actions -->
      <ComputeRecommendationsDetailActions
        :recommendation-id="data.id"
        :status="data.status"
        :recommendations="data.recommendations"
        @applied="onApplied"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { recommendationStatusLabel } from '#shared/constants/compute/recommendations'
import { recommendationStatusBadgeClass } from '#shared/utils/compute/color'
import type { RecommendationRecord } from '#shared/types/compute'

const route = useRoute()
const id = route.params.id as string

const { data, error, pending, refresh } = await useApiData<RecommendationRecord>(
  `/api/v1/compute/recommendations/${id}`,
)

const { parseError } = useApiError()
if (error.value) {
  const errorInfo = parseError(error.value, 'Failed to load recommendation')
  throw createError({
    status: errorInfo.status,
    statusText: errorInfo.statusText,
    message: errorInfo.message,
    fatal: true,
  })
}

const onApplied = async () => {
  await refresh()
}
</script>
