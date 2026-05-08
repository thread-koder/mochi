<template>
  <div class="glass rounded-xl p-6">
    <h2 class="text-2xl font-bold font-heading mb-4">
      Containers
    </h2>
    <UiEmptyState
      v-if="!containers || containers.length === 0"
      icon="lucide:box"
      title="No containers found"
      description="This workload doesn't have any containers."
    />
    <div
      v-else
      class="overflow-x-auto"
    >
      <table class="w-full">
        <thead>
          <tr class="border-b border-primary/20 text-sm text-on-surface-secondary">
            <th class="text-left py-2 px-4">
              Container
            </th>
            <th class="text-left py-2 px-4">
              Image
            </th>
            <th class="text-left py-2 px-4">
              CPU Request
            </th>
            <th class="text-left py-2 px-4">
              CPU Limit
            </th>
            <th class="text-left py-2 px-4">
              Memory Request
            </th>
            <th class="text-left py-2 px-4">
              Memory Limit
            </th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="container in containers"
            :key="container.name"
            class="border-b border-primary/20 hover:bg-primary/10 transition-all"
          >
            <td class="py-3 px-4">
              <span class="font-medium text-on-surface">{{ container.name }}</span>
            </td>
            <td class="py-3 px-4">
              <span class="text-on-surface-secondary text-sm">{{ container.image }}</span>
            </td>
            <td class="py-3 px-4">
              <span class="text-on-surface-secondary text-sm">{{ container.cpu_request || 'N/A' }}</span>
            </td>
            <td class="py-3 px-4">
              <span class="text-on-surface-secondary text-sm">{{ container.cpu_limit || 'N/A' }}</span>
            </td>
            <td class="py-3 px-4">
              <span class="text-on-surface-secondary text-sm">{{ container.memory_request || 'N/A' }}</span>
            </td>
            <td class="py-3 px-4">
              <span class="text-on-surface-secondary text-sm">{{ container.memory_limit || 'N/A' }}</span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Container } from '#shared/types/workload'

defineProps<{
  containers?: Container[]
}>()
</script>
