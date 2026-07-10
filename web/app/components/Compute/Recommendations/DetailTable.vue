<template>
  <section>
    <h2 class="text-2xl font-bold font-heading mb-1">
      Container Recommendations
    </h2>
    <p class="text-xs text-on-surface-secondary mb-4">
      Values reflect the resources at recommendation time, not live settings.
    </p>
    <div class="panel p-4">
      <UiEmptyState
        v-if="!hasRecommendations"
        icon="lucide:inbox"
        title="No recommendations available"
        description="This recommendation does not contain any container recommendations."
      />
      <div
        v-else
        class="overflow-x-auto"
      >
        <table class="w-full">
          <thead>
            <tr class="border-b border-primary/20 text-sm text-on-surface-secondary">
              <th class="text-left py-3 px-4">
                Container
              </th>
              <th class="text-left py-3 px-4">
                CPU
              </th>
              <th class="text-left py-3 px-4">
                Memory
              </th>
              <th class="text-center py-3 px-4">
                Confidence
              </th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="rec in recommendations"
              :key="rec.container_name"
              class="border-b border-primary/10 last:border-b-0"
            >
              <td class="py-3 px-4">
                <span class="font-medium text-on-surface">
                  {{ rec.container_name }}
                </span>
              </td>
              <td class="py-3 px-4">
                <ComputeRecommendationsResourceGroup :resource="rec.cpu" />
              </td>
              <td class="py-3 px-4">
                <ComputeRecommendationsResourceGroup :resource="rec.memory" />
              </td>
              <td class="py-3 px-4 text-center">
                <span
                  class="px-2 py-1 rounded-full text-sm font-medium border"
                  :class="scoreBadgeClass(rec.confidence)"
                >
                  {{ formatPercentage(rec.confidence) }}
                </span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import type { ContainerRecommendation } from '#shared/types/compute'

const props = defineProps<{
  recommendations: ContainerRecommendation[]
}>()

const hasRecommendations = computed(
  () => (props.recommendations?.length ?? 0) > 0,
)
</script>
