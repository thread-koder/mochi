<template>
  <div
    ref="scope"
    v-bind="$attrs"
    @mousemove="handleMouseMove"
    @mouseleave="handleMouseLeave"
  >
    <slot />
  </div>
</template>

<script setup lang="ts">
import { useAnimate } from 'motion-v'

const [scope, animate] = useAnimate()

let currentAnimation: ReturnType<typeof animate> | null = null

const handleMouseMove = (e: MouseEvent) => {
  if (!scope.value || !import.meta.client) {
    return
  }

  const rect = scope.value.getBoundingClientRect()
  const centerX = rect.left + rect.width / 2
  const centerY = rect.top + rect.height / 2

  const offsetX = (e.clientX - centerX) / (rect.width / 2)
  const offsetY = (e.clientY - centerY) / (rect.height / 2)

  const maxOffset = 15
  const x = offsetX * maxOffset
  const y = offsetY * maxOffset

  if (currentAnimation) {
    currentAnimation.stop()
  }

  currentAnimation = animate(
    scope.value,
    { x, y, scale: 1.02 },
    { type: 'spring', stiffness: 300, damping: 30 },
  )
}

const handleMouseLeave = () => {
  if (!scope.value || !import.meta.client) {
    return
  }

  if (currentAnimation) {
    currentAnimation.stop()
  }

  currentAnimation = animate(
    scope.value,
    { x: 0, y: 0, scale: 1 },
    { type: 'spring', stiffness: 300, damping: 30 },
  )
}
</script>
