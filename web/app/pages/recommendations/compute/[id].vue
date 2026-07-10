<template>
  <div class="p-8">
    <!-- Header -->
    <div class="mb-6">
      <UiBreadcrumb :items="breadcrumbs" />
      <h1 class="text-4xl font-bold font-heading mb-2">
        {{ data?.workload_name ?? 'Recommendation Details' }}
      </h1>
      <p
        v-if="data"
        class="text-sm text-on-surface-muted flex items-center gap-1.5 flex-wrap"
      >
        <span
          class="font-medium"
          :class="recommendationStatusTextClass(data.status)"
        >
          {{ recommendationStatusLabel(data.status) }}
        </span>
        <span aria-hidden="true">·</span>
        <span>{{ timestamp }}</span>
      </p>
    </div>

    <!-- Loading State -->
    <div
      v-if="pending && !error"
      class="py-6 flex flex-col items-center justify-center gap-2 text-on-surface-secondary"
    >
      <Icon
        name="lucide:loader-circle"
        class="text-2xl animate-spin"
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

const breadcrumbs = computed(() => [
  { label: 'Home', to: '/' },
  { label: 'Recommendations', to: '/recommendations/compute' },
  { label: 'Compute', to: '/recommendations/compute' },
  { label: `Recommendation #${id}` },
])

const { data, error, pending } = await useApiData<RecommendationRecord>(
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

const timestamp = computed(() => {
  if (!data.value) {
    return undefined
  }

  const { created_at, updated_at } = data.value

  if (timestampsDiffer(created_at, updated_at)) {
    return `Updated ${timeAgo(updated_at!)}`
  }

  return `Created ${timeAgo(created_at)}`
})
</script>
