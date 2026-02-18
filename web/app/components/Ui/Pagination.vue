<template>
  <div class="flex items-center justify-between gap-4 pt-4">
    <!-- Results info -->
    <div class="text-sm text-on-surface-secondary">
      <span>Showing </span>
      <span class="text-on-surface font-medium">{{ start }}</span>
      <span> - </span>
      <span class="text-on-surface font-medium">{{ end }}</span>
      <span> of </span>
      <span class="text-on-surface font-medium">{{ total }}</span>
    </div>

    <!-- Pagination controls -->
    <div class="flex items-center gap-2">
      <!-- Previous button -->
      <button
        :disabled="currentPage === 1"
        class="flex items-center justify-center px-3 py-2 rounded-lg text-sm font-medium transition-all cursor-pointer
         disabled:opacity-50 disabled:cursor-not-allowed
         text-on-surface-secondary hover:text-on-surface hover:bg-primary/10
         disabled:hover:bg-transparent disabled:hover:text-on-surface-secondary"
        @click="$emit('update:page', currentPage - 1)"
      >
        <Icon
          name="lucide:chevron-left"
          class="text-base shrink-0"
        />
      </button>

      <!-- Page numbers -->
      <div class="flex items-center gap-1">
        <template
          v-for="(item, i) in visibleItems"
          :key="item === 'ellipsis' ? `e-${i}` : item"
        >
          <span
            v-if="item === 'ellipsis'"
            class="px-2 py-1.5 text-on-surface-secondary select-none"
            aria-hidden="true"
          >…</span>
          <button
            v-else
            :class="[
              'px-3 py-1.5 rounded-lg text-sm font-medium transition-all cursor-pointer',
              item === currentPage
                ? 'bg-primary/20 text-primary-light'
                : 'text-on-surface-secondary hover:text-on-surface hover:bg-primary/10',
            ]"
            @click="$emit('update:page', item)"
          >
            {{ item }}
          </button>
        </template>
      </div>

      <!-- Next button -->
      <button
        :disabled="currentPage === totalPages"
        class="flex items-center justify-center px-3 py-2 rounded-lg text-sm font-medium transition-all cursor-pointer
         disabled:opacity-50 disabled:cursor-not-allowed
         text-on-surface-secondary hover:text-on-surface hover:bg-primary/10
         disabled:hover:bg-transparent disabled:hover:text-on-surface-secondary"
        @click="$emit('update:page', currentPage + 1)"
      >
        <Icon
          name="lucide:chevron-right"
          class="text-base shrink-0"
        />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
const props = defineProps<{
  currentPage: number
  totalPages: number
  total: number
  pageSize: number
}>()

defineEmits<{
  'update:page': [page: number]
}>()

const start = computed(() => {
  if (props.total === 0) return 0
  return (props.currentPage - 1) * props.pageSize + 1
})

const end = computed(() => {
  return Math.min(props.currentPage * props.pageSize, props.total)
})

const visibleItems = computed(() => {
  const total = props.totalPages
  const current = props.currentPage
  const items: (number | 'ellipsis')[] = []

  if (total <= 5) {
    // Show all pages, no ellipsis
    for (let i = 1; i <= total; i++) items.push(i)
    return items
  }

  const showLeft = current > 3
  const showRight = current < total - 2

  items.push(1)
  if (showLeft) items.push('ellipsis')

  const start = showLeft ? Math.max(2, current - 1) : 2
  const end = showRight ? Math.min(total - 1, current + 1) : total - 1
  for (let i = start; i <= end; i++) items.push(i)

  if (showRight) items.push('ellipsis')
  if (total > 1) items.push(total)

  return items
})
</script>
