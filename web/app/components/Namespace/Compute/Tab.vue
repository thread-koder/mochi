<template>
  <div class="mt-6">
    <!-- Time Range Selector -->
    <div class="glass rounded-xl p-4 mb-6">
      <div class="flex items-center justify-between flex-wrap gap-4">
        <label class="text-sm text-on-surface-secondary">
          Analysis Time Range:
        </label>
        <UiTimeRangeSelector v-model="timeRange" />
      </div>
    </div>

    <!-- Loading State -->
    <div
      v-if="analysisPending"
      class="text-center py-12"
    >
      <div class="inline-block animate-spin rounded-full h-12 w-12 border-b-2 border-primary-light" />
      <p class="mt-4 text-on-surface-secondary">
        Analyzing namespace...
      </p>
    </div>

    <!-- Error State -->
    <UiAlert
      v-else-if="analysisError"
      variant="error"
      title="Error loading analysis"
      :description="parseError(analysisError, 'Failed to load analysis').message"
    />

    <!-- Analysis Results -->
    <div
      v-else-if="analysis"
      class="space-y-6"
    >
      <!-- Summary Metrics -->
      <ComputeSummaryMetrics
        :utilization="analysis.utilization"
        :stability="analysis.stability"
      />

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

      <!-- Workloads Breakdown -->
      <NamespaceComputeWorkloadsBreakdown
        :workloads="analysis.workloads"
        :namespace="namespace"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import type { FetchError } from 'ofetch'
import type { NamespaceAnalysis } from '#shared/types/compute'

const props = defineProps<{
  namespace: string
  isActive?: boolean
}>()

const { parseError } = useApiError()

const timeRange = ref('24h')
const analysisPending = ref(false)
const analysisError = ref<FetchError | null>(null)
const analysis = ref<NamespaceAnalysis | null>(null)

const executeAnalysis = async () => {
  analysisPending.value = true
  try {
    analysis.value = await $api<NamespaceAnalysis>(`/api/v1/compute/analyze/namespaces/${props.namespace}?timeRange=${timeRange.value}`)
  }
  catch (error) {
    analysisError.value = error as FetchError
  }
  finally {
    analysisPending.value = false
  }
}

watch(
  [() => props.isActive === true, timeRange, () => props.namespace],
  async ([isActive, range, ns], [, prevRange]) => {
    if (!isActive || !range || !ns) return

    const timeChanged = prevRange !== undefined && range !== prevRange
    if (analysis.value !== null && !timeChanged) return

    await executeAnalysis()
  },
  { immediate: true },
)
</script>
