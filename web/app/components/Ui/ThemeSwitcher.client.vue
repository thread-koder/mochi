<template>
  <button
    class="theme-switcher cursor-pointer flex items-center justify-center
      transition-all duration-500 hover:rotate-12 hover:scale-110"
    :class="{ 'rotate-180': colorMode === 'dark' }"
    aria-label="Toggle theme"
    @click="toggleMode()"
  >
    <Icon
      :name="colorMode === 'dark' ? 'lucide:moon' : 'lucide:sun'"
      class="text-2xl text-primary-light"
    />
  </button>
</template>

<script setup lang="ts">
const colorMode = useColorMode({ storageKey: 'mochi-theme' })

const toggleMode = () => {
  colorMode.value = colorMode.value === 'dark' ? 'light' : 'dark'
}
</script>
