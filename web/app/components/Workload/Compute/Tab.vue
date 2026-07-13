<template>
  <div>
    <div class="flex items-center justify-between flex-wrap gap-4 mb-4">
      <h2 class="text-2xl font-bold font-heading">
        Summary
      </h2>
      <div class="flex items-center gap-3 flex-wrap">
        <div
          role="group"
          aria-label="Analysis period"
        >
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
      class="py-6 flex flex-col items-center justify-center gap-2 text-on-surface-secondary"
    >
      <Icon
        name="lucide:loader-circle"
        class="text-2xl animate-spin"
      />
      <p class="text-sm font-medium">
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
      <ComputeSummaryMetrics
        :utilization="analysis.utilization"
        :stability="analysis.stability"
      />
      <section v-if="analysis.time_series">
        <h2 class="text-2xl font-bold font-heading mb-4">
          Resource Utilization
        </h2>
        <div class="panel p-4">
          <div class="space-y-6">
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
      <WorkloadComputePodsBreakdown :pods="analysis.pods" />
      <WorkloadComputeContainersBreakdown :pods="analysis.pods" />
    </div>

    <ClientOnly>
      <WorkloadComputeGenerateRec
        v-model="showGenerateModal"
        :namespace="props.namespace"
        :workload-type="props.workloadType"
        :workload-name="props.workloadName"
        :default-time-range="timeRange"
        @generated="onRecommendationGenerated"
      />
      <WorkloadComputeRecResult
        v-model="showResultModal"
        :recommendation="generatedRecommendation"
      />
    </ClientOnly>
  </div>
</template>

<script setup lang="ts">
import type { FetchError } from 'ofetch'
import type { WorkloadAnalysis, Recommendation } from '#shared/types/compute'

const props = defineProps<{
  namespace: string
  workloadType: string
  workloadName: string
  isActive?: boolean
}>()

const { parseError } = useApiError()

const timeRange = ref('24h')
const analysisPending = ref(false)
const analysisError = ref<FetchError | null>(null)
const analysis = ref<WorkloadAnalysis | null>(null)

const showGenerateModal = ref(false)
const showResultModal = ref(false)
const generatedRecommendation = ref<Recommendation | null>(null)

const executeAnalysis = async () => {
  analysisPending.value = true
  analysisError.value = null
  try {
    analysis.value = await $api<WorkloadAnalysis>(`/api/v1/compute/analyze/workloads/${props.workloadType}/${props.workloadName}?namespace=${props.namespace}&timeRange=${timeRange.value}`)
  }
  catch (error) {
    analysisError.value = error as FetchError
  }
  finally {
    analysisPending.value = false
  }
}

watch(
  [
    () => props.isActive === true,
    timeRange,
    () => props.namespace,
    () => props.workloadType,
    () => props.workloadName,
  ],
  async ([isActive, range, ns, wlType, wlName], [, prevRange]) => {
    if (!isActive || !range || !ns || !wlType || !wlName) return

    const timeChanged = prevRange !== undefined && range !== prevRange
    if (analysis.value !== null && !timeChanged) return

    await executeAnalysis()
  },
  { immediate: true },
)

const onRecommendationGenerated = (recommendation: Recommendation) => {
  generatedRecommendation.value = recommendation
  showGenerateModal.value = false
  showResultModal.value = true
}
</script>
