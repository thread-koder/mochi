<template>
  <div class="glass rounded-xl p-6">
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
      class="py-12 flex flex-col items-center justify-center gap-3 text-on-surface-secondary"
    >
      <Icon
        name="lucide:loader-circle"
        class="text-3xl animate-spin"
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
              class="text-left py-2 px-4 cursor-pointer select-none hover:text-primary-light transition-colors"
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
              class="text-left py-2 px-4 cursor-pointer select-none hover:text-primary-light transition-colors"
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
              class="text-left py-2 px-4 cursor-pointer select-none hover:text-primary-light transition-colors"
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
              class="text-left py-2 px-4 cursor-pointer select-none hover:text-primary-light transition-colors"
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
              class="text-left py-2 px-4 cursor-pointer select-none hover:text-primary-light transition-colors"
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
              class="text-left py-2 px-4 cursor-pointer select-none hover:text-primary-light transition-colors"
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
              class="text-left py-2 px-4 cursor-pointer select-none hover:text-primary-light transition-colors"
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
            class="border-b border-primary/20 hover:bg-primary/10 transition-all cursor-pointer"
            @click="navigateToDetail(record.id)"
          >
            <td class="py-3 px-4">
              <span class="text-on-surface text-sm">{{ record.namespace }}</span>
            </td>
            <td class="py-3 px-4">
              <span class="text-primary-light font-medium text-sm">{{ record.workload_name }}</span>
            </td>
            <td class="py-3 px-4">
              <span class="px-2 py-1 rounded-full text-xs font-medium bg-primary/20 text-primary-light border border-primary/30">
                {{ record.workload_type }}
              </span>
            </td>
            <td class="py-3 px-4">
              <span
                :class="statusBadgeClass(record.status)"
                class="px-2 py-1 rounded-full text-xs font-medium border"
              >
                {{ formatTitleCase(record.status) }}
              </span>
            </td>
            <td class="py-3 px-4">
              <span class="px-2 py-1 rounded-full text-xs font-medium bg-secondary/20 text-secondary-light border border-secondary/30">
                {{ formatTitleCase(record.recommendation_mode) }}
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
      :current-page="currentPage"
      :total-pages="totalPages"
      :total="total"
      :page-size="pageSize"
      @update:page="currentPage = $event"
    />
  </div>
</template>

<script setup lang="ts">
import type * as Compute from '#shared/types/compute'
import type { FilterState } from '~/components/Compute/Recommendations/FilterBar.vue'

const props = defineProps<{
  filters: FilterState
}>()

// Pagination settings
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

const { data: recommendations, pending, error } = useLazyAsyncData<Compute.RecommendationsResponse>(
  () => `compute-recommendations-${queryString.value}`,
  () => $api(`/api/v1/compute/recommendations?${queryString.value}`),
  { watch: [queryString] },
)

const records = computed(() => recommendations.value?.recommendations ?? [])
const total = computed(() => recommendations.value?.total ?? 0)
const totalPages = computed(() => Math.ceil(total.value / pageSize))

// Reset to page 1 when filters change
watch(() => props.filters, () => {
  currentPage.value = 1
}, { deep: true })

const averageConfidence = (record: Compute.RecommendationRecord): number => {
  const recs = record.recommendations ?? []
  if (recs.length === 0) return 0
  const sum = recs.reduce((acc, r) => acc + (r.confidence_score ?? 0), 0)
  return sum / recs.length
}

type SortColumn = 'namespace' | 'workload_name' | 'workload_type' | 'status' | 'mode' | 'confidence' | 'created_at'
const sortBy = ref<SortColumn | null>('created_at')
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

const sortedRecords = computed(() => {
  const list = records.value ?? []
  if (!sortBy.value) return list
  const dir = sortDir.value === 'asc' ? 1 : -1
  return [...list].sort((a, b) => {
    let aVal: string | number
    let bVal: string | number
    switch (sortBy.value!) {
      case 'namespace':
        aVal = a.namespace
        bVal = b.namespace
        break
      case 'workload_name':
        aVal = a.workload_name
        bVal = b.workload_name
        break
      case 'workload_type':
        aVal = a.workload_type
        bVal = b.workload_type
        break
      case 'status':
        aVal = a.status
        bVal = b.status
        break
      case 'mode':
        aVal = a.recommendation_mode
        bVal = b.recommendation_mode
        break
      case 'confidence':
        aVal = averageConfidence(a)
        bVal = averageConfidence(b)
        break
      case 'created_at':
        aVal = new Date(a.created_at).getTime()
        bVal = new Date(b.created_at).getTime()
        break
      default:
        return 0
    }
    if (typeof aVal === 'string' && typeof bVal === 'string') {
      return dir * aVal.localeCompare(bVal)
    }
    return dir * ((aVal as number) - (bVal as number))
  })
})

const navigateToDetail = (id: number) => {
  navigateTo(`/recommendations/compute/${id}`)
}
</script>
