<template>
  <div class="glass rounded-xl p-6 mb-6">
    <h2 class="text-2xl font-bold font-heading mb-4">
      Pods
    </h2>
    <UiEmptyState
      v-if="!pods || pods.length === 0"
      icon="lucide:layers"
      title="No pods found"
    />
    <div
      v-else
      class="space-y-2"
    >
      <div
        v-for="pod in pods"
        :key="pod.uid"
        class="flex items-center space-x-4 glass hover:bg-primary/10 rounded-lg p-4 transition-all"
      >
        <div class="flex-1">
          <h3 class="font-medium text-on-surface">
            {{ pod.name }}
          </h3>
          <div class="text-sm text-on-surface-subtle mt-1">
            <span>Phase: {{ pod.phase }}</span>
            <span
              v-if="pod.node"
              class="mx-2"
            >•</span>
            <span v-if="pod.node">Node: {{ pod.node }}</span>
            <span class="mx-2">•</span>
            <span>Created: {{ timeAgo(pod.created_at) }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type * as Workload from '#shared/types/workload'

defineProps<{
  pods?: Workload.WorkloadResponse['pods']
}>()
</script>
