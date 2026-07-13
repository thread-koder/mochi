<template>
  <div class="panel p-4">
    <!-- Error state -->
    <UiAlert
      v-if="error"
      variant="error"
      title="Error loading recommendations"
      :description="parseError(error, 'Failed to load recommendations').message"
    />

    <!-- Loading state -->
    <div
      v-else-if="pending"
      class="py-6 flex flex-col items-center justify-center gap-2 text-on-surface-secondary"
    >
      <Icon
        name="lucide:loader-circle"
        class="text-2xl animate-spin"
      />
      <p class="text-sm font-medium">
        Loading recommendations...
      </p>
    </div>

    <!-- Empty state -->
    <UiEmptyState
      v-else-if="records.length === 0"
      icon="lucide:inbox"
      title="No recommendations found"
      description="Adjust filters or generate recommendations for your workloads."
    />

    <!-- Table -->
    <div
      v-else
      class="overflow-x-auto"
    >
      <table class="w-full">
        <thead>
          <tr class="border-b border-primary/20 text-sm text-on-surface-secondary">
            <th
              class="text-left py-2 px-4 cursor-pointer select-none hover:text-on-surface transition-colors"
              @click="setSort('namespace')"
            >
              <span class="inline-flex items-center gap-1">
                Namespace
                <Icon
                  v-if="sortBy === 'namespace'"
                  :name="sortDir === 'asc' ? 'lucide:chevron-up' : 'lucide:chevron-down'"
                  class="text-xs"
                />
              </span>
            </th>
            <th
              class="text-left py-2 px-4 cursor-pointer select-none hover:text-on-surface transition-colors"
              @click="setSort('workload_name')"
            >
              <span class="inline-flex items-center gap-1">
                Workload Name
                <Icon
                  v-if="sortBy === 'workload_name'"
                  :name="sortDir === 'asc' ? 'lucide:chevron-up' : 'lucide:chevron-down'"
                  class="text-xs"
                />
              </span>
            </th>
            <th
              class="text-left py-2 px-4 cursor-pointer select-none hover:text-on-surface transition-colors"
              @click="setSort('workload_type')"
            >
              <span class="inline-flex items-center gap-1">
                Workload Type
                <Icon
                  v-if="sortBy === 'workload_type'"
                  :name="sortDir === 'asc' ? 'lucide:chevron-up' : 'lucide:chevron-down'"
                  class="text-xs"
                />
              </span>
            </th>
            <th
              class="text-left py-2 px-4 cursor-pointer select-none hover:text-on-surface transition-colors"
              @click="setSort('status')"
            >
              <span class="inline-flex items-center gap-1">
                Status
                <Icon
                  v-if="sortBy === 'status'"
                  :name="sortDir === 'asc' ? 'lucide:chevron-up' : 'lucide:chevron-down'"
                  class="text-xs"
                />
              </span>
            </th>
            <th
              class="text-left py-2 px-4 cursor-pointer select-none hover:text-on-surface transition-colors"
              @click="setSort('mode')"
            >
              <span class="inline-flex items-center gap-1">
                Mode
                <Icon
                  v-if="sortBy === 'mode'"
                  :name="sortDir === 'asc' ? 'lucide:chevron-up' : 'lucide:chevron-down'"
                  class="text-xs"
                />
              </span>
            </th>
            <th
              class="text-left py-2 px-4 cursor-pointer select-none hover:text-on-surface transition-colors"
              @click="setSort('confidence')"
            >
              <span class="inline-flex items-center gap-1">
                Confidence
                <Icon
                  v-if="sortBy === 'confidence'"
                  :name="sortDir === 'asc' ? 'lucide:chevron-up' : 'lucide:chevron-down'"
                  class="text-xs"
                />
              </span>
            </th>
            <th
              class="text-left py-2 px-4 cursor-pointer select-none hover:text-on-surface transition-colors"
              @click="setSort('created_at')"
            >
              <span class="inline-flex items-center gap-1">
                Created At
                <Icon
                  v-if="sortBy === 'created_at'"
                  :name="sortDir === 'asc' ? 'lucide:chevron-up' : 'lucide:chevron-down'"
                  class="text-xs"
                />
              </span>
            </th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="record in sortedRecords"
            :key="record.id"
            class="border-b border-primary/10 hover:bg-primary/10 transition-all cursor-pointer last:border-b-0"
            @click="navigateToDetail(record.id)"
          >
            <td class="py-3 px-4">
              <span class="text-on-surface text-sm">{{ record.namespace }}</span>
            </td>
            <td class="py-3 px-4">
              <span class="text-on-surface font-medium text-sm">{{ record.workload_name }}</span>
            </td>
            <td class="py-3 px-4">
              <span
                class="badge-neutral px-2 py-1 rounded-full text-xs font-medium border"
              >
                {{ workloadTypeLabel(record.workload_type) }}
              </span>
            </td>
            <td class="py-3 px-4">
              <span
                :class="recommendationStatusBadgeClass(record.status)"
                class="px-2 py-1 rounded-full text-xs font-medium border"
              >
                {{ recommendationStatusLabel(record.status) }}
              </span>
            </td>
            <td class="py-3 px-4">
              <span
                class="badge-neutral px-2 py-1 rounded-full text-xs font-medium border"
              >
                {{ recommendationModeLabel(record.recommendation_mode) }}
              </span>
            </td>
            <td class="py-3 px-4">
              <span
                :class="scoreBadgeClass(averageConfidence(record))"
                class="px-2 py-1 rounded-full text-xs font-medium border"
              >
                {{ formatPercentage(averageConfidence(record)) }}
              </span>
            </td>
            <td class="py-3 px-4">
              <span class="text-on-surface-secondary text-sm">{{ timeAgo(record.created_at) }}</span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Pagination -->
    <UiPagination
      v-if="!error && !pending && total > 0"
      class="mt-4"
      :current-page="currentPage"
      :total-pages="totalPages"
      :total="total"
      :page-size="pageSize"
      @update:page="currentPage = $event"
    />
  </div>
