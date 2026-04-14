<template>
  <div class="mt-6">
    <!-- Action Bar -->
    <div class="glass rounded-xl p-4 mb-6">
      <div class="flex items-center justify-between flex-wrap gap-4">
        <div class="flex items-center gap-4 flex-wrap">
          <label class="text-sm text-on-surface-secondary">
            Analysis Time Range:
          </label>
          <UiTimeRangeSelector v-model="timeRange" />
        </div>
        <button
          class="px-4 py-2 rounded-lg text-sm font-medium text-primary-light bg-primary/20
           hover:bg-primary/30 transition-all cursor-pointer flex items-center gap-2"
          @click="showGenerateModal = true"
        >
          <Icon
            name="lucide:lightbulb"
            class="text-base"
          />
          <span>Generate Recommendation</span>
        </button>
      </div>
    </div>

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

      <!-- Pods Breakdown -->
      <WorkloadComputePodsBreakdown :pods="analysis.pods" />

      <!-- Containers Breakdown -->
      <WorkloadComputeContainersBreakdown :pods="analysis.pods" />
    </div>

    <ClientOnly>
      <!-- Generate Recommendation Modal -->
      <WorkloadComputeGenerateRec
        v-model="showGenerateModal"
        :namespace="props.namespace"
        :workload-type="props.workloadType"
        :workload-name="props.workloadName"
        :default-time-range="timeRange"
        @generated="onRecommendationGenerated"
      />

      <!-- Recommendation Results Modal -->
      <WorkloadComputeRecResult
        v-model="showResultModal"
        :recommendation="generatedRecommendation"
      />
    </ClientOnly>
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

const showGenerateModal = ref(false)
const showResultModal = ref(false)
const generatedRecommendation = ref<Compute.Recommendation | null>(null)

const executeAnalysis = async () => {
  analysisPending.value = true
  try {
    analysis.value = await $api<Compute.WorkloadAnalysis>(`/api/v1/compute/analyze/workloads/${props.workloadType}/${props.workloadName}?namespace=${props.namespace}&timeRange=${timeRange.value}`)
  }
  catch (error) {
    analysisError.value = error as FetchError
  }
  finally {
    analysisPending.value = false
  }
}

watch(timeRange, async (newTimeRange, oldTimeRange) => {
  if (!newTimeRange) {
    timeRange.value = '24h'
    return
  }

  const timeChanged = newTimeRange !== oldTimeRange
  if (timeChanged) {
    await executeAnalysis()
  }
})

watch(() => props.isActive, async (isActive) => {
  if (isActive && !timeRange.value) {
    timeRange.value = '24h'
  }
}, { immediate: true })

const onRecommendationGenerated = (recommendation: Compute.Recommendation) => {
  generatedRecommendation.value = recommendation
  showGenerateModal.value = false
  showResultModal.value = true
}
</script>
