<template>
  <section class="mb-6">
    <h2 class="text-2xl font-bold font-heading mb-4">
      Pods
    </h2>
    <div class="panel p-4">
      <UiEmptyState
        v-if="!pods || pods.length === 0"
        icon="lucide:layers"
        title="No pods found"
        description="This workload doesn't have any pods running."
      />
      <template v-else>
        <div
          v-for="pod in pods"
          :key="pod.uid"
          class="py-3 px-2 border-b border-primary/10 last:border-b-0"
        >
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
      </template>
    </div>
  </section>
</template>

<script setup lang="ts">
import type { Pod } from '#shared/types/workload'

defineProps<{
  pods?: Pod[]
}>()
</script>
