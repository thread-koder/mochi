<template>
  <div class="p-8">
    <!-- Header -->
    <div class="mb-8">
      <UiBreadcrumb :items="breadcrumbs" />
      <div class="flex items-center justify-between mb-4">
        <div>
          <h1 class="text-4xl font-bold font-heading mb-2">
            Compute Recommendations
          </h1>
          <p class="text-on-surface-secondary">
            Review and manage resource optimization recommendations for your workloads
          </p>
        </div>
      </div>
    </div>

    <!-- Filter Bar -->
    <ComputeRecommendationsFilterBar
      v-model="filters"
      :namespaces="namespaces"
    />

    <!-- Recommendations List -->
    <ComputeRecommendationsList :filters="filters" />
  </div>
</template>

<script setup lang="ts">
import type { Namespace } from '#shared/types/namespace'
import type { FilterState } from '~/components/Compute/Recommendations/FilterBar.vue'

const breadcrumbs = [
  { label: 'Home', to: '/' },
  { label: 'Recommendations', to: '/recommendations/compute' },
  { label: 'Compute' },
]

const filters = ref<FilterState>({
  namespace: null,
  status: null,
  mode: null,
  workloadType: null,
  workloadName: null,
})

const { data: namespaces } = await useApiData<Namespace[]>('/api/v1/namespaces')
</script>
