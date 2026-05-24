<template>
  <div class="flex items-center space-x-2 text-sm mb-4">
    <template
      v-for="(item, index) in items"
      :key="`${item.label}-${index}`"
    >
      <Icon
        v-if="index > 0"
        name="lucide:chevron-right"
        class="text-xs text-on-surface-muted"
      />
      <NuxtLink
        v-if="item.to"
        :to="item.to"
        class="text-primary-light hover:text-primary transition-colors"
      >
        {{ item.label }}
      </NuxtLink>
      <span
        v-else
        class="text-on-surface"
      >
        {{ item.label }}
      </span>
    </template>
  </div>
</template>

<script setup lang="ts">
export type BreadcrumbItem = {
  label: string
  to?: string
}

defineProps<{
  items: BreadcrumbItem[]
}>()
</script>
