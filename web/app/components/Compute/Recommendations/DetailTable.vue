<template>
  <div class="glass rounded-xl p-6">
    <h2 class="text-2xl font-bold font-heading mb-4">
      Container Recommendations
    </h2>
    <div
      v-if="!hasRecommendations "
      class="py-8"
    >
      <UiEmptyState
        icon="lucide:inbox"
        title="No recommendations available"
        description="This recommendation does not contain any container recommendations."
      />
    </div>
    <div
      v-else
      class="overflow-x-auto"
    >
      <table class="w-full">
        <thead>
          <tr class="border-b border-primary/20">
            <th class="text-left py-3 px-4 text-sm font-semibold text-on-surface-secondary">
              Container
            </th>
            <th class="text-left py-3 px-4 text-sm font-semibold text-on-surface-secondary">
              CPU
            </th>
            <th class="text-left py-3 px-4 text-sm font-semibold text-on-surface-secondary">
              Memory
            </th>
            <th class="text-center py-3 px-4 text-sm font-semibold text-on-surface-secondary">
              Confidence
            </th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="rec in recommendation.recommendations"
            :key="rec.container_name"
            class="border-b border-primary/10 hover:bg-primary/5 transition-colors"
          >
            <td class="py-4 px-4">
              <span class="font-medium text-on-surface">
                {{ rec.container_name }}
              </span>
            </td>
            <td class="py-4 px-4">
              <div class="space-y-1.5">
                <!-- CPU Request -->
                <div
                  v-if="rec.cpu.current_request || rec.cpu.recommended_request"
                  class="text-sm flex items-center"
                >
                  <span class="text-on-surface-secondary min-w-[60px]">Request:</span>
                  <span class="text-on-surface ml-2">
                    {{ rec.cpu.current_request ?? 'N/A' }}
                  </span>
                  <Icon
                    name="lucide:arrow-right"
                    class="mx-2 text-xs text-on-surface-muted shrink-0"
                  />
                  <span class="font-medium text-primary-light">
                    {{ rec.cpu.recommended_request ?? 'N/A' }}
                  </span>
                  <span
                    v-if="rec.cpu.request_change_percent !== null && rec.cpu.request_change_percent !== undefined"
                    class="ml-2 text-xs text-on-surface-muted"
                  >
                    ({{ formatChangePercent(rec.cpu.request_change_percent) }})
                  </span>
                </div>
                <!-- CPU Limit -->
                <div
                  v-if="rec.cpu.current_limit || rec.cpu.recommended_limit"
                  class="text-sm flex items-center"
                >
                  <span class="text-on-surface-secondary min-w-[60px]">Limit:</span>
                  <span class="text-on-surface ml-2">
                    {{ rec.cpu.current_limit ?? 'N/A' }}
                  </span>
                  <Icon
                    name="lucide:arrow-right"
                    class="mx-2 text-xs text-on-surface-muted shrink-0"
                  />
                  <span class="font-medium text-primary-light">
                    {{ rec.cpu.recommended_limit ?? 'N/A' }}
                  </span>
                  <span
                    v-if="rec.cpu.limit_change_percent !== null && rec.cpu.limit_change_percent !== undefined"
                    class="ml-2 text-xs text-on-surface-muted"
                  >
                    ({{ formatChangePercent(rec.cpu.limit_change_percent) }})
                  </span>
                </div>
              </div>
            </td>
            <td class="py-4 px-4">
              <div class="space-y-1.5">
                <!-- Memory Request -->
                <div
                  v-if="rec.memory.current_request || rec.memory.recommended_request"
                  class="text-sm flex items-center"
                >
                  <span class="text-on-surface-secondary min-w-[60px]">Request:</span>
                  <span class="text-on-surface ml-2">
                    {{ rec.memory.current_request ?? 'N/A' }}
                  </span>
                  <Icon
                    name="lucide:arrow-right"
                    class="mx-2 text-xs text-on-surface-muted shrink-0"
                  />
                  <span class="font-medium text-primary-light">
                    {{ rec.memory.recommended_request ?? 'N/A' }}
                  </span>
                  <span
                    v-if="rec.memory.request_change_percent !== null && rec.memory.request_change_percent !== undefined"
                    class="ml-2 text-xs text-on-surface-muted"
                  >
                    ({{ formatChangePercent(rec.memory.request_change_percent) }})
                  </span>
                </div>
                <!-- Memory Limit -->
                <div
                  v-if="rec.memory.current_limit || rec.memory.recommended_limit"
                  class="text-sm flex items-center"
                >
                  <span class="text-on-surface-secondary min-w-[60px]">Limit:</span>
                  <span class="text-on-surface ml-2">
                    {{ rec.memory.current_limit ?? 'N/A' }}
                  </span>
                  <Icon
                    name="lucide:arrow-right"
                    class="mx-2 text-xs text-on-surface-muted shrink-0"
                  />
                  <span class="font-medium text-primary-light">
                    {{ rec.memory.recommended_limit ?? 'N/A' }}
                  </span>
                  <span
                    v-if="rec.memory.limit_change_percent !== null && rec.memory.limit_change_percent !== undefined"
                    class="ml-2 text-xs text-on-surface-muted"
                  >
                    ({{ formatChangePercent(rec.memory.limit_change_percent) }})
                  </span>
                </div>
              </div>
            </td>
            <td class="py-4 px-4 text-center">
              <span
                class="px-2 py-1 rounded text-sm font-medium"
                :class="scoreBadgeClass(rec.confidence_score)"
              >
                {{ formatPercentage(rec.confidence_score) }}
              </span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import type * as Compute from '#shared/types/compute'

const props = defineProps<{
  recommendation: Compute.RecommendationRecord
}>()

const hasRecommendations = computed(
  () => (props.recommendation?.recommendations?.length ?? 0) > 0,
)
</script>
