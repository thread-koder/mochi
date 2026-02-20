<template>
  <div class="glass rounded-xl p-4 mb-6">
    <div class="space-y-4">
      <!-- Filter Inputs -->
      <div class="flex items-center gap-3 flex-wrap">
        <!-- Namespace Filter -->
        <div class="flex items-center gap-2">
          <Icon
            name="lucide:layers"
            class="text-lg text-on-surface-secondary shrink-0"
          />
          <UiSearchableSelect
            v-model="filters.namespace"
            :options="namespaceOptions"
            placeholder="All Namespaces"
            search-placeholder="Search namespaces..."
            null-option="All Namespaces"
          />
        </div>

        <!-- Status Filter -->
        <div class="flex items-center gap-2">
          <Icon
            name="lucide:circle-check"
            class="text-lg text-on-surface-secondary shrink-0"
          />
          <UiSearchableSelect
            v-model="filters.status"
            :options="statusOptions"
            :searchable="false"
            placeholder="All Status"
            null-option="All Status"
          />
        </div>

        <!-- Mode Filter -->
        <div class="flex items-center gap-2">
          <Icon
            name="lucide:settings"
            class="text-lg text-on-surface-secondary shrink-0"
          />
          <UiSearchableSelect
            v-model="filters.mode"
            :options="modeOptions"
            :searchable="false"
            placeholder="All Modes"
            null-option="All Modes"
          />
        </div>

        <!-- Workload Type Filter -->
        <div class="flex items-center gap-2">
          <Icon
            name="lucide:server"
            class="text-lg text-on-surface-secondary shrink-0"
          />
          <UiSearchableSelect
            v-model="filters.workloadType"
            :options="workloadTypeOptions"
            :searchable="false"
            placeholder="All Types"
            null-option="All Types"
          />
        </div>

        <!-- Workload Name Search -->
        <div class="flex items-center gap-2">
          <Icon
            name="lucide:search"
            class="text-lg text-on-surface-secondary shrink-0"
          />
          <input
            v-model="workloadNameInput"
            type="text"
            placeholder="Search workload name..."
            class="bg-surface-elevated border border-primary/20 rounded-lg px-3 py-2
             text-sm text-on-surface-secondary placeholder:text-on-surface-muted
             focus:outline-none focus:border-primary/50 min-w-[180px]"
          >
        </div>
      </div>

      <!-- Active Filters & Clear All -->
      <Transition
        enter-active-class="transition ease-out duration-200"
        enter-from-class="opacity-0 -translate-y-2"
        enter-to-class="opacity-100 translate-y-0"
        leave-active-class="transition ease-in duration-150"
        leave-from-class="opacity-100 translate-y-0"
        leave-to-class="opacity-0 -translate-y-2"
      >
        <div
          v-if="hasActiveFilters"
          class="flex items-center gap-3 flex-wrap pt-2 border-t border-primary/10"
        >
          <!-- Active Filter Badges -->
          <div class="flex items-center gap-2 flex-wrap flex-1">
            <Transition
              enter-active-class="transition ease-out duration-200"
              enter-from-class="opacity-0 scale-90"
              enter-to-class="opacity-100 scale-100"
              leave-active-class="transition ease-in duration-150"
              leave-from-class="opacity-100 scale-100"
              leave-to-class="opacity-0 scale-90"
            >
              <span
                v-if="filters.namespace"
                class="inline-flex items-center gap-1.5 px-2 py-1 rounded-full text-xs font-medium
                 bg-primary/20 text-primary-light border border-primary/30"
              >
                <Icon
                  name="lucide:layers"
                  class="text-xs shrink-0"
                />
                <span>{{ filters.namespace }}</span>
                <button
                  class="hover:text-primary transition-colors shrink-0 flex items-center cursor-pointer"
                  @click="filters.namespace = null"
                >
                  <Icon
                    name="lucide:x"
                    class="text-xs"
                  />
                </button>
              </span>
            </Transition>
            <Transition
              enter-active-class="transition ease-out duration-200"
              enter-from-class="opacity-0 scale-90"
              enter-to-class="opacity-100 scale-100"
              leave-active-class="transition ease-in duration-150"
              leave-from-class="opacity-100 scale-100"
              leave-to-class="opacity-0 scale-90"
            >
              <span
                v-if="filters.status"
                class="inline-flex items-center gap-1.5 px-2 py-1 rounded-full text-xs font-medium
                 bg-primary/20 text-primary-light border border-primary/30"
              >
                <Icon
                  name="lucide:circle-check"
                  class="text-xs shrink-0"
                />
                <span>{{ statusLabel(filters.status) }}</span>
                <button
                  class="hover:text-primary transition-colors shrink-0 flex items-center cursor-pointer"
                  @click="filters.status = null"
                >
                  <Icon
                    name="lucide:x"
                    class="text-xs"
                  />
                </button>
              </span>
            </Transition>
            <Transition
              enter-active-class="transition ease-out duration-200"
              enter-from-class="opacity-0 scale-90"
              enter-to-class="opacity-100 scale-100"
              leave-active-class="transition ease-in duration-150"
              leave-from-class="opacity-100 scale-100"
              leave-to-class="opacity-0 scale-90"
            >
              <span
                v-if="filters.mode"
                class="inline-flex items-center gap-1.5 px-2 py-1 rounded-full text-xs font-medium
                 bg-primary/20 text-primary-light border border-primary/30"
              >
                <Icon
                  name="lucide:settings"
                  class="text-xs shrink-0"
                />
                <span>{{ modeLabel(filters.mode) }}</span>
                <button
                  class="hover:text-primary transition-colors shrink-0 flex items-center cursor-pointer"
                  @click="filters.mode = null"
                >
                  <Icon
                    name="lucide:x"
                    class="text-xs"
                  />
                </button>
              </span>
            </Transition>
            <Transition
              enter-active-class="transition ease-out duration-200"
              enter-from-class="opacity-0 scale-90"
              enter-to-class="opacity-100 scale-100"
              leave-active-class="transition ease-in duration-150"
              leave-from-class="opacity-100 scale-100"
              leave-to-class="opacity-0 scale-90"
            >
              <span
                v-if="filters.workloadType"
                class="inline-flex items-center gap-1.5 px-2 py-1 rounded-full text-xs font-medium
                 bg-primary/20 text-primary-light border border-primary/30"
              >
                <Icon
                  name="lucide:server"
                  class="text-xs shrink-0"
                />
                <span>{{ filters.workloadType }}</span>
                <button
                  class="hover:text-primary transition-colors shrink-0 flex items-center cursor-pointer"
                  @click="filters.workloadType = null"
                >
                  <Icon
                    name="lucide:x"
                    class="text-xs"
                  />
                </button>
              </span>
            </Transition>
            <Transition
              enter-active-class="transition ease-out duration-200"
              enter-from-class="opacity-0 scale-90"
              enter-to-class="opacity-100 scale-100"
              leave-active-class="transition ease-in duration-150"
              leave-from-class="opacity-100 scale-100"
              leave-to-class="opacity-0 scale-90"
            >
              <span
                v-if="filters.workloadName"
                class="inline-flex items-center gap-1.5 px-2 py-1 rounded-full text-xs font-medium
                 bg-primary/20 text-primary-light border border-primary/30"
              >
                <Icon
                  name="lucide:search"
                  class="text-xs shrink-0"
                />
                <span>{{ filters.workloadName }}</span>
                <button
                  class="hover:text-primary transition-colors shrink-0 flex items-center cursor-pointer"
                  @click="filters.workloadName = null"
                >
                  <Icon
                    name="lucide:x"
                    class="text-xs"
                  />
                </button>
              </span>
            </Transition>
          </div>

          <!-- Clear All Button -->
          <button
            class="px-4 py-2 rounded-lg text-xs font-medium text-on-surface-secondary
             hover:bg-primary/10 hover:text-on-surface transition-all cursor-pointer
             flex items-center gap-1.5"
            @click="clearFilters"
          >
            <Icon
              name="lucide:circle-x"
              class="text-sm shrink-0"
            />
            <span>Clear All</span>
          </button>
        </div>
      </Transition>
    </div>
  </div>