</template>

<script setup lang="ts">
import {
  recommendationModeLabel,
  recommendationStatusLabel,
} from '#shared/constants/compute/recommendations'
import { workloadTypeLabel } from '#shared/constants/workload'
import { recommendationStatusBadgeClass } from '#shared/utils/compute/color'
import { scoreBadgeClass } from '#shared/utils/color'
import type { RecommendationsResponse, RecommendationRecord } from '#shared/types/compute'
import type { FilterState } from '~/components/Compute/Recommendations/FilterBar.vue'

const props = defineProps<{
  filters: FilterState
}>()

const pageSize = 20
const currentPage = ref(1)

const buildQuery = (filters: FilterState, page: number): string => {
  const params = new URLSearchParams()
  params.set('limit', pageSize.toString())
  params.set('offset', ((page - 1) * pageSize).toString())
  if (filters.namespace) params.set('namespace', filters.namespace)
  if (filters.status) params.set('status', filters.status)
  if (filters.mode) params.set('mode', filters.mode)
  if (filters.workloadType) params.set('workloadType', filters.workloadType)
  if (filters.workloadName) params.set('workloadName', filters.workloadName)
  return params.toString()
}

const queryString = computed(() => buildQuery(props.filters, currentPage.value))

const { parseError } = useApiError()

const { data: recommendations, pending, error } = useApiData<RecommendationsResponse>(
  () => `/api/v1/compute/recommendations?${queryString.value}`,
  { lazy: true, watch: [queryString] },
)

const records = computed(() => recommendations.value?.recommendations ?? [])
const total = computed(() => recommendations.value?.total ?? 0)
const totalPages = computed(() => Math.ceil(total.value / pageSize))

watch(() => props.filters, () => {
  currentPage.value = 1
}, { deep: true })

const averageConfidence = (record: RecommendationRecord): number => {
  const recs = record.recommendations
  if (recs.length === 0) return 0
  const sum = recs.reduce((acc, r) => acc + r.confidence, 0)
  return sum / recs.length
}

type SortColumn = 'namespace' | 'workload_name' | 'workload_type' | 'status' | 'mode' | 'confidence' | 'created_at'

const sortBy = ref<SortColumn>('created_at')
const sortDir = ref<'asc' | 'desc'>('desc')

const setSort = (column: SortColumn) => {
  if (sortBy.value === column) {
    sortDir.value = sortDir.value === 'asc' ? 'desc' : 'asc'
  }
  else {
    sortBy.value = column
    sortDir.value = 'desc'
  }
}

const sortValue = (record: RecommendationRecord, column: SortColumn): string | number => {
  switch (column) {
    case 'namespace':
      return record.namespace
    case 'workload_name':
      return record.workload_name
    case 'workload_type':
      return record.workload_type
    case 'status':
      return record.status
    case 'mode':
      return record.recommendation_mode
    case 'confidence':
      return averageConfidence(record)
    case 'created_at':
      return new Date(record.created_at).getTime()
  }
}

const sortedRecords = computed(() => {
  const list = records.value
  const column = sortBy.value
  const dir = sortDir.value === 'asc' ? 1 : -1

  const mapped = list.map(record => ({
    record,
    value: sortValue(record, column),
  }))

  mapped.sort((a, b) => {
    const aVal = a.value
    const bVal = b.value
    if (typeof aVal === 'string' && typeof bVal === 'string') {
      return dir * aVal.localeCompare(bVal)
    }
    return dir * ((aVal as number) - (bVal as number))
  })

  return mapped.map(item => item.record)
})

const navigateToDetail = (id: string) => {
  navigateTo(`/recommendations/compute/${id}`)
}
</script>
