<template>
  <div class="mt-6">
    <!-- Stats Cards -->
    <div class="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
      <UiStatsCard
        title="Workloads"
        :value="nsData?.stats.workloads ?? 0"
        icon="lucide:server"
        color="text-primary-light"
      />
      <UiStatsCard
        title="Pods"
        :value="nsData?.stats.pods ?? 0"
        icon="lucide:rocket"
        color="text-secondary-light"
      />
      <UiStatsCard
        title="Containers"
        :value="nsData?.stats.containers ?? 0"
        icon="lucide:container"
        color="text-tertiary-light"
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
      badge-label="Pod"
      badge-class="bg-secondary/20 text-secondary-light"
    />

    <!-- System Pods -->
    <NamespacePodsList
      v-if="nsData?.system_pods && nsData.system_pods.length > 0"
      :pods="nsData.system_pods"
      :namespace="nsData.name"
      title="System Pods"
      badge-label="System"
      badge-class="bg-warning/20 text-warning-light"
    />
  </div>
</template>

<script setup lang="ts">
import type * as Namespace from '#shared/types/namespace'

defineProps<{
  nsData?: Namespace.NamespaceResponse
}>()
</script>
