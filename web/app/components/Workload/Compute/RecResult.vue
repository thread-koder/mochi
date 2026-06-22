<template>
  <UiModal
    v-model="isOpen"
    title="Recommendation Results"
    class="max-w-4xl max-h-[85vh]"
    :close-on-backdrop-click="false"
  >
    <div class="space-y-6">
      <!-- Empty state -->
      <UiEmptyState
        v-if="!hasRecommendations"
        icon="lucide:inbox"
        title="No recommendations available."
        description="Try a different time range or recommendation mode."
      />

      <!-- Recommendations Table -->
      <div
        v-else
        class="overflow-x-auto"
      >
        <table class="w-full">
          <thead>
            <tr class="border-b border-primary/20 text-sm text-on-surface-secondary">
              <th class="text-left py-3 px-4">
                Container
              </th>
              <th class="text-left py-3 px-4">
                CPU
              </th>
              <th class="text-left py-3 px-4">
                Memory
              </th>
              <th class="text-center py-3 px-4">
                Confidence
              </th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="rec in recommendation?.recommendations"
              :key="rec.container_name"
              class="border-b border-primary/10 hover:bg-primary/5 transition-colors"
            >
              <td class="py-4 px-4">
                <span class="font-medium text-on-surface">
                  {{ rec.container_name }}
                </span>
              </td>
              <td class="py-4 px-4">
                <ComputeRecommendationsResourceGroup :resource="rec.cpu" />
              </td>
              <td class="py-4 px-4">
                <ComputeRecommendationsResourceGroup :resource="rec.memory" />
              </td>
              <td class="py-4 px-4 text-center">
                <span
                  class="px-2 py-1 rounded-full text-sm font-medium border"
                  :class="scoreBadgeClass(rec.confidence)"
                >
                  {{ formatPercentage(rec.confidence) }}
                </span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Success State -->
      <Transition
        enter-active-class="transition ease-out duration-200"
        enter-from-class="opacity-0 translate-y-2"
        enter-to-class="opacity-100 translate-y-0"
        leave-active-class="transition ease-in duration-150"
        leave-from-class="opacity-100 translate-y-0"
        leave-to-class="opacity-0 translate-y-2"
      >
        <UiAlert
          v-if="applied"
          variant="success"
          title="Recommendation applied successfully"
          description="The resource recommendations have been applied to the workload."
        />
      </Transition>

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
          v-if="error && !applied"
          variant="error"
          title="Error applying recommendation"
          :description="parseError(error, 'Failed to apply recommendation').message"
        />
      </Transition>

      <!-- Action Buttons -->
      <div class="flex items-center justify-end gap-3 pt-4 border-t border-primary/20">
        <button
          v-if="applied || !hasRecommendations"
          class="px-4 py-2 rounded-lg text-sm font-medium text-primary-light bg-primary/20
           hover:bg-primary/30 transition-all cursor-pointer flex items-center gap-2"
          @click="close"
        >
          <Icon
            name="lucide:x"
            class="text-base"
          />
          <span>Close</span>
        </button>
        <button
          v-if="hasRecommendations && !applied"
          class="px-4 py-2 rounded-lg text-sm font-medium text-primary-light bg-primary/20
           hover:bg-primary/30 transition-all cursor-pointer disabled:opacity-50
           disabled:cursor-not-allowed flex items-center gap-2"
          :disabled="applying"
          @click="applyRecommendation"
        >
          <Icon
            v-if="applying"
            name="lucide:loader-circle"
            class="text-base animate-spin"
          />
          <Icon
            v-else
            name="lucide:circle-check"
            class="text-base"
          />
          <span>{{ applying ? 'Applying...' : 'Apply Recommendation' }}</span>
        </button>
      </div>
    </div>
  </UiModal>
</template>

<script setup lang="ts">
import type { FetchError } from 'ofetch'
import type { Recommendation } from '#shared/types/compute'

const props = defineProps<{
  recommendation: Recommendation | null
}>()

const isOpen = defineModel<boolean>({ required: true })

const applying = ref(false)
const applied = ref(false)
const error = ref<FetchError | null>(null)

const { parseError } = useApiError()

const hasRecommendations = computed(
  () => (props.recommendation?.recommendations?.length ?? 0) > 0,
)

const close = () => {
  isOpen.value = false
  applied.value = false
  error.value = null
}

const applyRecommendation = async () => {
  if (!props.recommendation || applying.value || applied.value) {
    return
  }

  applying.value = true
  error.value = null
  try {
    await $api('/api/v1/compute/recommendations/apply', {
      method: 'POST',
      body: props.recommendation,
    })

    applied.value = true
  }
  catch (err) {
    error.value = err as FetchError
  }
  finally {
    applying.value = false
  }
}

watch(isOpen, () => {
  applied.value = false
  error.value = null
})
</script>