</template>

<script setup lang="ts">
import type * as Namespace from '#shared/types/namespace'

export interface FilterState {
  namespace: string | null
  status: string | null
  mode: string | null
  workloadType: string | null
  workloadName: string | null
}

const props = defineProps<{
  namespaces?: Namespace.Namespace[]
}>()

const filters = defineModel<FilterState>({
  required: true,
  default: () => ({
    namespace: null,
    status: null,
    mode: null,
    workloadType: null,
    workloadName: null,
  }),
})

const workloadNameInput = ref<string>('')

// Debounced function to update workload name filter
const updateWorkloadNameFilter = useDebounceFn((value: string) => {
  filters.value.workloadName = value || null
}, 500)

// Watch workload name input and trigger debounced update
watch(workloadNameInput, (value) => {
  updateWorkloadNameFilter(value)
})

// Sync workload name filter to input
watch(() => filters.value.workloadName, (value) => {
  if (value !== workloadNameInput.value) {
    workloadNameInput.value = value || ''
  }
}, { immediate: true })

const namespaceOptions = computed(() => {
  return props.namespaces?.map((ns: Namespace.Namespace) => ns.name) || []
})

const statusOptions: Array<{ value: string, label: string }> = [
  { value: 'pending', label: 'Pending' },
  { value: 'applied', label: 'Applied' },
  { value: 'rejected', label: 'Rejected' },
  { value: 'superseded', label: 'Superseded' },
]

const modeOptions: Array<{ value: string, label: string }> = [
  { value: 'cost_optimized', label: 'Cost Optimized' },
  { value: 'burstable', label: 'Burstable' },
  { value: 'guaranteed', label: 'Guaranteed' },
]

const workloadTypeOptions: Array<{ value: string, label: string }> = [
  { value: 'Deployment', label: 'Deployment' },
  { value: 'StatefulSet', label: 'StatefulSet' },
  { value: 'DaemonSet', label: 'DaemonSet' },
  { value: 'Pod', label: 'Pod' },
]

const hasActiveFilters = computed(() => {
  return !!(filters.value.namespace
    || filters.value.status || filters.value.mode
    || filters.value.workloadType || filters.value.workloadName)
})

const modeLabel = (mode: string): string => {
  const labels: Record<string, string> = {
    cost_optimized: 'Cost Optimized',
    burstable: 'Burstable',
    guaranteed: 'Guaranteed',
  }
  return labels[mode] || mode
}

const statusLabel = (status: string): string => {
  const labels: Record<string, string> = {
    pending: 'Pending',
    applied: 'Applied',
    rejected: 'Rejected',
    superseded: 'Superseded',
  }
  return labels[status] || status
}

const clearFilters = () => {
  filters.value = {
    namespace: null,
    status: null,
    mode: null,
    workloadType: null,
    workloadName: null,
  }
}
</script>
