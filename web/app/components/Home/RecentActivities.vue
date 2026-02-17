<template>
  <div class="glass rounded-xl p-6 mb-8">
    <h2 class="text-2xl font-bold font-heading mb-4">
      Recent Activity
    </h2>
    <div class="space-y-3">
      <UiEmptyState
        v-if="!activities || activities.length === 0"
        icon="lucide:activity"
        title="No recent activity"
        description="Activity will appear here as recommendations are generated or applied."
      />
      <template v-else>
        <div
          v-for="activity in activities"
          :key="`${activity.timestamp}-${activity.type}`"
          class="flex items-center space-x-3 p-4 rounded-lg glass
           hover:bg-primary/10 transition-all hover:text-on-surface"
        >
          <Icon
            :name="activityIcon(activity.type)"
            class="text-xl"
            :class="activityIconColor(activity.type)"
          />
          <span class="text-sm font-medium flex-1">{{ activity.message }}</span>
          <span class="text-on-surface-secondary text-sm ml-auto">
            {{ timeAgo(activity.timestamp) }}
          </span>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import type * as Home from '#shared/types/home'

defineProps<{
  activities?: Home.HomeResponse['activities']
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
      return 'text-primary-light'
    default:
      return 'text-secondary-light'
  }
}
</script>
