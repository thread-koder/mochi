<template>
  <div
    v-if="workloads && workloads.length > 0"
    class="glass rounded-xl p-6"
  >
    <div class="flex items-center justify-between mb-4">
      <h2 class="text-2xl font-bold font-heading">
        Workloads by Usage
      </h2>
      <div class="flex items-center gap-2">
        <UiSearchableSelect
          v-model="sortMetric"
          :options="metricOptions"
          :searchable="false"
          placeholder="Metric"
        />
        <UiSearchableSelect
          v-model="sortResource"
          :options="resourceOptions"
          :searchable="false"
          placeholder="Resource"
        />
        <UiSearchableSelect
          v-model="filterType"
          :options="workloadTypeOptions"
          :searchable="false"
          placeholder="All Types"
          null-option="All Types"
        />
      </div>
    </div>
    <div class="overflow-x-auto">
      <table class="w-full">
        <thead>
          <tr class="border-b border-primary/20 text-sm text-on-surface-secondary">
            <th class="text-left py-2 px-4">
              Workload
            </th>
            <th class="text-left py-2 px-4">
              Type
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
          <tr v-if="filteredWorkloads.length === 0">
            <td
              colspan="7"
              class="py-8 px-4"
            >
              <div class="text-on-surface-secondary text-center font-medium">
                <span>No workloads found</span>
                <span v-if="filterType">
                  <span> for type: </span>
                  <span class="text-primary-light capitalize">{{ filterType }}</span>
                </span>
              </div>
            </td>
          </tr>
          <tr
            v-for="workload in filteredWorkloads"
            :key="`${workload.workload_type}-${workload.workload_name}`"
            class="border-b border-primary/20 hover:bg-primary/10 transition-all cursor-pointer"
            @click="navigateToWorkload(workload.workload_type, workload.workload_name)"
          >
            <td class="py-3 px-4">
              <NuxtLink
                :to="`/namespaces/${namespace}/workloads/${workload.workload_type}/${workload.workload_name}`"
                class="text-primary-light font-medium text-sm"
                @click.stop
              >
                {{ workload.workload_name }}
              </NuxtLink>
            </td>
            <td class="py-3 px-4">
              <span class="px-2 py-1 rounded-full text-xs font-medium bg-primary/20 text-primary-light border border-primary/30">
                {{ workload.workload_type }}
              </span>
            </td>
            <td class="py-3 px-4 text-right">
              <div class="text-sm">
                <div class="text-primary-light">
                  {{ formatCPU(workload.utilization.cpu.current) }}
                </div>
                <div class="text-secondary-light text-sm">
                  {{ formatBytes(workload.utilization.memory.current) }}
                </div>
              </div>
            </td>
            <td class="py-3 px-4 text-right">
              <div class="text-sm">
                <div class="text-primary-light">
                  {{ formatCPU(workload.utilization.cpu.stats.percentile.p95) }}
                </div>
                <div class="text-secondary-light text-sm">
                  {{ formatBytes(workload.utilization.memory.stats.percentile.p95) }}
                </div>
              </div>
            </td>
            <td class="py-3 px-4 text-right">
              <div class="text-sm">
                <div class="text-primary-light">
                  {{ formatCPU(workload.utilization.cpu.stats.mean) }}
                </div>
                <div class="text-secondary-light text-sm">
                  {{ formatBytes(workload.utilization.memory.stats.mean) }}
                </div>
              </div>
            </td>
            <td class="py-3 px-4 text-right">
              <div class="text-sm">
                <div class="text-primary-light">
                  {{ formatCPU(workload.utilization.cpu.stats.max) }}
                </div>
                <div class="text-secondary-light text-sm">
                  {{ formatBytes(workload.utilization.memory.stats.max) }}
                </div>
              </div>
            </td>
            <td class="py-3 px-4 text-center">
              <div class="flex flex-col items-center gap-1.5">
                <div class="inline-flex items-center">
                  <Icon
                    v-if="workload.utilization.cpu.trend.direction === 'increasing'"
                    name="lucide:trending-up"
                    class="text-xs text-error-light"
                  />
                  <Icon
                    v-else-if="workload.utilization.cpu.trend.direction === 'decreasing'"
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
                    v-if="workload.utilization.memory.trend.direction === 'increasing'"
                    name="lucide:trending-up"
                    class="text-xs text-error-light"
                  />
                  <Icon
                    v-else-if="workload.utilization.memory.trend.direction === 'decreasing'"
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
import type * as Compute from '#shared/types/compute'

const props = defineProps<{
  workloads?: Compute.NamespaceAnalysis['workloads']
  namespace: string
}>()

const sortMetric = ref<string | null>('p95')
const sortResource = ref<string | null>('cpu')
const filterType = ref<string | null>(null)

const metricOptions: Array<{ value: string, label: string }> = [
  { value: 'current', label: 'Current' },
  { value: 'p95', label: 'P95' },
  { value: 'mean', label: 'Mean' },
  { value: 'max', label: 'Max' },
]

const resourceOptions: Array<{ value: string, label: string }> = [
  { value: 'cpu', label: 'CPU' },
  { value: 'memory', label: 'Memory' },
]

const workloadTypeOptions: Array<{ value: string, label: string }> = [
  { value: 'deployment', label: 'Deployment' },
  { value: 'statefulset', label: 'StatefulSet' },
  { value: 'daemonset', label: 'DaemonSet' },
  { value: 'pod', label: 'Pod' },
]

const filteredWorkloads = computed(() => {
  if (!props.workloads) return []

  let filtered = [...props.workloads]

  if (filterType.value) {
    filtered = filtered.filter(
      w => w.workload_type.toLowerCase() === filterType.value!.toLowerCase(),
    )
  }

  filtered.sort((a, b) => {
    let aValue: number | undefined
    let bValue: number | undefined
    const resource = (sortResource.value ?? 'cpu') as 'cpu' | 'memory'

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

  return filtered
})

const navigateToWorkload = (workloadType: string, workloadName: string) => {
  navigateTo(`/namespaces/${props.namespace}/workloads/${workloadType}/${workloadName}`)
}
</script>
