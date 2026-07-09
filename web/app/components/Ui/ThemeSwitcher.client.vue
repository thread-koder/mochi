<template>
  <button
    class="cursor-pointer flex items-center justify-center p-1 rounded-lg
      text-on-surface-muted hover:text-primary transition-colors"
    aria-label="Toggle theme"
    @click="toggleMode()"
  >
    <Icon
      :name="colorMode === 'dark' ? 'lucide:moon' : 'lucide:sun'"
      class="text-xl"
    />
  </button>
</template>

<script setup lang="ts">
const colorMode = useColorMode({ storageKey: 'mochi-theme' })

const toggleMode = () => {
  colorMode.value = colorMode.value === 'dark' ? 'light' : 'dark'
}
</script>
