<template>
  <section class="mb-6">
    <h2 class="text-2xl font-bold font-heading mb-4">
      {{ title }}
    </h2>
    <div class="panel p-4">
      <UiEmptyState
        v-if="pods.length === 0"
        icon="lucide:layers"
        title="No pods found"
        :description="`No ${title.toLowerCase()} found in this namespace.`"
      />
      <div v-else>
        <NuxtLink
          v-for="pod in pods"
          :key="pod.name"
          :to="`/namespaces/${namespace}/workloads/Pod/${pod.name}`"
          class="flex items-center gap-4 py-3 px-2 border-b border-primary/10 last:border-b-0
           text-on-surface-secondary hover:text-on-surface hover:bg-primary/10 transition-colors"
        >
          <div class="flex-1 min-w-0">
            <h3 class="font-medium truncate">
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
        </NuxtLink>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import type { StandalonePod } from '#shared/types/namespace'

defineProps<{
  pods: StandalonePod[]
  namespace: string
  title: string
}>()
</script>
