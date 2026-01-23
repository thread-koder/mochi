<template>
  <div class="mt-6">
    <!-- Stats Cards -->
    <div class="grid grid-cols-1 md:grid-cols-2 gap-6 mb-8">
      <UiStatsCard
        title="Pods"
        :value="workloadData?.stats.pods ?? 0"
        icon="lucide:rocket"
        color="text-primary-light"
      />
      <UiStatsCard
        title="Containers"
        :value="workloadData?.stats.containers ?? 0"
        icon="lucide:container"
        color="text-secondary-light"
      />
    </div>

    <!-- Pods List -->
    <WorkloadPodsList :pods="workloadData?.pods" />

    <!-- Containers Table -->
    <WorkloadContainersTable :containers="workloadData?.containers" />
  </div>
</template>

<script setup lang="ts">
import type * as Workload from '#shared/types/workload'

defineProps<{
  workloadData?: Workload.WorkloadResponse
}>()
</script>
