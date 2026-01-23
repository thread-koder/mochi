<template>
  <div class="space-y-2">
    <button
      class="flex items-center justify-between w-full px-4 py-2
        rounded-lg hover:bg-primary/10 hover:text-on-surface
      text-on-surface-subtle cursor-pointer transition-all text-sm"
      @click="isOpen = !isOpen"
    >
      <div class="flex items-center space-x-3">
        <Icon
          v-if="icon"
          :name="icon"
        />
        <span class="font-medium">{{ title }}</span>
      </div>
      <Icon
        name="lucide:chevron-right"
        class="transition-transform duration-200"
        :class="{ 'rotate-90': isOpen }"
      />
    </button>

    <Transition
      enter-active-class="transition ease-out duration-200"
      enter-from-class="opacity-0 -translate-y-2"
      enter-to-class="opacity-100 translate-y-0"
      leave-active-class="transition ease-in duration-150"
      leave-from-class="opacity-100 translate-y-0"
      leave-to-class="opacity-0 -translate-y-2"
    >
      <div
        v-if="isOpen"
        class="pl-8 space-y-1 border-l border-primary/20 ml-4"
      >
        <slot />
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
const props = withDefaults(defineProps<{
  title: string
  icon?: string
  defaultOpen?: boolean
}>(), {
  defaultOpen: false,
})

const isOpen = ref(props.defaultOpen)
</script>
