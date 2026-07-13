<template>
  <div>
    <div class="grid grid-cols-1 md:grid-cols-3 gap-6 mb-6">
      <UiStatsCard
        title="Workloads"
        :value="namespace.stats.workloads"
        icon="lucide:server"
      />
      <UiStatsCard
        title="Pods"
        :value="namespace.stats.pods"
        icon="lucide:rocket"
      />
      <UiStatsCard
        title="Containers"
        :value="namespace.stats.containers"
        icon="lucide:container"
      />
    </div>
    <NamespaceWorkloadsList
      :workloads="namespace.workloads"
      :namespace="namespace.name"
    />
    <NamespacePodsList
      v-if="namespace.standalone_pods.length > 0"
      :pods="namespace.standalone_pods"
      :namespace="namespace.name"
      title="Standalone Pods"
    />
    <NamespacePodsList
      v-if="namespace.system_pods.length > 0"
      :pods="namespace.system_pods"
      :namespace="namespace.name"
      title="System Pods"
    />
  </div>
</template>

<script setup lang="ts">
import type { NamespaceResponse } from '#shared/types/namespace'

defineProps<{
  namespace: NamespaceResponse
}>()
</script>
