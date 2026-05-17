<template>
  <div
    v-if="filteredContainers.length > 0"
    class="glass rounded-xl p-6"
  >
    <div class="flex items-center justify-between mb-4">
      <h2 class="text-2xl font-bold font-heading">
        Containers by Usage
      </h2>
      <div class="flex items-center gap-2">
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
    <div class="overflow-x-auto">
      <table class="w-full">
        <thead>
          <tr class="border-b border-primary/20 text-sm text-on-surface-secondary">
            <th class="text-left py-2 px-4">
              Container
            </th>
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
            <th class="text-center py-2 px-4">
              Stability
            </th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-if="filteredContainers.length === 0"
            class="border-b border-primary/20"
          >
            <td
              colspan="8"
              class="py-8 px-4"
            >
              <div class="text-on-surface-secondary text-center font-medium">
                <span>No containers found</span>
              </div>
            </td>
          </tr>
          <template
            v-for="container in filteredContainers"
            :key="`${container.pod_name}-${container.container_name}`"
          >
            <tr
              class="border-b border-primary/20 hover:bg-primary/10 transition-all cursor-pointer"
              @click="toggleExpand(`${container.pod_name}-${container.container_name}`)"
            >
              <td class="py-3 px-4">
                <span class="text-primary-light font-medium text-sm">{{ container.container_name }}</span>
              </td>
              <td class="py-3 px-4">
                <span class="text-on-surface-secondary text-sm">{{ container.pod_name }}</span>
              </td>
              <td class="py-3 px-4 text-right">
                <div class="text-sm">
                  <div class="text-primary-light">
                    {{ formatCPU(container.utilization.cpu.current) }}
                  </div>
                  <div class="text-secondary-light text-sm">
                    {{ formatBytes(container.utilization.memory.current) }}
                  </div>
                </div>
              </td>
              <td class="py-3 px-4 text-right">
                <div class="text-sm">
                  <div class="text-primary-light">
                    {{ formatCPU(container.utilization.cpu.stats.percentile.p95) }}
                  </div>
                  <div class="text-secondary-light text-sm">
                    {{ formatBytes(container.utilization.memory.stats.percentile.p95) }}
                  </div>
                </div>
              </td>
              <td class="py-3 px-4 text-right">
                <div class="text-sm">
                  <div class="text-primary-light">
                    {{ formatCPU(container.utilization.cpu.stats.mean) }}
                  </div>
                  <div class="text-secondary-light text-sm">
                    {{ formatBytes(container.utilization.memory.stats.mean) }}
                  </div>
                </div>
              </td>
              <td class="py-3 px-4 text-right">
                <div class="text-sm">
                  <div class="text-primary-light">
                    {{ formatCPU(container.utilization.cpu.stats.max) }}
                  </div>
                  <div class="text-secondary-light text-sm">
                    {{ formatBytes(container.utilization.memory.stats.max) }}
                  </div>
                </div>
              </td>
              <td class="py-3 px-4 text-center">
                <div class="flex flex-col items-center gap-1.5">
                  <div class="inline-flex items-center">
                    <Icon
                      v-if="container.utilization.cpu.trend.direction === 'increasing'"
                      name="lucide:trending-up"
                      class="text-xs text-error-light"
                    />
                    <Icon
                      v-else-if="container.utilization.cpu.trend.direction === 'decreasing'"
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
                      v-if="container.utilization.memory.trend.direction === 'increasing'"
                      name="lucide:trending-up"
                      class="text-xs text-error-light"
                    />
                    <Icon
                      v-else-if="container.utilization.memory.trend.direction === 'decreasing'"
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
              <td class="py-3 px-4 text-center">
                <div class="flex flex-col items-center gap-1">
                  <span
                    class="text-xs px-2 py-1 rounded-full font-medium border"
                    :class="scoreBadgeClass(container.stability?.stability_score)"
                  >
                    {{ formatPercentage(container.stability?.stability_score ?? 0) }}
                  </span>
                  <div
                    v-if="(container.stability?.restarts ?? 0) > 0 || (container.stability?.memory_oom ?? 0) > 0"
                    class="flex items-center gap-1 text-xs text-error-light"
                  >
                    <Icon
                      name="lucide:triangle-alert"
                      class="text-xs"
                    />
                    <span>Issues</span>
                  </div>
                </div>
              </td>
            </tr>
            <!-- Container details sub-row -->
            <Transition
              enter-active-class="transition ease-out duration-200"
              enter-from-class="opacity-0 -translate-y-2"
              enter-to-class="opacity-100 translate-y-0"
              leave-active-class="transition ease-in duration-150"
              leave-from-class="opacity-100 translate-y-0"
              leave-to-class="opacity-0 -translate-y-2"
            >
              <tr
                v-if="expandedRows.has(`${container.pod_name}-${container.container_name}`)"
                class="border-b border-primary/20 bg-surface-elevated/50"
              >
                <td
                  colspan="8"
                  class="py-4 px-4"
                >
                  <div class="grid grid-cols-3 gap-6">
                    <!-- CPU Provisioning -->
                    <div class="space-y-3">
                      <h4 class="text-sm font-semibold text-primary-light mb-3">
                        CPU Provisioning
                      </h4>
                      <div class="space-y-2">
                        <!-- Efficiency -->
                        <div>
                          <div class="flex items-center justify-between mb-1">
                            <span class="text-xs text-on-surface-secondary">Efficiency</span>
                            <span
                              class="text-xs font-medium"
                              :class="scoreColor(container.provisioning.cpu.efficiency, { midThreshold: 0.5, type: 'text' })"
                            >
                              {{ formatPercentage(container.provisioning.cpu.efficiency) }}
                            </span>
                          </div>
                          <div class="w-full bg-surface-elevated rounded-full h-2">
                            <div
                              class="h-2 rounded-full transition-all"
                              :class="scoreColor(container.provisioning.cpu.efficiency, { midThreshold: 0.5, type: 'bg' })"
                              :style="{ width: `${container.provisioning.cpu.efficiency * 100}%` }"
                            />
                          </div>
                        </div>
                        <!-- Current Request -->
                        <div class="flex items-center justify-between">
                          <span class="text-xs text-on-surface-secondary">Current Request</span>
                          <span class="text-xs font-medium text-on-surface">
                            {{ formatCPU(container.provisioning.cpu.current_request ?? undefined) }}
                          </span>
                        </div>
                        <!-- Request Utilization -->
                        <div class="flex items-center justify-between">
                          <span class="text-xs text-on-surface-secondary">Request Utilization</span>
                          <span class="text-xs font-medium text-on-surface">
                            {{ formatPercentage(container.provisioning.cpu.request_utilization) }}
                          </span>
                        </div>
                        <!-- Current Limit -->
                        <div class="flex items-center justify-between">
                          <span class="text-xs text-on-surface-secondary">Current Limit</span>
                          <span class="text-xs font-medium text-on-surface">
                            {{ formatCPU(container.provisioning.cpu.current_limit ?? undefined) }}
                          </span>
                        </div>
                        <!-- Limit Utilization -->
                        <div class="flex items-center justify-between">
                          <span class="text-xs text-on-surface-secondary">Limit Utilization</span>
                          <span class="text-xs font-medium text-on-surface">
                            {{ formatPercentage(container.provisioning.cpu.limit_utilization) }}
                          </span>
                        </div>
                        <!-- Status Badge -->
                        <div class="flex items-center justify-between pt-1">
                          <span class="text-xs text-on-surface-secondary">Status</span>
                          <span
                            class="text-xs px-2 py-1 rounded-full font-medium border"
                            :class="provisioningStatusClass(container.provisioning.cpu)"
                          >
                            {{ provisioningStatus(container.provisioning.cpu) }}
                          </span>
                        </div>
                      </div>
                    </div>
                    <!-- Memory Provisioning -->
                    <div class="space-y-3">
                      <h4 class="text-sm font-semibold text-secondary-light mb-3">
                        Memory Provisioning
                      </h4>
                      <div class="space-y-2">
                        <!-- Efficiency -->
                        <div>
                          <div class="flex items-center justify-between mb-1">
                            <span class="text-xs text-on-surface-secondary">Efficiency</span>
                            <span
                              class="text-xs font-medium"
                              :class="scoreColor(container.provisioning.memory.efficiency, { midThreshold: 0.5, type: 'text' })"
                            >
                              {{ formatPercentage(container.provisioning.memory.efficiency) }}
                            </span>
                          </div>
                          <div class="w-full bg-surface-elevated rounded-full h-2">
                            <div
                              class="h-2 rounded-full transition-all"
                              :class="scoreColor(container.provisioning.memory.efficiency, { midThreshold: 0.5, type: 'bg' })"
                              :style="{ width: `${container.provisioning.memory.efficiency * 100}%` }"
                            />
                          </div>
                        </div>
                        <!-- Current Request -->
                        <div class="flex items-center justify-between">
                          <span class="text-xs text-on-surface-secondary">Current Request</span>
                          <span class="text-xs font-medium text-on-surface">
                            {{ formatBytes(container.provisioning.memory.current_request ?? undefined) }}
                          </span>
                        </div>
                        <!-- Request Utilization -->
                        <div class="flex items-center justify-between">
                          <span class="text-xs text-on-surface-secondary">Request Utilization</span>
                          <span class="text-xs font-medium text-on-surface">
                            {{ formatPercentage(container.provisioning.memory.request_utilization) }}
                          </span>
                        </div>
                        <!-- Current Limit -->
                        <div class="flex items-center justify-between">
                          <span class="text-xs text-on-surface-secondary">Current Limit</span>
                          <span class="text-xs font-medium text-on-surface">
                            {{ formatBytes(container.provisioning.memory.current_limit ?? undefined) }}
                          </span>
                        </div>
                        <!-- Limit Utilization -->
                        <div class="flex items-center justify-between">
                          <span class="text-xs text-on-surface-secondary">Limit Utilization</span>
                          <span class="text-xs font-medium text-on-surface">
                            {{ formatPercentage(container.provisioning.memory.limit_utilization) }}
                          </span>
                        </div>
                        <!-- Status Badge -->
                        <div class="flex items-center justify-between pt-1">
                          <span class="text-xs text-on-surface-secondary">Status</span>
                          <span
                            class="text-xs px-2 py-1 rounded-full font-medium border"
                            :class="provisioningStatusClass(container.provisioning.memory)"
                          >
                            {{ provisioningStatus(container.provisioning.memory) }}
                          </span>
                        </div>
                      </div>
                    </div>
                    <!-- Stability Details -->
                    <div class="space-y-3">
                      <h4 class="text-sm font-semibold text-tertiary-light mb-3">
                        Stability Metrics
                      </h4>
                      <div class="space-y-2">
                        <!-- Stability Score -->
                        <div>
                          <div class="flex items-center justify-between mb-1">
                            <span class="text-xs text-on-surface-secondary">Health Score</span>
                            <span
                              class="text-xs font-medium"
                              :class="scoreColor(container.stability?.stability_score, { midThreshold: 0.6, type: 'text' })"
                            >
                              {{ formatPercentage(container.stability?.stability_score ?? 0) }}
                            </span>
                          </div>
                          <div class="w-full bg-surface-elevated rounded-full h-2">
                            <div
                              class="h-2 rounded-full transition-all"
                              :class="scoreColor(container.stability?.stability_score, { midThreshold: 0.6, type: 'bg' })"
                              :style="{ width: `${(container.stability?.stability_score ?? 0) * 100}%` }"
                            />
                          </div>
                        </div>
                        <!-- CPU Throttling -->
                        <div class="flex items-center justify-between">
                          <span class="text-xs text-on-surface-secondary">CPU Throttling</span>
                          <span
                            class="text-xs font-medium"
                            :class="metricColor(container.stability?.cpu_throttling ?? 0, 0.1)"
                          >
                            {{ formatPercentage(container.stability?.cpu_throttling ?? 0) }}
                          </span>
                        </div>
                        <!-- CPU Pressure -->
                        <div class="flex items-center justify-between">
                          <span class="text-xs text-on-surface-secondary">CPU Pressure</span>
                          <span
                            class="text-xs font-medium"
                            :class="metricColor(container.stability?.cpu_pressure ?? 0, 0.2)"
                          >
                            {{ formatPercentage(container.stability?.cpu_pressure ?? 0) }}
                          </span>
                        </div>
                        <!-- Memory Fail Count -->
                        <div class="flex items-center justify-between">
                          <span class="text-xs text-on-surface-secondary">Memory Fail</span>
                          <span
                            class="text-xs font-medium"
                            :class="metricColor(container.stability?.memory_fail_cnt ?? 0, 1)"
                          >
                            {{ container.stability?.memory_fail_cnt ?? 0 }}
                          </span>
                        </div>
                        <!-- Memory Pressure -->
                        <div class="flex items-center justify-between">
                          <span class="text-xs text-on-surface-secondary">Memory Pressure</span>
                          <span
                            class="text-xs font-medium"
                            :class="metricColor(container.stability?.memory_pressure ?? 0, 0.1)"
                          >
                            {{ formatPercentage(container.stability?.memory_pressure ?? 0) }}
                          </span>
                        </div>
                        <!-- Memory OOM -->
                        <div class="flex items-center justify-between">
                          <span class="text-xs text-on-surface-secondary">
                            Memory OOM
                          </span>
                          <span
                            class="text-xs font-medium"
                            :class="(container.stability?.memory_oom ?? 0) > 0 ? 'text-error-light' : 'text-on-surface'"
                          >
                            {{ container.stability?.memory_oom ?? 0 }}
                          </span>
                        </div>
                        <!-- Restarts -->
                        <div class="flex items-center justify-between pt-1">
                          <span class="text-xs text-on-surface-secondary">Restarts</span>
                          <span
                            class="text-xs font-medium"
                            :class="(container.stability?.restarts ?? 0) > 0 ? 'text-error-light' : 'text-on-surface'"
                          >
                            {{ container.stability?.restarts ?? 0 }}
                          </span>
                        </div>
                      </div>
                    </div>
                  </div>
                </td>
              </tr>
            </Transition>
          </template>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import {
  UTILIZATION_METRIC_OPTIONS,
  UTILIZATION_RESOURCE_OPTIONS,
} from '#shared/constants/compute/utilization'
import { utilizationSortMetricValue } from '#shared/utils/compute/utilization'
import { formatCPU } from '#shared/utils/compute/format'
import type { PodAnalysis, ContainerAnalysis, ResourceProvisioning } from '#shared/types/compute'

