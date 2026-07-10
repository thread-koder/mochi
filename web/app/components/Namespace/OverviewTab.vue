<template>
  <div>
    <!-- Stats Cards -->
    <div class="grid grid-cols-1 md:grid-cols-3 gap-6 mb-6">
      <UiStatsCard
        title="Workloads"
        :value="nsData?.stats.workloads ?? 0"
        icon="lucide:server"
      />
      <UiStatsCard
        title="Pods"
        :value="nsData?.stats.pods ?? 0"
        icon="lucide:rocket"
      />
      <UiStatsCard
        title="Containers"
        :value="nsData?.stats.containers ?? 0"
        icon="lucide:container"
      />
    </div>

    <!-- Workloads List -->
    <NamespaceWorkloadsList
      :workloads="nsData?.workloads"
      :namespace="nsData?.name ?? ''"
    />

    <!-- Standalone Pods -->
    <NamespacePodsList
      v-if="nsData?.standalone_pods && nsData.standalone_pods.length > 0"
      :pods="nsData.standalone_pods"
      :namespace="nsData.name"
      title="Standalone Pods"
    />

    <!-- System Pods -->
    <NamespacePodsList
      v-if="nsData?.system_pods && nsData.system_pods.length > 0"
      :pods="nsData.system_pods"
      :namespace="nsData.name"
      title="System Pods"
    />
  </div>
</template>

<script setup lang="ts">
import type { NamespaceResponse } from '#shared/types/namespace'

defineProps<{
  nsData?: NamespaceResponse
}>()
</script>
