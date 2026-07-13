<template>
  <div class="p-8">
    <!-- Header -->
    <div class="mb-6">
      <UiBreadcrumb :items="breadcrumbs" />
      <h1 class="text-4xl font-bold font-heading mb-2">
        {{ recommendation.workload_name }}
      </h1>
      <p class="text-sm text-on-surface-muted flex items-center gap-1.5 flex-wrap">
        <span
          class="font-medium"
          :class="recommendationStatusTextClass(recommendation.status)"
        >
          {{ recommendationStatusLabel(recommendation.status) }}
        </span>
        <span aria-hidden="true">·</span>
        <span>{{ timestamp }}</span>
      </p>
    </div>

    <div class="space-y-6">
      <ComputeRecommendationsDetailOverview :recommendation="recommendation" />
      <ComputeRecommendationsDetailTable :recommendations="recommendation.recommendations" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { recommendationStatusLabel } from '#shared/constants/compute/recommendations'
import { recommendationStatusTextClass } from '#shared/utils/compute/color'
import { timestampsDiffer, timeAgo } from '#shared/utils/time'
import type { RecommendationRecord } from '#shared/types/compute'

const route = useRoute()
const id = route.params.id as string

const { data: recommendationData, error } = await useApiData<RecommendationRecord>(
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

if (!recommendationData.value) {
  throw createError({
    statusCode: 404,
    message: 'Recommendation not found',
    fatal: true,
  })
}
const recommendation = recommendationData.value

const breadcrumbs = [
  { label: 'Home', to: '/' },
  { label: 'Recommendations', to: '/recommendations/compute' },
  { label: 'Compute', to: '/recommendations/compute' },
  { label: `Recommendation #${recommendation.id}` },
]

const timestamp = timestampsDiffer(
  recommendation.created_at, recommendation.updated_at)
  ? `Updated ${timeAgo(recommendation.updated_at)}`
  : `Created ${timeAgo(recommendation.created_at)}`
</script>
