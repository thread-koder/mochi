<template>
  <div>
    <div class="flex items-center justify-between flex-wrap gap-4 mb-4">
      <h2 class="text-2xl font-bold font-heading">
        Summary
      </h2>
      <div
        role="group"
        aria-label="Analysis period"
      >
        <UiTimeRangeSelector v-model="timeRange" />
      </div>
    </div>

    <!-- Loading State -->
    <div
      v-if="analysisPending"
      class="py-6 flex flex-col items-center justify-center gap-2 text-on-surface-secondary"
    >
      <Icon
        name="lucide:loader-circle"
        class="text-2xl animate-spin"
      />
      <p class="text-sm font-medium">
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
      <section v-if="analysis.time_series">
        <h2 class="text-2xl font-bold font-heading mb-4">
          Resource Utilization
        </h2>
        <div class="panel p-4">
          <div class="space-y-6">
            <!-- CPU Chart -->
            <div>
              <h3 class="text-lg font-semibold font-heading mb-2 text-on-surface">
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
              <h3 class="text-lg font-semibold font-heading mb-2 text-on-surface">
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
      </section>

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
  analysisError.value = null
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
