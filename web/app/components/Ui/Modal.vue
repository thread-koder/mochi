<template>
  <Teleport to="body">
    <Transition
      enter-active-class="transition ease-out duration-250"
      enter-from-class="opacity-0"
      enter-to-class="opacity-100"
      leave-active-class="transition ease-in duration-150"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <div
        v-show="isOpen"
        class="fixed inset-0 z-50 flex items-center justify-center backdrop-blur-sm"
      >
        <!-- Backdrop -->
        <div
          class="absolute inset-0 bg-black/50"
          @click.self="close"
        />

        <!-- Modal Content -->
        <Transition
          enter-active-class="transition ease-out duration-250"
          enter-from-class="opacity-0 scale-75"
          enter-to-class="opacity-100 scale-100"
          leave-active-class="transition ease-in duration-150"
          leave-from-class="opacity-100 scale-100"
          leave-to-class="opacity-0 scale-75"
        >
          <div
            v-show="isOpen"
            class="relative z-10 glass rounded-xl p-4 w-full max-w-lg max-h-[90vh] overflow-y-auto"
            role="dialog"
            aria-modal="true"
            v-bind="$attrs"
            :aria-labelledby="title"
          >
            <!-- Header -->
            <div
              class="flex items-center justify-between mb-6"
            >
              <h2
                class="text-2xl font-bold font-heading text-on-surface"
              >
                {{ title }}
              </h2>
              <Icon
                v-if="showClose"
                name="lucide:x"
                class="ml-auto text-xl text-on-surface-muted hover:text-on-surface transition-colors cursor-pointer"
                aria-label="Close"
                @click="close"
              />
            </div>

            <!-- Content -->
            <slot />
          </div>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
defineOptions({
  inheritAttrs: false,
})

withDefaults(defineProps<{
  title: string
  showClose?: boolean
}>(), {
  showClose: true,
})

const isOpen = defineModel<boolean>({ required: true })

const close = () => {
  isOpen.value = false
}

// Close on ESC key
onKeyStroke('Escape', () => {
  if (isOpen.value) {
    close()
  }
})
</script>
