<template>
  <section v-if="pods && pods.length > 0">
    <div class="flex items-center justify-between flex-wrap gap-4 mb-4">
      <h2 class="text-2xl font-bold font-heading">
        Pods by Usage
      </h2>
      <div class="flex items-center gap-2 flex-wrap">
        <UiSearchableSelect
          v-model="sortMetric"
          :options="UTILIZATION_METRIC_OPTIONS"
          :searchable="false"
          placeholder="Metric"
        />
        <UiSearchableSelect
          v-model="sortResource"
          :options="UTILIZATION_RESOURCE_OPTIONS"
          :searchable="false"
          placeholder="Resource"
        />
      </div>
    </div>
    <div class="panel p-4">
      <div class="overflow-x-auto">
        <table class="w-full">
          <thead>
            <tr class="border-b border-primary/20 text-sm text-on-surface-secondary">
              <th class="text-left py-2 px-4">
                Pod
              </th>
              <th class="text-right py-2 px-4">
                Current
              </th>
              <th class="text-right py-2 px-4">
                P95
              </th>
              <th class="text-right py-2 px-4">
                Mean
              </th>
              <th class="text-right py-2 px-4">
                Max
              </th>
              <th class="text-center py-2 px-4">
                Trend
              </th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-if="filteredPods.length === 0"
              class="border-b border-primary/20 last:border-b-0"
            >
              <td
                colspan="6"
                class="py-8 px-4"
              >
                <div class="text-on-surface-secondary text-center font-medium">
                  <span>No pods found</span>
                </div>
              </td>
            </tr>
            <tr
              v-for="pod in filteredPods"
              :key="pod.pod_uid"
              class="border-b border-primary/10 last:border-b-0"
            >
              <td class="py-3 px-4">
                <span class="text-on-surface font-medium text-sm">{{ pod.pod_name }}</span>
              </td>
              <td class="py-3 px-4 text-right">
                <div class="text-sm">
                  <div class="text-on-surface">
                    {{ formatCPU(pod.utilization.cpu.current) }}
                  </div>
                  <div class="text-on-surface-secondary text-sm">
                    {{ formatBytes(pod.utilization.memory.current) }}
                  </div>
                </div>
              </td>
              <td class="py-3 px-4 text-right">
                <div class="text-sm">
                  <div class="text-on-surface">
                    {{ formatCPU(pod.utilization.cpu.stats.percentile.p95) }}
                  </div>
                  <div class="text-on-surface-secondary text-sm">
                    {{ formatBytes(pod.utilization.memory.stats.percentile.p95) }}
                  </div>
                </div>
              </td>
              <td class="py-3 px-4 text-right">
                <div class="text-sm">
                  <div class="text-on-surface">
                    {{ formatCPU(pod.utilization.cpu.stats.mean) }}
                  </div>
                  <div class="text-on-surface-secondary text-sm">
                    {{ formatBytes(pod.utilization.memory.stats.mean) }}
                  </div>
                </div>
              </td>
              <td class="py-3 px-4 text-right">
                <div class="text-sm">
                  <div class="text-on-surface">
                    {{ formatCPU(pod.utilization.cpu.stats.max) }}
                  </div>
                  <div class="text-on-surface-secondary text-sm">
                    {{ formatBytes(pod.utilization.memory.stats.max) }}
                  </div>
                </div>
              </td>
              <td class="py-3 px-4 text-center">
                <div class="flex flex-col items-center gap-1.5">
                  <div class="inline-flex items-center">
                    <Icon
                      v-if="pod.utilization.cpu.trend.direction === 'increasing'"
                      name="lucide:trending-up"
                      class="text-xs text-error-light"
                    />
                    <Icon
                      v-else-if="pod.utilization.cpu.trend.direction === 'decreasing'"
                      name="lucide:trending-down"
                      class="text-xs text-success-light"
                    />
                    <Icon
                      v-else
                      name="lucide:arrow-right"
                      class="text-xs text-on-surface-secondary"
                    />
                  </div>
                  <div class="inline-flex items-center">
                    <Icon
                      v-if="pod.utilization.memory.trend.direction === 'increasing'"
                      name="lucide:trending-up"
                      class="text-xs text-error-light"
                    />
                    <Icon
                      v-else-if="pod.utilization.memory.trend.direction === 'decreasing'"
                      name="lucide:trending-down"
                      class="text-xs text-success-light"
                    />
                    <Icon
                      v-else
                      name="lucide:arrow-right"
                      class="text-xs text-on-surface-secondary"
                    />
                  </div>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import {
  UTILIZATION_METRIC_OPTIONS,
  UTILIZATION_RESOURCE_OPTIONS,
} from '#shared/constants/compute/utilization'
import { utilizationSortMetricValue } from '#shared/utils/compute/utilization'
import { formatCPU } from '#shared/utils/compute/format'
import type { PodAnalysis } from '#shared/types/compute'

const props = defineProps<{
  pods: PodAnalysis[]
}>()

const sortMetric = ref<string>('p95')
const sortResource = ref<string>('cpu')

const filteredPods = computed(() => {
  return props.pods.slice().sort((a, b) => {
    const aValue = utilizationSortMetricValue(a.utilization, sortMetric.value, sortResource.value)
    const bValue = utilizationSortMetricValue(b.utilization, sortMetric.value, sortResource.value)
    return bValue - aValue
  })
})
</script>
