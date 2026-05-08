<template>
  <div class="glass rounded-xl p-6">
    <h2 class="text-2xl font-bold font-heading mb-4">
      Overview
    </h2>
    <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
      <!-- Workload Information -->
      <div>
        <h3 class="text-sm font-semibold text-on-surface-secondary mb-3">
          Workload Information
        </h3>
        <div class="space-y-2">
          <div class="flex items-center justify-between min-h-9">
            <span class="text-sm text-on-surface-secondary shrink-0">Namespace:</span>
            <NuxtLink
              :to="`/namespaces/${recommendation.namespace}`"
              class="text-sm text-primary-light hover:text-primary transition-colors font-medium text-right truncate ml-2"
            >
              {{ recommendation.namespace }}
            </NuxtLink>
          </div>
          <div class="flex items-center justify-between min-h-9">
            <span class="text-sm text-on-surface-secondary shrink-0">Workload:</span>
            <NuxtLink
              :to="`/namespaces/${recommendation.namespace}/workloads/${recommendation.workload_type}/${recommendation.workload_name}`"
              class="text-sm text-primary-light hover:text-primary transition-colors font-medium text-right truncate ml-2"
            >
              {{ recommendation.workload_name }}
            </NuxtLink>
          </div>
          <div class="flex items-center justify-between min-h-9">
            <span class="text-sm text-on-surface-secondary shrink-0">Type:</span>
            <span class="px-2 py-1 rounded-full text-sm font-medium bg-primary/20 text-primary-light border border-primary/30 shrink-0">{{ recommendation.workload_type }}</span>
          </div>
        </div>
      </div>

      <!-- Recommendation Details -->
      <div>
        <h3 class="text-sm font-semibold text-on-surface-secondary mb-3">
          Recommendation Details
        </h3>
        <div class="space-y-2">
          <div class="flex items-center justify-between min-h-9">
            <span class="text-sm text-on-surface-secondary shrink-0">Mode:</span>
            <span class="text-sm text-on-surface font-medium">{{ formatTitleCase(recommendation.recommendation_mode) }}</span>
          </div>
          <div
            v-if="recommendation.analysis_time_range"
            class="flex items-center justify-between min-h-9"
          >
            <span class="text-sm text-on-surface-secondary shrink-0">Analysis Period:</span>
            <span class="text-sm text-on-surface font-medium">{{ formatDuration(recommendation.analysis_time_range) }}</span>
          </div>
          <div class="flex items-center justify-between min-h-9">
            <span class="text-sm text-on-surface-secondary shrink-0">Status:</span>
            <span
              :class="statusBadgeClass(recommendation.status)"
              class="px-2 py-1 rounded-full text-sm font-medium border shrink-0"
            >
              {{ formatTitleCase(recommendation.status) }}
            </span>
          </div>
        </div>
      </div>

      <!-- Timestamps -->
      <div class="md:col-span-2">
        <h3 class="text-sm font-semibold text-on-surface-secondary mb-3">
          Timestamps
        </h3>
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <span class="text-xs text-on-surface-muted">Created</span>
            <p class="text-sm text-on-surface font-medium mt-1">
              {{ formatDate(recommendation.created_at) }}
            </p>
            <p class="text-xs text-on-surface-muted mt-0.5">
              {{ timeAgo(recommendation.created_at) }}
            </p>
          </div>
          <div
            v-if="recommendation.updated_at"
          >
            <span class="text-xs text-on-surface-muted">Updated</span>
            <p class="text-sm text-on-surface font-medium mt-1">
              {{ formatDate(recommendation.updated_at) }}
            </p>
            <p class="text-xs text-on-surface-muted mt-0.5">
              {{ timeAgo(recommendation.updated_at) }}
            </p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { RecommendationRecord } from '#shared/types/compute'

defineProps<{
  recommendation: RecommendationRecord
}>()

const formatDate = (dateString: string): string => {
  return new Date(dateString).toLocaleString()
}
</script>
