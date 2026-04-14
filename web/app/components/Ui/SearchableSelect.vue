<template>
  <div
    ref="containerRef"
    class="relative"
  >
    <!-- Trigger Button -->
    <button
      ref="triggerRef"
      type="button"
      class="bg-surface-elevated border border-primary/20 rounded-lg px-3 py-2
       text-sm text-on-surface-secondary focus:outline-none focus:border-primary/50
       min-w-[140px] text-left flex items-center justify-between gap-2 cursor-pointer"
      :class="modelValue ? 'text-on-surface' : ''"
      @click="toggleDropdown"
    >
      <span class="truncate">
        {{ displayValue }}
      </span>
      <Icon
        name="lucide:chevron-down"
        class="text-xs shrink-0 transition-transform"
        :class="isOpen ? 'rotate-180' : ''"
      />
    </button>

    <!-- Dropdown -->
    <Teleport to="body">
      <Transition
        enter-active-class="transition ease-out duration-200"
        enter-from-class="opacity-0 scale-95"
        enter-to-class="opacity-100 scale-100"
        leave-active-class="transition ease-in duration-150"
        leave-from-class="opacity-100 scale-100"
        leave-to-class="opacity-0 scale-95"
      >
        <div
          v-if="isOpen"
          ref="dropdownRef"
          class="fixed z-9999 glass rounded-lg border border-primary/20 shadow-lg overflow-hidden"
          :style="dropdownStyle"
        >
          <!-- Search Input -->
          <div
            v-if="searchable"
            class="p-2 border-b border-primary/10"
          >
            <div class="relative">
              <Icon
                name="lucide:search"
                class="absolute left-2 top-1/2 -translate-y-1/2 text-xs text-on-surface-muted"
              />
              <input
                ref="searchInputRef"
                v-model="searchQuery"
                type="text"
                :placeholder="searchPlaceholder"
                class="w-full bg-surface-elevated border border-primary/20 rounded-lg px-8 py-2
                 text-sm text-on-surface-secondary placeholder:text-on-surface-muted
                 focus:outline-none focus:border-primary/50"
              >
            </div>
          </div>

          <!-- Options List -->
          <div class="max-h-60 overflow-y-auto">
            <!-- Null Option -->
            <button
              v-if="nullOption"
              type="button"
              class="w-full px-3 py-2 text-sm text-left transition-colors
               hover:bg-primary/10 hover:text-on-surface cursor-pointer"
              :class="modelValue === null ? 'bg-primary/20 text-primary-light' : 'text-on-surface-secondary'"
              @click="selectNull"
            >
              {{ nullOption }}
            </button>
            <div
              v-if="filteredOptions.length === 0"
              class="px-3 py-2 text-sm text-on-surface-muted text-center"
            >
              No options found
            </div>
            <button
              v-for="option in filteredOptions"
              :key="optionValue(option)"
              type="button"
              class="w-full px-3 py-2 text-sm text-left transition-colors
               hover:bg-primary/10 hover:text-on-surface cursor-pointer"
              :class="isSelected(option) ? 'bg-primary/20 text-primary-light' : 'text-on-surface-secondary'"
              @click="selectOption(option)"
            >
              {{ optionLabel(option) }}
            </button>
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
interface Props {
  options: Array<string | { value: string, label: string }>
  placeholder?: string
  searchPlaceholder?: string
  nullOption?: string
  searchable?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  placeholder: 'Select...',
  searchPlaceholder: 'Search...',
  nullOption: undefined,
  searchable: true,
})

const modelValue = defineModel<string | null>({ required: true })

const emit = defineEmits<{
  select: [value: string | null]
}>()

const isOpen = ref(false)
const searchQuery = ref('')
const searchInputRef = ref<HTMLInputElement | null>(null)
const containerRef = ref<HTMLDivElement | null>(null)
const triggerRef = ref<HTMLButtonElement | null>(null)
const dropdownRef = ref<HTMLDivElement | null>(null)

const triggerBounding = useElementBounding(triggerRef)

const dropdownStyle = computed(() => {
  if (!triggerBounding.width.value) return {}

  return {
    top: `${triggerBounding.bottom.value + 8}px`,
    left: `${triggerBounding.left.value}px`,
    width: `${triggerBounding.width.value}px`,
  }
})

const optionValue = (option: string | { value: string, label: string }): string => {
  return typeof option === 'string' ? option : option.value
}

const optionLabel = (option: string | { value: string, label: string }): string => {
  return typeof option === 'string' ? option : option.label
}

const isSelected = (option: string | { value: string, label: string }): boolean => {
  return modelValue.value === optionValue(option)
}

const filteredOptions = computed(() => {
  if (!searchQuery.value.trim()) {
    return props.options
  }

  const query = searchQuery.value.toLowerCase().trim()
  return props.options.filter((option) => {
    const label = optionLabel(option).toLowerCase()
    return label.includes(query)
  })
})

const displayValue = computed(() => {
  if (!modelValue.value) {
    return props.nullOption || props.placeholder
  }

  const option = props.options.find(opt => optionValue(opt) === modelValue.value)
  return option ? optionLabel(option) : props.placeholder
})

const toggleDropdown = () => {
  isOpen.value = !isOpen.value
  if (isOpen.value) {
    triggerBounding.update()
    nextTick(() => {
      searchInputRef.value?.focus()
      searchQuery.value = ''
    })
  }
}

const closeDropdown = () => {
  isOpen.value = false
  searchQuery.value = ''
}

const selectNull = () => {
  modelValue.value = null
  emit('select', null)
  closeDropdown()
}

const selectOption = (option: string | { value: string, label: string }) => {
  const value = optionValue(option)
  if (value === modelValue.value) {
    if (props.nullOption) {
      modelValue.value = null
      emit('select', null)
    }
    else {
      emit('select', value)
    }
    closeDropdown()
    return
  }
  modelValue.value = value
  emit('select', value)
  closeDropdown()
}

onClickOutside(containerRef, () => {
  if (isOpen.value) {
    closeDropdown()
  }
}, { ignore: [dropdownRef] })

onKeyStroke('Escape', () => {
  if (isOpen.value) {
    closeDropdown()
  }
})
</script>
