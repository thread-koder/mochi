<template>
  <div class="glass rounded-xl p-6">
    <h2 class="text-2xl font-bold font-heading mb-4">
      Summary
    </h2>
    <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
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
            class="text-sm font-medium bg-error/20 text-error-light border border-error-light/30 px-2 py-1 rounded-full"
          >
            {{ utilization?.cpu?.anomalies?.anomaly_count ?? 0 }} detected
          </span>
          <span
            v-else
            class="text-sm text-on-surface-secondary py-1 rounded-full"
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
            class="text-sm font-medium bg-error/20 text-error-light border border-error-light/30 px-2 py-1 rounded-full"
          >
            {{ utilization?.memory?.anomalies?.anomaly_count ?? 0 }} detected
          </span>
          <span
            v-else
            class="text-sm text-on-surface-secondary py-1 rounded-full"
          >
            No anomalies
          </span>
        </div>
      </div>

      <!-- Stability Metrics -->
      <div class="space-y-3">
        <p class="text-on-surface-secondary font-medium">
          Stability
        </p>
        <div>
          <p class="text-sm text-on-surface-secondary mb-1">
            Health Score
          </p>
          <p
            class="text-2xl font-bold"
            :class="scoreColor(stability?.stability_score, { midThreshold: 0.6, type: 'text' })"
          >
            {{ formatPercentage(stability?.stability_score ?? 0) }}
          </p>
        </div>
        <div class="grid grid-cols-2 gap-3 text-sm">
          <div>
            <p class="text-on-surface-secondary mb-1">
              CPU Throttling
            </p>
            <p class="text-tertiary-light font-medium">
              {{ formatPercentage(stability?.cpu_throttling ?? 0) }}
            </p>
          </div>
          <div>
            <p class="text-on-surface-secondary mb-1">
              CPU Pressure
            </p>
            <p class="text-tertiary-light font-medium">
              {{ formatPercentage(stability?.cpu_pressure ?? 0) }}
            </p>
          </div>
          <div>
            <p class="text-on-surface-secondary mb-1">
              Memory Fail
            </p>
            <p class="text-tertiary-light font-medium">
              {{ stability?.memory_fail_cnt ?? 0 }}
            </p>
          </div>
          <div>
            <p class="text-on-surface-secondary mb-1">
              Memory Pressure
            </p>
            <p class="text-tertiary-light font-medium">
              {{ formatPercentage(stability?.memory_pressure ?? 0) }}
            </p>
          </div>
        </div>
        <div class="flex items-center gap-2 pt-2">
          <span class="text-sm text-on-surface-secondary">OOM:</span>
          <span
            class="text-sm font-medium"
            :class="(stability?.memory_oom ?? 0) > 0 ? 'text-error-light' : 'text-on-surface'"
          >
            {{ stability?.memory_oom ?? 0 }}
          </span>
        </div>
        <div class="flex items-center gap-2">
          <span class="text-sm text-on-surface-secondary">Restarts:</span>
          <span
            class="text-sm font-medium py-1 rounded"
            :class="(stability?.restarts ?? 0) > 0 ? 'text-error-light' : 'text-on-surface'"
          >
            {{ stability?.restarts ?? 0 }}
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { formatCPU } from '#shared/utils/compute/format'
import type { UtilizationResult, StabilityResult } from '#shared/types/compute'

defineProps<{
  utilization?: UtilizationResult
  stability?: StabilityResult
}>()
</script>
