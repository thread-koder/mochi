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
              v-for="pod in filteredPods"
              :key="pod.pod_uid"
              class="border-b border-primary/10 last:border-b-0"
            >
              <td class="py-3 px-4">
                <span class="text-on-surface font-medium text-sm">{{ pod.pod_name }}</span>
              </td>
              <td class="py-3 px-4 text-right">
                <div class="text-sm">
                  <div :class="utilizationMetricClass(pod.utilization.cpu.sample_size)">
                    {{ formatUtilizationCPU(pod.utilization.cpu.current, pod.utilization.cpu.sample_size) }}
                  </div>
                  <div
                    class="text-sm"
                    :class="utilizationMetricClass(pod.utilization.memory.sample_size, 'secondary')"
                  >
                    {{ formatUtilizationBytes(pod.utilization.memory.current, pod.utilization.memory.sample_size) }}
                  </div>
                </div>
              </td>
              <td class="py-3 px-4 text-right">
                <div class="text-sm">
                  <div :class="utilizationMetricClass(pod.utilization.cpu.sample_size)">
                    {{ formatUtilizationCPU(pod.utilization.cpu.stats.percentile.p95, pod.utilization.cpu.sample_size) }}
                  </div>
                  <div
                    class="text-sm"
                    :class="utilizationMetricClass(pod.utilization.memory.sample_size, 'secondary')"
                  >
                    {{ formatUtilizationBytes(pod.utilization.memory.stats.percentile.p95, pod.utilization.memory.sample_size) }}
                  </div>
                </div>
              </td>
              <td class="py-3 px-4 text-right">
                <div class="text-sm">
                  <div :class="utilizationMetricClass(pod.utilization.cpu.sample_size)">
                    {{ formatUtilizationCPU(pod.utilization.cpu.stats.mean, pod.utilization.cpu.sample_size) }}
                  </div>
                  <div
                    class="text-sm"
                    :class="utilizationMetricClass(pod.utilization.memory.sample_size, 'secondary')"
                  >
                    {{ formatUtilizationBytes(pod.utilization.memory.stats.mean, pod.utilization.memory.sample_size) }}
                  </div>
                </div>
              </td>
              <td class="py-3 px-4 text-right">
                <div class="text-sm">
                  <div :class="utilizationMetricClass(pod.utilization.cpu.sample_size)">
                    {{ formatUtilizationCPU(pod.utilization.cpu.stats.max, pod.utilization.cpu.sample_size) }}
                  </div>
                  <div
                    class="text-sm"
                    :class="utilizationMetricClass(pod.utilization.memory.sample_size, 'secondary')"
                  >
                    {{ formatUtilizationBytes(pod.utilization.memory.stats.max, pod.utilization.memory.sample_size) }}
                  </div>
                </div>
              </td>
              <td class="py-3 px-4 text-center">
                <div class="flex flex-col items-center gap-1.5">
                  <div class="inline-flex items-center">
                    <UiTrendIcon
                      :direction="pod.utilization.cpu.trend.direction"
                      :available="hasEnoughPoints(pod.utilization.cpu.sample_size)"
                    />
                  </div>
                  <div class="inline-flex items-center">
                    <UiTrendIcon
                      :direction="pod.utilization.memory.trend.direction"
                      :available="hasEnoughPoints(pod.utilization.memory.sample_size)"
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
import {
  formatUtilizationBytes,
  formatUtilizationCPU,
  utilizationMetricClass,
  utilizationSortMetricValue,
} from '#shared/utils/compute/utilization'
import { hasEnoughPoints } from '#shared/utils/timeseries'
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
