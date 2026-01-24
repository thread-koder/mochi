<template>
  <div class="mt-6">
    <!-- Time Range Selector -->
    <UiTimeRangeSelector v-model="timeRange" />

    <!-- Loading State -->
    <div
      v-if="analysisPending"
      class="text-center py-12"
    >
      <div class="inline-block animate-spin rounded-full h-12 w-12 border-b-2 border-primary-light" />
      <p class="mt-4 text-on-surface-secondary">
        Analyzing workload...
      </p>
    </div>

    <!-- Error State -->
    <div
      v-else-if="analysisError"
      class="glass rounded-xl p-6 mb-6"
    >
      <div class="text-error-light">
        <p class="font-semibold mb-2">
          Error
        </p>
        <p>{{ parseError(analysisError, 'Failed to load analysis').message }}</p>
      </div>
    </div>

    <!-- Analysis Results -->
    <div
      v-else-if="analysis"
      class="space-y-6"
    >
      <!-- Summary Metrics -->
      <ComputeSummaryMetrics :utilization="analysis.utilization" />

      <!-- Resource Utilization Charts -->
      <div
        v-if="analysis.time_series"
        class="glass rounded-xl p-6"
      >
        <h2 class="text-2xl font-bold font-heading mb-4">
          Resource Utilization
        </h2>
        <div class="space-y-6">
          <!-- CPU Chart -->
          <div>
            <h3 class="text-xl font-semibold font-heading mb-2 text-primary-light">
              CPU Utilization Over Time
            </h3>
            <ComputeResourceChart
              :data="analysis.time_series.cpu"
              type="cpu"
              title="CPU Utilization"
            />
          </div>
          <!-- Memory Chart -->
          <div>
            <h3 class="text-xl font-semibold font-heading mb-2 text-secondary-light">
              Memory Utilization Over Time
            </h3>
            <ComputeResourceChart
              :data="analysis.time_series.memory"
              type="memory"
              title="Memory Utilization"
            />
          </div>
        </div>
      </div>

      <!-- Pods Breakdown -->
      <WorkloadPodsBreakdown :pods="analysis.pods" />

      <!-- Containers Breakdown -->
      <WorkloadContainersBreakdown :pods="analysis.pods" />
    </div>
  </div>
</template>

<script setup lang="ts">
import type { FetchError } from 'ofetch'
import type * as Compute from '#shared/types/compute'

const props = defineProps<{
  namespace: string
  workloadType: string
  workloadName: string
  isActive?: boolean
}>()

const { parseError } = useApiError()

const timeRange = ref<string | null>(null)
const analysisPending = ref(false)
const analysisError = ref<FetchError | null>(null)
const analysis = ref<Compute.WorkloadAnalysis | null>(null)

const executeAnalysis = async () => {
  analysisPending.value = true
  try {
    analysis.value = await $api<Compute.WorkloadAnalysis>(`/api/v1/compute/analyze/workloads/${props.workloadType}/${props.workloadName}?namespace=${props.namespace}&timeRange=${timeRange.value}&includeTimeSeries=true`)
  }
  catch (error) {
    analysisError.value = error as FetchError
  }
  finally {
    analysisPending.value = false
  }
}

watch(timeRange, async (newTimeRange, oldTimeRange) => {
  if (newTimeRange === 'custom') return

  if (!newTimeRange) {
    timeRange.value = '24h'
    return
  }

  const timeChanged = newTimeRange !== oldTimeRange
  if (timeChanged) {
    await executeAnalysis()
  }
})

// Watch for when tab becomes active to trigger initial load
watch(() => props.isActive, async (isActive) => {
  if (isActive && !timeRange.value) {
    timeRange.value = '24h'
  }
}, { immediate: true })
</script>
