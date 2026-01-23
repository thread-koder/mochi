<template>
  <UiMagnetic class="glass hover:border-primary/50 rounded-xl p-6 magnetic-card">
    <div class="flex items-center justify-between mb-4">
      <h3 class="text-on-surface-secondary text-sm">
        {{ title }}
      </h3>
      <Icon
        v-if="icon"
        :name="icon"
        class="text-xl"
      />
    </div>
    <p
      class="text-3xl font-bold"
      :class="color"
    >
      {{ displayValue }}{{ trailing }}
    </p>
  </UiMagnetic>
</template>

<script setup lang="ts">
const props = defineProps<{
  title: string
  value: number
  icon?: string
  color: string
  trailing?: string
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
