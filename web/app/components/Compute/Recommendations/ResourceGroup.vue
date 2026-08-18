<template>
  <div class="space-y-1.5">
    <div
      v-for="field in fields"
      :key="field.label"
      class="text-sm flex items-center"
    >
      <span class="text-on-surface-secondary min-w-15">{{ field.label }}:</span>

      <template v-if="field.state === 'empty'">
        <span class="text-on-surface-muted ml-2">N/A</span>
      </template>

      <template v-else-if="field.state === 'unchanged'">
        <span class="text-on-surface-secondary ml-2">
          {{ field.current ?? 'N/A' }}
        </span>
      </template>

      <template v-else>
        <span class="text-on-surface ml-2">
          {{ field.current ?? 'N/A' }}
        </span>
        <Icon
          name="lucide:arrow-right"
          class="mx-2 text-xs text-on-surface-muted shrink-0"
        />
        <span class="font-medium text-primary-light">
          {{ field.recommended ?? 'N/A' }}
        </span>
        <span
          v-if="field.changePercent != null"
          class="ml-2 text-xs text-on-surface-muted"
        >
          ({{ formatChangePercent(field.changePercent) }})
        </span>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import {
  recommendationFieldState,
} from '#shared/utils/compute/recommendations'
import type { CPURecommendation, MemoryRecommendation } from '#shared/types/compute'

const props = defineProps<{
  resource: CPURecommendation | MemoryRecommendation
}>()

const fields = computed(() => {
  return [
    {
      label: 'Request',
      current: props.resource.current_request,
      recommended: props.resource.recommended_request,
      changePercent: props.resource.request_change_percent,
      state: recommendationFieldState(
        props.resource.current_request,
        props.resource.recommended_request,
        props.resource.request_change_percent,
      ),
    },
    {
      label: 'Limit',
      current: props.resource.current_limit,
      recommended: props.resource.recommended_limit,
      changePercent: props.resource.limit_change_percent,
      state: recommendationFieldState(
        props.resource.current_limit,
        props.resource.recommended_limit,
        props.resource.limit_change_percent,
      ),
    },
  ]
})
</script>
