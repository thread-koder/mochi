<template>
  <div
    v-if="containers && containers.length > 0"
    class="glass rounded-xl p-6"
  >
    <div class="flex items-center justify-between mb-4">
      <h2 class="text-2xl font-bold font-heading">
        Containers by Usage
      </h2>
      <div class="flex items-center gap-2">
        <select
          v-model="sortMetric"
          class="bg-surface-elevated border border-primary/20 rounded-lg px-3 py-1 text-sm text-on-surface-secondary focus:outline-none focus:border-primary/50"
          @change="sortContainers"
        >
          <option value="current">
            Current
          </option>
          <option value="p95">
            P95
          </option>
          <option value="mean">
            Mean
          </option>
          <option value="max">
            Max
          </option>
        </select>
        <select
          v-model="sortResource"
          class="bg-surface-elevated border border-primary/20 rounded-lg px-3 py-1 text-sm text-on-surface-secondary focus:outline-none focus:border-primary/50"
          @change="sortContainers"
        >
          <option value="cpu">
            CPU
          </option>
          <option value="memory">
            Memory
          </option>
        </select>
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
          </tr>
        </thead>
        <tbody>
          <tr
            v-if="filteredContainers.length === 0"
            class="border-b border-primary/20"
          >
            <td
              colspan="7"
              class="py-8 px-4"
            >
              <div class="text-on-surface-secondary text-center font-medium">
                <span>No containers found</span>
              </div>
            </td>
          </tr>
          <tr
            v-for="container in filteredContainers"
            :key="`${container.pod_name}-${container.container_name}`"
            class="border-b border-primary/20 hover:bg-primary/10 transition-all"
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
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import type * as Compute from '#shared/types/compute/analysis'

const props = defineProps<{
  pods?: Compute.WorkloadAnalysis['pods']
}>()

const sortMetric = ref<'current' | 'p95' | 'mean' | 'max'>('p95')
const sortResource = ref<'cpu' | 'memory'>('cpu')
const filteredContainers = ref<Array<Compute.WorkloadContainerAnalysis & { pod_name: string }>>([])

const containers = computed(() => {
  if (!props.pods) return []
  const allContainers: Array<Compute.WorkloadContainerAnalysis & { pod_name: string }> = []
  props.pods.forEach((pod) => {
    if (pod.containers) {
      pod.containers.forEach((container) => {
        allContainers.push({
          ...container,
          pod_name: pod.pod_name,
        })
      })
    }
  })
  return allContainers
})

const sortContainers = () => {
  if (!containers.value || containers.value.length === 0) {
    filteredContainers.value = []
    return
  }

  filteredContainers.value = [...containers.value].sort((a, b) => {
    let aValue: number | undefined
    let bValue: number | undefined
    const resource = sortResource.value

    if (sortMetric.value === 'current') {
      aValue = a.utilization[resource].current
      bValue = b.utilization[resource].current
    }
    else if (sortMetric.value === 'p95') {
      aValue = a.utilization[resource].stats.percentile.p95
      bValue = b.utilization[resource].stats.percentile.p95
    }
    else if (sortMetric.value === 'mean') {
      aValue = a.utilization[resource].stats.mean
      bValue = b.utilization[resource].stats.mean
    }
    else if (sortMetric.value === 'max') {
      aValue = a.utilization[resource].stats.max
      bValue = b.utilization[resource].stats.max
    }

    return (bValue ?? 0) - (aValue ?? 0)
  })
}

watch(containers, () => {
  sortContainers()
}, { immediate: true })
</script>
