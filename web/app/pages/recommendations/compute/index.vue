<template>
  <div class="p-8">
    <!-- Header -->
    <div class="mb-8">
      <!-- Breadcrumb -->
      <div class="flex items-center space-x-2 text-sm mb-4">
        <NuxtLink
          to="/"
          class="text-primary-light hover:text-primary transition-colors"
        >
          Home
        </NuxtLink>
        <Icon
          name="lucide:chevron-right"
          class="text-xs text-on-surface-muted"
        />
        <NuxtLink
          to="/recommendations/compute"
          class="text-primary-light hover:text-primary transition-colors"
        >
          Recommendations
        </NuxtLink>
        <Icon
          name="lucide:chevron-right"
          class="text-xs text-on-surface-muted"
        />
        <span class="text-on-surface">Compute</span>
      </div>

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
      :namespaces="namespacesData"
    />

    <!-- Recommendations List -->
    <ComputeRecommendationsList :filters="filters" />
  </div>
</template>

<script setup lang="ts">
import type * as Namespace from '#shared/types/namespace'
import type { FilterState } from '~/components/Compute/Recommendations/FilterBar.vue'

const filters = ref<FilterState>({
  namespace: null,
  status: null,
  mode: null,
  workloadType: null,
  workloadName: null,
})

// Fetch namespaces for the filter dropdown
const { data: namespacesData } = await useApiData<Namespace.Namespace[]>('/api/v1/namespaces')
</script>
