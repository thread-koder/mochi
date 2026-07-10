<template>
  <div class="p-8">
    <!-- Header -->
    <div class="mb-6">
      <UiBreadcrumb :items="breadcrumbs" />
      <h1 class="text-4xl font-bold font-heading mb-2">
        Compute Recommendations
      </h1>
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
