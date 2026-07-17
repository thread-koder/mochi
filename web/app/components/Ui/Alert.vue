<template>
  <div
    :class="[
      'p-4 rounded-lg flex items-start gap-3',
      variantClasses.container,
    ]"
  >
    <Icon
      :name="iconName"
      :class="[
        'text-xl shrink-0 mt-0.5',
        variantClasses.icon,
      ]"
    />
    <div class="flex-1">
      <p
        :class="[
          'text-sm font-medium mb-1',
          variantClasses.text,
        ]"
      >
        {{ title }}
      </p>
      <p
        v-if="description"
        :class="[
          'text-xs',
          variantClasses.text,
        ]"
      >
        {{ description }}
      </p>
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
type AlertVariant = 'success' | 'error' | 'warning' | 'info'

const props = withDefaults(
  defineProps<{
    variant?: AlertVariant
    icon?: string
    title: string
    description?: string
  }>(),
  {
    variant: 'info',
    icon: undefined,
    description: undefined,
  },
)

const iconName = computed(() => {
  if (props.icon) return props.icon
  switch (props.variant) {
    case 'success':
      return 'lucide:circle-check'
    case 'error':
      return 'lucide:circle-alert'
    case 'warning':
      return 'lucide:triangle-alert'
    default:
      return 'lucide:info'
  }
})

const variantClasses = computed(() => {
  switch (props.variant) {
    case 'success':
      return {
        container: 'bg-success/10 border border-success/20',
        icon: 'text-success-light',
        text: 'text-success-light',
      }
    case 'error':
      return {
        container: 'bg-error/10 border border-error/20',
        icon: 'text-error-light',
        text: 'text-error-light',
      }
    case 'warning':
      return {
        container: 'bg-warning/10 border border-warning/20',
        icon: 'text-warning-light',
        text: 'text-warning-light',
      }
    default:
      return {
        container: 'bg-info/10 border border-info/20',
        icon: 'text-info-light',
        text: 'text-info-light',
      }
  }
})
</script>
