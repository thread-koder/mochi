<template>
  <div class="p-8">
    <!-- Header -->
    <div class="mb-6">
      <UiBreadcrumb :items="breadcrumbs" />
      <h1 class="text-4xl font-bold font-heading mb-2">
        Compute Recommendations
      </h1>
    </div>

    <ComputeRecommendationsFilterBar v-model="filters" />
    <ComputeRecommendationsList :filters="filters" />
  </div>
</template>

<script setup lang="ts">
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
</script>
