<template>
  <UiModal
    v-model="isOpen"
    title="Recommendation Results"
    class="max-w-4xl max-h-[85vh]"
    :close-on-backdrop-click="false"
  >
    <div class="space-y-6">
      <!-- Empty state: no recommendations -->
      <div
        v-if="!hasRecommendations"
        class="py-12 px-6 rounded-lg bg-primary/5 border border-primary/10 text-center"
      >
        <Icon
          name="lucide:inbox"
          class="mx-auto text-4xl text-on-surface-muted mb-4"
        />
        <p class="text-sm font-medium text-on-surface mb-1">
          Workload is stable, no recommendations available.
        </p>
        <p class="text-xs text-on-surface-muted">
          Try a different time range or recommendation mode.
        </p>
      </div>

      <!-- Recommendations Table -->
      <div
        v-else
        class="overflow-x-auto"
      >
        <table class="w-full">
          <thead>
            <tr class="border-b border-primary/20">
              <th class="text-left py-3 px-4 text-sm font-semibold text-on-surface-secondary">
                Container
              </th>
              <th class="text-left py-3 px-4 text-sm font-semibold text-on-surface-secondary">
                CPU
              </th>
              <th class="text-left py-3 px-4 text-sm font-semibold text-on-surface-secondary">
                Memory
              </th>
              <th class="text-center py-3 px-4 text-sm font-semibold text-on-surface-secondary">
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
                <div class="space-y-1">
                  <!-- CPU Request -->
                  <div
                    v-if="rec.cpu.current_request || rec.cpu.recommended_request"
                    class="text-sm flex items-center"
                  >
                    <span class="text-on-surface-secondary">Request:</span>
                    <span class="text-on-surface ml-2">
                      {{ rec.cpu.current_request ?? 'N/A' }}
                    </span>
                    <Icon
                      name="lucide:arrow-right"
                      class="mx-2 text-xs text-on-surface-muted"
                    />
                    <span
                      class="font-medium text-primary-light"
                    >
                      {{ rec.cpu.recommended_request ?? 'N/A' }}
                    </span>
                    <span
                      v-if="rec.cpu.request_change_percent !== null && rec.cpu.request_change_percent !== undefined"
                      class="ml-2 text-xs text-on-surface-muted"
                    >
                      ({{ formatChangePercent(rec.cpu.request_change_percent) }})
                    </span>
                  </div>
                  <!-- CPU Limit -->
                  <div
                    v-if="rec.cpu.current_limit || rec.cpu.recommended_limit"
                    class="text-sm flex items-center"
                  >
                    <span class="text-on-surface-secondary">Limit:</span>
                    <span class="text-on-surface ml-2">
                      {{ rec.cpu.current_limit ?? 'N/A' }}
                    </span>
                    <Icon
                      name="lucide:arrow-right"
                      class="mx-2 text-xs text-on-surface-muted"
                    />
                    <span
                      class="font-medium text-primary-light"
                    >
                      {{ rec.cpu.recommended_limit ?? 'N/A' }}
                    </span>
                    <span
                      v-if="rec.cpu.limit_change_percent !== null && rec.cpu.limit_change_percent !== undefined"
                      class="ml-2 text-xs text-on-surface-muted"
                    >
                      ({{ formatChangePercent(rec.cpu.limit_change_percent) }})
                    </span>
                  </div>
                </div>
              </td>
              <td class="py-4 px-4">
                <div class="space-y-1">
                  <!-- Memory Request -->
                  <div
                    v-if="rec.memory.current_request || rec.memory.recommended_request"
                    class="text-sm flex items-center"
                  >
                    <span class="text-on-surface-secondary">Request:</span>
                    <span class="text-on-surface ml-2">
                      {{ rec.memory.current_request ?? 'N/A' }}
                    </span>
                    <Icon
                      name="lucide:arrow-right"
                      class="mx-2 text-xs text-on-surface-muted"
                    />
                    <span
                      class="font-medium text-primary-light"
                    >
                      {{ rec.memory.recommended_request ?? 'N/A' }}
                    </span>
                    <span
                      v-if="rec.memory.request_change_percent !== null && rec.memory.request_change_percent !== undefined"
                      class="ml-2 text-xs text-on-surface-muted"
                    >
                      ({{ formatChangePercent(rec.memory.request_change_percent) }})
                    </span>
                  </div>
                  <!-- Memory Limit -->
                  <div
                    v-if="rec.memory.current_limit || rec.memory.recommended_limit"
                    class="text-sm flex items-center"
                  >
                    <span class="text-on-surface-secondary">Limit:</span>
                    <span class="text-on-surface ml-2">
                      {{ rec.memory.current_limit ?? 'N/A' }}
                    </span>
                    <Icon
                      name="lucide:arrow-right"
                      class="mx-2 text-xs text-on-surface-muted"
                    />
                    <span
                      class="font-medium text-primary-light"
                    >
                      {{ rec.memory.recommended_limit ?? 'N/A' }}
                    </span>
                    <span
                      v-if="rec.memory.limit_change_percent !== null && rec.memory.limit_change_percent !== undefined"
                      class="ml-2 text-xs text-on-surface-muted"
                    >
                      ({{ formatChangePercent(rec.memory.limit_change_percent) }})
                    </span>
                  </div>
                </div>
              </td>
              <td class="py-4 px-4 text-center">
                <span
                  class="px-2 py-1 rounded text-sm font-medium"
                  :class="scoreBadgeClass(rec.confidence_score)"
                >
                  {{ formatPercentage(rec.confidence_score) }}
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
        <div
          v-if="applied"
          class="p-4 rounded-lg bg-success/10 border border-success/20"
        >
          <div class="flex items-start gap-3">
            <Icon
              name="lucide:check-circle-2"
              class="text-success-light text-xl shrink-0 mt-0.5"
            />
            <div class="flex-1">
              <p class="text-sm font-medium text-success-light mb-1">
                Recommendation applied successfully
              </p>
              <p class="text-xs text-success-light/80">
                The resource recommendations have been applied to the workload.
              </p>
            </div>
          </div>
        </div>
      </Transition>

      <!-- Error Message -->
      <Transition
        enter-active-class="transition ease-out duration-200"
        enter-from-class="opacity-0 translate-y-2"
        enter-to-class="opacity-100 translate-y-0"
        leave-active-class="transition ease-in duration-150"
        leave-from-class="opacity-100 translate-y-0"
        leave-to-class="opacity-0 translate-y-2"
      >
        <div
          v-if="error && !applied"
          class="p-4 rounded-lg bg-error/10 border border-error/20"
        >
          <div class="flex items-start gap-3">
            <Icon
              name="lucide:alert-circle"
              class="text-error-light text-xl shrink-0 mt-0.5"
            />
            <div class="flex-1">
              <p class="text-sm font-medium text-error-light mb-1">
                Error applying recommendation
              </p>
              <p class="text-xs text-error-light/80">
                {{ parseError(error, 'Failed to apply recommendation').message }}
              </p>
            </div>
          </div>
        </div>
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
            name="lucide:loader-2"
            class="text-base animate-spin"
          />
          <Icon
            v-else
            name="lucide:check-circle-2"
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
import type * as Compute from '#shared/types/compute'

const props = defineProps<{
  recommendation: Compute.Recommendation | null
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

// Reset state when modal opens
watch(isOpen, () => {
  applied.value = false
  error.value = null
})
</script>
