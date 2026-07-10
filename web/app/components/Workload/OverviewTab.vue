<template>
  <div>
    <!-- Stats Cards -->
    <div class="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">
      <UiStatsCard
        title="Pods"
        :value="workloadData?.stats.pods ?? 0"
        icon="lucide:rocket"
      />
      <UiStatsCard
        title="Containers"
        :value="workloadData?.stats.containers ?? 0"
        icon="lucide:container"
      />
    </div>

    <!-- Pods List -->
    <WorkloadPodsList :pods="workloadData?.pods" />

    <!-- Containers Table -->
    <WorkloadContainersTable :containers="workloadData?.containers" />
  </div>
</template>

<script setup lang="ts">
import type { WorkloadResponse } from '#shared/types/workload'

defineProps<{
  workloadData?: WorkloadResponse
}>()
</script>
