<template>
  <div class="mb-6 space-y-4">
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
          :placeholder="namespacePlaceholder"
          :empty-message="namespaceEmptyMessage"
          :disabled="namespacesPending"
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
          :options="RECOMMENDATION_STATUS_OPTIONS"
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
          :options="RECOMMENDATION_MODE_OPTIONS"
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
          :options="WORKLOAD_TYPE_OPTIONS"
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
          class="bg-surface-elevated border border-on-surface-muted/20 rounded-lg px-3 py-2
             text-sm text-on-surface-secondary placeholder:text-on-surface-muted
             focus:outline-none focus:border-primary/50 min-w-45"
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
        class="flex items-center gap-3 flex-wrap"
      >
        <!-- Active Filter Badges -->
        <TransitionGroup
          tag="div"
          class="flex items-center gap-2 flex-wrap flex-1"
          enter-active-class="transition ease-out duration-200"
          enter-from-class="opacity-0 scale-90"
          enter-to-class="opacity-100 scale-100"
          leave-active-class="transition ease-in duration-150"
          leave-from-class="opacity-100 scale-100"
          leave-to-class="opacity-0 scale-90"
        >
          <span
            v-for="filter in activeFilters"
            :key="filter.key"
            class="badge-neutral inline-flex items-center gap-1.5 px-2 py-1 rounded-full text-xs font-medium border"
          >
            <Icon
              :name="filter.icon"
              class="text-xs shrink-0"
            />
            <span>{{ filter.label }}</span>
            <button
              class="text-on-surface-muted hover:text-on-surface transition-colors shrink-0 flex items-center cursor-pointer"
              @click="clearFilter(filter.key)"
            >
              <Icon
                name="lucide:x"
                class="text-xs"
              />
            </button>
          </span>
        </TransitionGroup>

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
</template>

<script setup lang="ts">
import {
  RECOMMENDATION_MODE_OPTIONS,
  RECOMMENDATION_STATUS_OPTIONS,
  recommendationModeLabel,
  recommendationStatusLabel,
} from '#shared/constants/compute/recommendations'
import { WORKLOAD_TYPE_OPTIONS, workloadTypeLabel } from '#shared/constants/workload'
import type { Namespace } from '#shared/types/namespace'

export interface FilterState {
  namespace: string | null
  status: string | null
  mode: string | null
  workloadType: string | null
  workloadName: string | null
}

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

const { parseError } = useApiError()
const { data: namespaces, error: namespacesError, pending: namespacesPending }
  = useApiData<Namespace[]>('/api/v1/namespaces', { lazy: true })

const namespaceOptions = computed(() => {
  return (namespaces.value ?? []).map((ns: Namespace) => ns.name)
})

const namespacePlaceholder = computed(() => {
  if (namespacesPending.value) {
    return 'Loading namespaces...'
  }
  if (namespacesError.value) {
    return 'Namespaces unavailable'
  }
  return 'All Namespaces'
})

const namespaceEmptyMessage = computed(() => {
  if (namespacesPending.value) {
    return 'Loading namespaces...'
  }
  if (namespacesError.value) {
    return parseError(namespacesError.value, 'Failed to load namespaces').message
  }
  return 'No namespaces found'
})

const workloadNameInput = ref<string>('')

const updateWorkloadNameFilter = useDebounceFn((value: string) => {
  filters.value.workloadName = value || null
}, 500)

watch(workloadNameInput, (value) => {
  updateWorkloadNameFilter(value)
})

watch(() => filters.value.workloadName, (value) => {
  if (value !== workloadNameInput.value) {
    workloadNameInput.value = value || ''
  }
}, { immediate: true })

const hasActiveFilters = computed(() => activeFilters.value.length > 0)

const activeFilters = computed(() => {
  const badges: { key: keyof FilterState, icon: string, label: string }[] = []

  if (filters.value.namespace) {
    badges.push({
      key: 'namespace',
      icon: 'lucide:layers',
      label: filters.value.namespace,
    })
  }
  if (filters.value.status) {
    badges.push({
      key: 'status',
      icon: 'lucide:circle-check',
      label: recommendationStatusLabel(filters.value.status),
    })
  }
  if (filters.value.mode) {
    badges.push({
      key: 'mode',
      icon: 'lucide:settings',
      label: recommendationModeLabel(filters.value.mode),
    })
  }
  if (filters.value.workloadType) {
    badges.push({
      key: 'workloadType',
      icon: 'lucide:server',
      label: workloadTypeLabel(filters.value.workloadType),
    })
  }
  if (filters.value.workloadName) {
    badges.push({
      key: 'workloadName',
      icon: 'lucide:search',
      label: filters.value.workloadName,
    })
  }

  return badges
})

const clearFilter = (key: keyof FilterState) => {
  filters.value[key] = null
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
