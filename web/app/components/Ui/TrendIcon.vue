<template>
  <template v-if="available">
    <Icon
      v-if="direction === 'increasing'"
      name="lucide:trending-up"
      :class="iconClass"
    />
    <Icon
      v-else-if="direction === 'decreasing'"
      name="lucide:trending-down"
      :class="iconClass"
    />
    <Icon
      v-else
      name="lucide:arrow-right"
      :class="iconClass"
    />
  </template>
  <span
    v-else
    class="text-xs text-on-surface-muted"
  >N/A</span>
</template>

<script setup lang="ts">
const props = withDefaults(
  defineProps<{
    direction?: string
    available?: boolean
  }>(),
  {
    direction: 'stable',
    available: false,
  },
)

const iconClass = computed(() => {
  if (props.direction === 'increasing') {
    return 'text-xs text-error-light'
  }
  if (props.direction === 'decreasing') {
    return 'text-xs text-success-light'
  }
  return 'text-xs text-on-surface-secondary'
})
</script>
