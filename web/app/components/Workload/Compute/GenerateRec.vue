<template>
  <UiModal
    v-model="isOpen"
    title="Generate Recommendation"
    class="max-w-xl max-h-[85vh]"
    :close-on-backdrop-click="false"
  >
    <div class="space-y-6">
      <!-- Recommendation Mode Selection -->
      <div>
        <label class="text-sm font-medium text-on-surface-secondary mb-3 block">
          Recommendation Mode
        </label>
        <div class="grid grid-cols-1 gap-3">
          <button
            v-for="mode in modes"
            :key="mode.value"
            class="flex items-start gap-4 p-4 rounded-lg border-2 transition-all text-left
             hover:bg-primary/5 cursor-pointer"
            :class="selectedMode === mode.value
              ? 'border-primary-light bg-primary/10'
              : 'border-primary/20 bg-surface-elevated'"
            @click="selectedMode = mode.value"
          >
            <div
              class="shrink-0 w-10 h-10 rounded-lg flex items-center justify-center
               transition-colors self-center text-primary-light"
              :class="selectedMode === mode.value
                ? 'bg-primary-light/20'
                : 'bg-primary/10'"
            >
              <Icon
                :name="mode.icon"
                class="text-xl"
              />
            </div>
            <div class="flex-1 min-w-0">
              <h3 class="font-semibold text-on-surface mb-1">
                {{ mode.label }}
              </h3>
              <p class="text-sm text-on-surface-secondary">
                {{ mode.description }}
              </p>
            </div>
            <div
              v-if="selectedMode === mode.value"
              class="shrink-0"
            >
              <Icon
                name="lucide:circle-check"
                class="text-primary-light text-xl"
              />
            </div>
          </button>
        </div>
      </div>

      <!-- Time Range Selector -->
      <div>
        <label class="text-sm font-medium text-on-surface-secondary mb-3 block">
          Analysis Time Range
        </label>
        <UiTimeRangeSelector v-model="timeRange" />
      </div>

      <!-- Error State -->
      <Transition
        enter-active-class="transition ease-out duration-200"
        enter-from-class="opacity-0 translate-y-2"
        enter-to-class="opacity-100 translate-y-0"
        leave-active-class="transition ease-in duration-150"
        leave-from-class="opacity-100 translate-y-0"
        leave-to-class="opacity-0 translate-y-2"
      >
        <UiAlert
          v-if="error"
          variant="error"
          title="Error generating recommendation"
          :description="parseError(error, 'Failed to generate recommendation').message"
        />
      </Transition>

      <!-- Action Buttons -->
      <div class="flex items-center justify-end gap-3 pt-4 border-t border-primary/20">
        <button
          class="px-4 py-2 rounded-lg text-sm font-medium text-on-surface-secondary
           hover:bg-primary/10 hover:text-on-surface transition-all cursor-pointer"
          :disabled="generating"
          @click="close"
        >
          Cancel
        </button>
        <button
          class="px-4 py-2 rounded-lg text-sm font-medium text-primary-light bg-primary/20
           hover:bg-primary/30 transition-all cursor-pointer disabled:opacity-50
           disabled:cursor-not-allowed flex items-center gap-2"
          :disabled="generating || !selectedMode || !timeRange"
          @click="generateRecommendation"
        >
          <Icon
            v-if="generating"
            name="lucide:loader-circle"
            class="text-base animate-spin"
          />
          <Icon
            v-else
            name="lucide:lightbulb"
            class="text-base"
          />
          <span>{{ generating ? 'Generating...' : 'Generate' }}</span>
        </button>
      </div>
    </div>
  </UiModal>
</template>

<script setup lang="ts">
import type { FetchError } from 'ofetch'
import type { Recommendation, RecommendationMode } from '#shared/types/compute'

const props = defineProps<{
  namespace: string
  workloadType: string
  workloadName: string
  defaultTimeRange?: string | null
}>()

const emit = defineEmits<{
  generated: [recommendation: Recommendation]
}>()

const isOpen = defineModel<boolean>({ required: true })

const modes = [
  {
    value: 'cost_optimized',
    label: 'Cost Optimized',
    icon: 'lucide:dollar-sign',
    description: 'Maximum cost savings, accept throttling risk',
  },
  {
    value: 'burstable',
    label: 'Burstable',
    icon: 'lucide:activity',
    description: 'Balance performance, reliability, and efficiency',
  },
  {
    value: 'guaranteed',
    label: 'Guaranteed',
    icon: 'lucide:circle-check',
    description: 'Best performance, no throttling risk',
  },
] as const

const { parseError } = useApiError()

const selectedMode = ref<RecommendationMode>('burstable')
const timeRange = ref<string | null>(props.defaultTimeRange ?? '7d')
const generating = ref(false)
const error = ref<FetchError | null>(null)

const close = () => {
  isOpen.value = false
  error.value = null
}

const generateRecommendation = async () => {
  if (!selectedMode.value || !timeRange.value || generating.value) {
    return
  }

  generating.value = true
  error.value = null
  try {
    const recommendation = await $api<Recommendation>(
      `/api/v1/compute/recommendations/generate/${props.workloadType}/${props.workloadName}?namespace=${props.namespace}&timeRange=${timeRange.value}&mode=${selectedMode.value}`,
      {
        method: 'POST',
      },
    )

    emit('generated', recommendation)
    close()
  }
  catch (err) {
    error.value = err as FetchError
  }
  finally {
    generating.value = false
  }
}

watch(isOpen, () => {
  selectedMode.value = 'burstable'
  timeRange.value = props.defaultTimeRange ?? '7d'
  error.value = null
})

watch([selectedMode, timeRange], () => {
  error.value = null
})
</script>
