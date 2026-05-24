<template>
  <div class="mb-6">
    <div class="flex space-x-4 border-b border-primary/20">
      <button
        v-for="tab in tabs"
        :key="tab.id"
        type="button"
        class="px-4 py-2 cursor-pointer transition-all"
        :class="tabButtonClass(tab.id)"
        @click="emit('update:modelValue', tab.id)"
      >
        {{ tab.label }}
      </button>
    </div>

    <Transition
      v-for="tab in tabs"
      :key="tab.id"
      enter-active-class="transition ease-out duration-200"
      enter-from-class="opacity-0 scale-90"
      enter-to-class="opacity-100 scale-100"
      leave-active-class="transition ease-in duration-150"
      leave-from-class="opacity-100 scale-100"
      leave-to-class="opacity-0 scale-90"
    >
      <div v-show="modelValue === tab.id">
        <slot :name="tab.id" />
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
export type TabItem = {
  id: string
  label: string
}

const props = defineProps<{
  tabs: TabItem[]
  modelValue: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const tabButtonClass = (id: string) => {
  return props.modelValue === id
    ? 'border-b-2 border-primary-light text-primary-light font-semibold'
    : 'text-on-surface-muted hover:text-on-surface-subtle'
}
</script>
