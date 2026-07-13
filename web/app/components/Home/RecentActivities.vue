<template>
  <section class="flex flex-col">
    <h2 class="text-2xl font-bold font-heading mb-4">
      Recent Activity
    </h2>

    <div
      v-if="activities.length === 0"
      class="panel p-4"
    >
      <UiEmptyState
        icon="lucide:activity"
        title="No recent activity"
        description="Activity will appear here as recommendations are generated or applied."
      />
    </div>

    <div
      v-else
      class="panel max-h-72 overflow-y-auto"
    >
      <div class="p-4 space-y-2">
        <div
          v-for="activity in activities"
          :key="`${activity.timestamp}-${activity.type}`"
          class="flex items-center gap-3 px-2 py-2"
        >
          <Icon
            :name="activityIcon(activity.type)"
            class="text-base shrink-0"
            :class="activityIconColor(activity.type)"
          />
          <span
            class="text-sm font-medium flex-1 min-w-0 line-clamp-2"
            :title="activity.message"
          >
            {{ activity.message }}
          </span>
          <span class="text-xs text-on-surface-secondary shrink-0 ml-2 tabular-nums">
            {{ timeAgo(activity.timestamp) }}
          </span>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import type { Activity } from '#shared/types/home'

defineProps<{
  activities: Activity[]
}>()

const activityIcon = (type: string) => {
  switch (type) {
    case 'recommendation_applied':
      return 'lucide:circle-check'
    case 'recommendation_generated':
      return 'lucide:lightbulb'
    default:
      return 'lucide:circle-alert'
  }
}

const activityIconColor = (type: string) => {
  switch (type) {
    case 'recommendation_applied':
      return 'text-success-light'
    case 'recommendation_generated':
      return 'text-info-light'
    default:
      return 'text-on-surface-muted'
  }
}
</script>
