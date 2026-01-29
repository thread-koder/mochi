<template>
  <div class="glass rounded-xl p-6 mb-6">
    <h2 class="text-2xl font-bold font-heading mb-4">
      {{ title }}
    </h2>
    <div
      v-if="!pods || pods.length === 0"
      class="text-base font-medium text-on-surface-secondary text-center py-12"
    >
      No pods found
    </div>
    <div
      v-else
      class="space-y-2"
    >
      <NuxtLink
        v-for="pod in pods"
        :key="pod.name"
        :to="`/namespaces/${namespace}/workloads/Pod/${pod.name}`"
        class="flex items-center space-x-4 glass hover:bg-primary/10 rounded-lg p-4
         transition-all text-on-surface-secondary hover:text-on-surface"
      >
        <span
          class="px-2 py-1 rounded text-xs"
          :class="badgeClass"
        >
          {{ badgeLabel }}
        </span>
        <div class="flex-1">
          <h3 class="font-medium">
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
</template>

<script setup lang="ts">
import type * as Namespace from '#shared/types/namespace'

defineProps<{
  pods?: Namespace.NamespaceResponse['standalone_pods'] | Namespace.NamespaceResponse['system_pods']
  namespace: string
  title: string
  badgeLabel: string
  badgeClass: string
}>()
</script>
