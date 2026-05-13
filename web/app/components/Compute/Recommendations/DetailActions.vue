<template>
  <div class="glass rounded-xl p-6">
    <h2 class="text-2xl font-bold font-heading mb-4">
      Actions
    </h2>

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
    <div
      v-if="canApply"
      class="flex items-center justify-end gap-3 pt-4"
    >
      <button
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
    <!-- Info Messages -->
    <template v-else>
      <Transition
        enter-active-class="transition ease-out duration-200"
        enter-from-class="opacity-0 translate-y-2"
        enter-to-class="opacity-100 translate-y-0"
        leave-active-class="transition ease-in duration-150"
        leave-from-class="opacity-100 translate-y-0"
        leave-to-class="opacity-0 translate-y-2"
      >
        <UiAlert
          v-if="status === 'applied'"
          variant="success"
          title="Recommendation already applied"
          description="This recommendation has already been applied to the workload."
        />
        <UiAlert
          v-else-if="status === 'rejected'"
          variant="error"
          title="Recommendation rejected"
          description="This recommendation has been rejected."
        />
        <UiAlert
          v-else-if="status === 'superseded'"
          variant="info"
          title="Recommendation superseded"
          description="This recommendation has been superseded by a newer one."
        />
        <UiAlert
          v-else-if="!hasRecommendations"
          variant="info"
          title="No recommendations available"
          description="No recommendations are available to apply."
        />
      </Transition>
    </template>
  </div>
</template>

<script setup lang="ts">
import type { FetchError } from 'ofetch'
import type { ContainerRecommendation } from '#shared/types/compute'

const props = defineProps<{
  recommendationId: number
  status: string
  recommendations: ContainerRecommendation[]
}>()

const emit = defineEmits<{
  applied: []
}>()

const applying = ref(false)
const applied = ref(false)
const error = ref<FetchError | null>(null)

const { parseError } = useApiError()

const hasRecommendations = computed(
  () => (props.recommendations?.length ?? 0) > 0,
)

const canApply = computed(() => {
  return (
    hasRecommendations.value && props.status === 'pending'
  )
})

const applyRecommendation = async () => {
  if (!canApply.value || applying.value) {
    return
  }

  applying.value = true
  error.value = null
  try {
    await $api(`/api/v1/compute/recommendations/apply?id=${props.recommendationId}`, {
      method: 'POST',
    })

    applied.value = true
    emit('applied')
  }
  catch (err) {
    error.value = err as FetchError
  }
  finally {
    applying.value = false
  }
}
</script>
