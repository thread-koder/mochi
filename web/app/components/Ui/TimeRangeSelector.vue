<template>
  <div class="flex items-center gap-2">
    <select
      v-model="timeRange"
      class="bg-surface-elevated border border-primary/20 rounded-lg px-4 py-2
       text-sm text-on-surface-secondary focus:outline-none focus:border-primary/50"
    >
      <option value="1h">
        Last Hour
      </option>
      <option value="6h">
        Last 6 Hours
      </option>
      <option value="12h">
        Last 12 Hours
      </option>
      <option value="24h">
        Last 24 Hours
      </option>
      <option value="3d">
        Last 3 Days
      </option>
      <option value="7d">
        Last 7 Days
      </option>
      <option value="30d">
        Last 30 Days
      </option>
      <option :value="customTimeRange ? timeRange : 'custom'">
        {{ customTimeRange && customValue ? `Custom (${customValue})` : 'Custom' }}
      </option>
    </select>
    <div
      v-if="customTimeRange"
      class="flex items-center gap-2"
    >
      <input
        v-model="customValue"
        type="text"
        placeholder="e.g., 2h, 48h, 5d"
        class="bg-surface-elevated border border-primary/20 rounded-lg px-3 py-2
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
  </div>
</template>

<script setup lang="ts">
const timeRange = defineModel<string | null>({ required: true })
const customValue = ref('')

const predefinedOptions = ['1h', '6h', '12h', '24h', '3d', '7d', '30d']

const customTimeRange = computed(() => {
  return timeRange.value !== null && !predefinedOptions.includes(timeRange.value)
})

const applyCustom = () => {
  const value = customValue.value.trim()
  if (value) {
    timeRange.value = value
  }
}

watch(timeRange, (newTimeRange) => {
  if (newTimeRange && predefinedOptions.includes(newTimeRange)) {
    customValue.value = ''
  }
})
</script>
