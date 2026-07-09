<template>
  <div class="text-center py-6">
    <Icon
      :name="iconName"
      class="mx-auto text-on-surface-muted text-3xl mb-2"
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
