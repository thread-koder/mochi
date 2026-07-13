<template>
  <div>
    <div class="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">
      <UiStatsCard
        title="Pods"
        :value="workload.stats.pods"
        icon="lucide:rocket"
      />
      <UiStatsCard
        title="Containers"
        :value="workload.stats.containers"
        icon="lucide:container"
      />
    </div>
    <WorkloadPodsList :pods="workload.pods" />
    <WorkloadContainersTable :containers="workload.containers" />
  </div>
</template>

<script setup lang="ts">
import type { WorkloadResponse } from '#shared/types/workload'

defineProps<{
  workload: WorkloadResponse
}>()
</script>
