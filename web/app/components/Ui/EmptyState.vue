<template>
  <div class="py-12 px-6 rounded-lg bg-primary/5 border border-primary/10 text-center">
    <Icon
      :name="iconName"
      class="mx-auto text-4xl text-on-surface-muted mb-4"
    />
    <p class="text-sm font-medium text-on-surface mb-1">
      {{ title }}
    </p>
    <p
      v-if="description"
      class="text-xs text-on-surface-muted"
    >
      {{ description }}
    </p>
  </div>
</template>

<script setup lang="ts">
const props = withDefaults(
  defineProps<{
    icon?: string
    title: string
    description?: string
  }>(),
  {
    icon: undefined,
    description: undefined,
  },
)

const iconName = computed(() => props.icon ?? 'lucide:inbox')
</script>
