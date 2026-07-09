<template>
  <UiMagnetic class="panel p-4 magnetic-card">
    <div class="flex items-center justify-between mb-2">
      <h3 class="text-on-surface-secondary text-sm">
        {{ title }}
      </h3>
      <Icon
        v-if="icon"
        :name="icon"
        class="text-lg text-primary-light"
      />
    </div>
    <p class="text-2xl font-bold text-on-surface">
      {{ displayValue }}
    </p>
  </UiMagnetic>
</template>

<script setup lang="ts">
const props = defineProps<{
  title: string
  value: number
  icon?: string
}>()

const displayValue = ref(0)
let animationInterval: ReturnType<typeof setInterval> | null = null

const animateCounter = (target: number, intervalMs = 50) => {
  if (animationInterval) {
    clearInterval(animationInterval)
    animationInterval = null
  }

  if (target === 0) {
    displayValue.value = target
    return
  }

  animationInterval = setInterval(() => {
    if (displayValue.value < target) {
      displayValue.value++
    }
    else {
      if (animationInterval) {
        clearInterval(animationInterval)
        animationInterval = null
      }
    }
  }, intervalMs)
}

onBeforeMount(() => {
  animateCounter(props.value)
})

onUnmounted(() => {
  if (animationInterval) {
    clearInterval(animationInterval)
    animationInterval = null
  }
})
</script>
