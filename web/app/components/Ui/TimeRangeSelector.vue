<template>
  <div class="flex items-center gap-2">
    <UiSearchableSelect
      v-model="selectedValue"
      :options="timeRangeOptions"
      :searchable="false"
      placeholder="Select time range"
      @select="onOptionSelect"
    />
    <Transition
      enter-active-class="transition ease-out duration-200"
      enter-from-class="opacity-0 scale-95"
      enter-to-class="opacity-100 scale-100"
      leave-active-class="transition ease-in duration-150"
      leave-from-class="opacity-100 scale-100"
      leave-to-class="opacity-0 scale-95"
    >
      <div
        v-if="isEditingCustom"
        class="flex items-center gap-2"
      >
        <input
          ref="customInputRef"
          v-model="customInputValue"
          type="text"
          placeholder="e.g., 2h, 48h, 5d"
          class="bg-surface-elevated border border-on-surface-muted/20 rounded-lg px-3 py-2
           text-sm text-on-surface-secondary focus:outline-none focus:border-primary/50 w-32"
          @keyup.enter="applyCustom"
        >
        <button
          class="px-3 py-2 bg-primary/20 hover:bg-primary/30 text-sm text-primary-light
           rounded-lg transition-all cursor-pointer"
          @click="applyCustom"
        >
          Apply
        </button>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
const timeRange = defineModel<string | null>({ required: true })

const CUSTOM_SENTINEL = '__custom__'
const predefinedOptions = ['1h', '6h', '12h', '24h', '3d', '7d', '14d', '30d']

const isEditingCustom = ref(false)
const customInputValue = ref('')
const customInputRef = ref<HTMLInputElement | null>(null)
const customValue = ref<string | null>(null)

const timeRangeOptions = computed(() => {
  const options: Array<{ value: string, label: string }> = [
    { value: '1h', label: 'Last Hour' },
    { value: '6h', label: 'Last 6 Hours' },
    { value: '12h', label: 'Last 12 Hours' },
    { value: '24h', label: 'Last 24 Hours' },
    { value: '3d', label: 'Last 3 Days' },
    { value: '7d', label: 'Last 7 Days' },
    { value: '14d', label: 'Last 14 Days' },
    { value: '30d', label: 'Last 30 Days' },
  ]

  if (customValue.value) {
    options.push({ value: CUSTOM_SENTINEL, label: `Custom (${customValue.value})` })
  }
  else {
    options.push({ value: CUSTOM_SENTINEL, label: 'Custom' })
  }

  return options
})

const selectedValue = computed({
  get: () => {
    if (timeRange.value && predefinedOptions.includes(timeRange.value)) {
      return timeRange.value
    }
    return CUSTOM_SENTINEL
  },
  set: (value: string | null) => {
    if (value !== CUSTOM_SENTINEL) {
      isEditingCustom.value = false
      customValue.value = null
      timeRange.value = value
    }
  },
})

const onOptionSelect = (value: string | null) => {
  if (value === CUSTOM_SENTINEL) {
    isEditingCustom.value = true
    customInputValue.value = timeRange.value || customValue.value || ''
    nextTick(() => {
      customInputRef.value?.focus()
    })
  }
}

const applyCustom = () => {
  const value = customInputValue.value.trim()
  if (value) {
    timeRange.value = value
    customValue.value = predefinedOptions.includes(value) ? null : value
    isEditingCustom.value = false
  }
}

watch(timeRange, (newValue) => {
  if (newValue && !predefinedOptions.includes(newValue)) {
    customValue.value = newValue
  }
}, { immediate: true })
</script>
