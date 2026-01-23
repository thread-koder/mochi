<template>
  <div class="glass rounded-xl p-6">
    <h2 class="text-2xl font-bold font-heading mb-4">
      Summary
    </h2>
    <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
      <!-- CPU Metrics -->
      <div class="space-y-3">
        <p class="text-on-surface-secondary font-medium">
          CPU Utilization
        </p>
        <div>
          <p class="text-sm text-on-surface-secondary mb-1">
            Current
          </p>
          <p class="text-2xl font-bold text-primary-light">
            {{ formatCPU(utilization?.cpu?.current) }}
          </p>
        </div>
        <div class="grid grid-cols-2 gap-3 text-sm">
          <div>
            <p class="text-on-surface-secondary mb-1">
              Mean
            </p>
            <p class="text-primary-light font-medium">
              {{ formatCPU(utilization?.cpu?.stats?.mean) }}
            </p>
          </div>
          <div>
            <p class="text-on-surface-secondary mb-1">
              Median
            </p>
            <p class="text-primary-light font-medium">
              {{ formatCPU(utilization?.cpu?.stats?.median) }}
            </p>
          </div>
          <div>
            <p class="text-on-surface-secondary mb-1">
              P95
            </p>
            <p class="text-primary-light font-medium">
              {{ formatCPU(utilization?.cpu?.stats?.percentile?.p95) }}
            </p>
          </div>
          <div>
            <p class="text-on-surface-secondary mb-1">
              Max
            </p>
            <p class="text-primary-light font-medium">
              {{ formatCPU(utilization?.cpu?.stats?.max) }}
            </p>
          </div>
        </div>
        <div class="flex items-center gap-2 pt-2">
          <span class="text-sm text-on-surface-secondary">Trend:</span>
          <span
            v-if="utilization?.cpu?.trend?.direction === 'increasing'"
            class="flex items-center gap-1 text-sm text-error-light"
          >
            <Icon
              name="lucide:trending-up"
              class="text-base"
            />
            <span>Increasing</span>
          </span>
          <span
            v-else-if="utilization?.cpu?.trend?.direction === 'decreasing'"
            class="flex items-center gap-1 text-sm text-success-light"
          >
            <Icon
              name="lucide:trending-down"
              class="text-base"
            />
            <span>Decreasing</span>
          </span>
          <span
            v-else
            class="flex items-center gap-1 text-sm text-on-surface-secondary"
          >
            <Icon
              name="lucide:arrow-right"
              class="text-base"
            />
            <span>Stable</span>
          </span>
        </div>
        <div class="flex items-center gap-2">
          <span class="text-sm text-on-surface-secondary">Anomalies:</span>
          <span
            v-if="(utilization?.cpu?.anomalies?.anomaly_count ?? 0) > 0"
            class="text-sm bg-error/20 text-error-light px-2 py-1 rounded"
          >
            {{ utilization?.cpu?.anomalies?.anomaly_count ?? 0 }} detected
          </span>
          <span
            v-else
            class="text-sm text-on-surface-secondary"
          >
            No anomalies
          </span>
        </div>
      </div>

      <!-- Memory Metrics -->
      <div class="space-y-3">
        <p class="text-on-surface-secondary font-medium">
          Memory Utilization
        </p>
        <div>
          <p class="text-sm text-on-surface-secondary mb-1">
            Current
          </p>
          <p class="text-2xl font-bold text-secondary-light">
            {{ formatBytes(utilization?.memory?.current) }}
          </p>
        </div>
        <div class="grid grid-cols-2 gap-3 text-sm">
          <div>
            <p class="text-on-surface-secondary mb-1">
              Mean
            </p>
            <p class="text-secondary-light font-medium">
              {{ formatBytes(utilization?.memory?.stats?.mean) }}
            </p>
          </div>
          <div>
            <p class="text-on-surface-secondary mb-1">
              Median
            </p>
            <p class="text-secondary-light font-medium">
              {{ formatBytes(utilization?.memory?.stats?.median) }}
            </p>
          </div>
          <div>
            <p class="text-on-surface-secondary mb-1">
              P95
            </p>
            <p class="text-secondary-light font-medium">
              {{ formatBytes(utilization?.memory?.stats?.percentile?.p95) }}
            </p>
          </div>
          <div>
            <p class="text-on-surface-secondary mb-1">
              Max
            </p>
            <p class="text-secondary-light font-medium">
              {{ formatBytes(utilization?.memory?.stats?.max) }}
            </p>
          </div>
        </div>
        <div class="flex items-center gap-2 pt-2">
          <span class="text-sm text-on-surface-secondary">Trend:</span>
          <span
            v-if="utilization?.memory?.trend?.direction === 'increasing'"
            class="flex items-center gap-1 text-sm text-error-light"
          >
            <Icon
              name="lucide:trending-up"
              class="text-base"
            />
            <span>Increasing</span>
          </span>
          <span
            v-else-if="utilization?.memory?.trend?.direction === 'decreasing'"
            class="flex items-center gap-1 text-sm text-success-light"
          >
            <Icon
              name="lucide:trending-down"
              class="text-base"
            />
            <span>Decreasing</span>
          </span>
          <span
            v-else
            class="flex items-center gap-1 text-sm text-on-surface-secondary"
          >
            <Icon
              name="lucide:arrow-right"
              class="text-base"
            />
            <span>Stable</span>
          </span>
        </div>
        <div class="flex items-center gap-2">
          <span class="text-sm text-on-surface-secondary">Anomalies:</span>
          <span
            v-if="(utilization?.memory?.anomalies?.anomaly_count ?? 0) > 0"
            class="text-sm bg-error/20 text-error-light px-2 py-1 rounded"
          >
            {{ utilization?.memory?.anomalies?.anomaly_count ?? 0 }} detected
          </span>
          <span
            v-else
            class="text-sm text-on-surface-secondary px-2 py-1 rounded"
          >
            No anomalies
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type * as Compute from '#shared/types/compute/analysis'

defineProps<{
  utilization?: Compute.UtilizationResult
}>()
</script>