const props = defineProps<{
  pods?: PodAnalysis[]
}>()

const sortMetric = ref<string>('p95')
const sortResource = ref<string>('cpu')

const filteredContainers = computed(() => {
  if (!props.pods?.length) return []

  const containers: Array<ContainerAnalysis & { pod_name: string }> = []
  for (const pod of props.pods) {
    if (!pod.containers?.length) continue
    for (const container of pod.containers) {
      containers.push({
        ...container,
        pod_name: pod.pod_name,
      })
    }
  }

  return containers.sort((a, b) => {
    const aValue = utilizationSortMetricValue(a.utilization, sortMetric.value, sortResource.value)
    const bValue = utilizationSortMetricValue(b.utilization, sortMetric.value, sortResource.value)
    return bValue - aValue
  })
})

const expandedRows = ref<Set<string>>(new Set())

const toggleExpand = (key: string) => {
  if (expandedRows.value.has(key)) {
    expandedRows.value.delete(key)
  }
  else {
    expandedRows.value.add(key)
  }
}

const provisioningStatus = (provisioning: ResourceProvisioning): string => {
  if (provisioning.is_over_provisioned) return 'Over-provisioned'
  if (provisioning.is_under_provisioned) return 'Under-provisioned'
  return 'Optimal'
}

const provisioningStatusClass = (provisioning: ResourceProvisioning): string => {
  if (provisioning.is_over_provisioned) return 'bg-warning-light/20 text-warning-light border-warning-light/30'
  if (provisioning.is_under_provisioned) return 'bg-error-light/20 text-error-light border-error-light/30'
  return 'bg-success-light/20 text-success-light border-success-light/30'
}

const metricColor = (value: number, threshold: number): string => {
  if (value > threshold) return 'text-error-light'
  if (value > threshold * 0.5) return 'text-warning-light'
  return 'text-on-surface'
}
</script>
