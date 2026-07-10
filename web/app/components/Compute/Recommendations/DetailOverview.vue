<template>
  <section class="grid grid-cols-1 md:grid-cols-2 gap-4">
    <!-- Workload -->
    <div
      class="panel p-4 cursor-pointer hover:bg-primary/10 transition-colors"
      role="link"
      tabindex="0"
      @click="navigateToWorkload"
      @keydown.enter="navigateToWorkload"
    >
      <dl class="space-y-3">
        <div>
          <dt class="text-sm text-on-surface-secondary mb-0.5">
            Type
          </dt>
          <dd class="m-0 text-sm font-semibold text-on-surface">
            {{ workloadTypeLabel(recommendation.workload_type) }}
          </dd>
        </div>
        <div>
          <dt class="text-sm text-on-surface-secondary mb-0.5">
            Namespace
          </dt>
          <dd class="m-0">
            <NuxtLink
              :to="`/namespaces/${recommendation.namespace}`"
              class="text-sm font-semibold text-on-surface-secondary hover:text-on-surface transition-colors"
              @click.stop
            >
              {{ recommendation.namespace }}
            </NuxtLink>
          </dd>
        </div>
      </dl>
    </div>

    <!-- Analysis -->
    <div class="panel p-4">
      <dl class="space-y-3">
        <div>
          <dt class="text-sm text-on-surface-secondary mb-0.5">
            Mode
          </dt>
          <dd class="m-0 text-sm font-semibold text-on-surface">
            {{ recommendationModeLabel(recommendation.recommendation_mode) }}
          </dd>
        </div>
        <div v-if="recommendation.analysis_time_range">
          <dt class="text-sm text-on-surface-secondary mb-0.5">
            Analysis Period
          </dt>
          <dd class="m-0 text-sm font-semibold text-on-surface">
            {{ formatDuration(recommendation.analysis_time_range) }}
          </dd>
        </div>
      </dl>
    </div>
  </section>
</template>

<script setup lang="ts">
import { recommendationModeLabel } from '#shared/constants/compute/recommendations'
import { workloadTypeLabel } from '#shared/constants/workload'
import type { RecommendationRecord } from '#shared/types/compute'

const props = defineProps<{
  recommendation: RecommendationRecord
}>()

const navigateToWorkload = () => {
  navigateTo(
    `/namespaces/${props.recommendation.namespace}/workloads/${props.recommendation.workload_type}/${props.recommendation.workload_name}`,
  )
}
</script>
