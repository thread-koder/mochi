<template>
  <div class="glass rounded-xl p-6 mb-6">
    <h2 class="text-2xl font-bold font-heading mb-4">
      Workloads
    </h2>
    <UiEmptyState
      v-if="!workloads || workloads.length === 0"
      icon="lucide:server"
      title="No workloads found"
      description="This namespace doesn't have any workloads yet."
    />
    <div
      v-else
      class="space-y-2"
    >
      <NuxtLink
        v-for="workload in workloads"
        :key="`${workload.type}-${workload.name}`"
        :to="`/namespaces/${namespace}/workloads/${workload.type}/${workload.name}`"
        class="flex items-center space-x-4 glass hover:bg-primary/10 rounded-lg p-4 transition-all text-on-surface-secondary hover:text-on-surface"
      >
        <span class="px-2 py-1 rounded-full text-xs font-medium bg-primary/20 text-primary-light border border-primary/30">
          {{ workload.type }}
        </span>
        <div class="flex-1">
          <h3 class="font-medium">
            {{ workload.name }}
          </h3>
          <div class="text-sm text-on-surface-subtle mt-1">
            <span>Replicas: {{ workload.ready }}/{{ workload.replicas }}</span>
            <span class="mx-2">•</span>
            <span>Created: {{ timeAgo(workload.created_at) }}</span>
          </div>
        </div>
      </NuxtLink>
    </div>
  </div>
</template>

<script setup lang="ts">
import type * as Namespace from '#shared/types/namespace'

defineProps<{
  workloads?: Namespace.NamespaceResponse['workloads']
  namespace: string
}>()
</script>
