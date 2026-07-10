<template>
  <section class="mb-6">
    <h2 class="text-2xl font-bold font-heading mb-4">
      Workloads
    </h2>
    <div class="panel p-4">
      <UiEmptyState
        v-if="!workloads || workloads.length === 0"
        icon="lucide:server"
        title="No workloads found"
        description="This namespace doesn't have any workloads yet."
      />
      <div v-else>
        <NuxtLink
          v-for="workload in workloads"
          :key="`${workload.type}-${workload.name}`"
          :to="`/namespaces/${namespace}/workloads/${workload.type}/${workload.name}`"
          class="flex items-center gap-4 py-3 px-2 border-b border-primary/10 last:border-b-0
           text-on-surface-secondary hover:text-on-surface hover:bg-primary/10 transition-colors"
        >
          <span
            class="badge-neutral px-2 py-1 rounded-full text-xs font-medium border shrink-0 w-28 text-center"
          >
            {{ workloadTypeLabel(workload.type) }}
          </span>
          <div class="flex-1 min-w-0">
            <h3 class="font-medium truncate">
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
  </section>
</template>

<script setup lang="ts">
import { workloadTypeLabel } from '#shared/constants/workload'
import type { Workload } from '#shared/types/namespace'

defineProps<{
  workloads?: Workload[]
  namespace: string
}>()
</script>
