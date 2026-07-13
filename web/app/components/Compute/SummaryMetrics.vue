<template>
  <section class="grid grid-cols-1 md:grid-cols-3 gap-4">
    <!-- CPU -->
    <div class="panel p-4 space-y-3">
      <p class="text-lg font-semibold font-heading text-on-surface">
        CPU
      </p>
      <div>
        <p class="text-sm text-on-surface-secondary mb-1">
          Current
        </p>
        <p class="text-on-surface text-2xl font-bold">
          {{ formatCPU(utilization.cpu.current) }}
        </p>
      </div>
      <div class="grid grid-cols-2 gap-3 text-sm">
        <div>
          <p class="text-on-surface-secondary mb-1">
            Mean
          </p>
          <p class="text-on-surface font-medium">
            {{ formatCPU(utilization.cpu.stats.mean) }}
          </p>
        </div>
        <div>
          <p class="text-on-surface-secondary mb-1">
            Median
          </p>
          <p class="text-on-surface font-medium">
            {{ formatCPU(utilization.cpu.stats.median) }}
          </p>
        </div>
        <div>
          <p class="text-on-surface-secondary mb-1">
            P95
          </p>
          <p class="text-on-surface font-medium">
            {{ formatCPU(utilization.cpu.stats.percentile.p95) }}
          </p>
        </div>
        <div>
          <p class="text-on-surface-secondary mb-1">
            Max
          </p>
          <p class="text-on-surface font-medium">
            {{ formatCPU(utilization.cpu.stats.max) }}
          </p>
        </div>
      </div>
      <div class="flex items-center gap-2 pt-2">
        <span class="text-sm text-on-surface-secondary">Trend:</span>
        <span
          v-if="utilization.cpu.trend.direction === 'increasing'"
          class="flex items-center gap-1 text-sm text-error-light"
        >
          <Icon
            name="lucide:trending-up"
            class="text-base"
          />
          <span>Increasing</span>
        </span>
        <span
          v-else-if="utilization.cpu.trend.direction === 'decreasing'"
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
          v-if="utilization.cpu.anomalies.anomaly_count > 0"
          class="text-sm font-medium text-error-light"
        >
          {{ utilization.cpu.anomalies.anomaly_count }} detected
        </span>
        <span
          v-else
          class="text-sm text-on-surface-secondary"
        >
          None
        </span>
      </div>
    </div>

    <!-- Memory -->
    <div class="panel p-4 space-y-3">
      <p class="text-lg font-semibold font-heading text-on-surface">
        Memory
      </p>
      <div>
        <p class="text-sm text-on-surface-secondary mb-1">
          Current
        </p>
        <p class="text-on-surface text-2xl font-bold">
          {{ formatBytes(utilization.memory.current) }}
        </p>
      </div>
      <div class="grid grid-cols-2 gap-3 text-sm">
        <div>
          <p class="text-on-surface-secondary mb-1">
            Mean
          </p>
          <p class="text-on-surface font-medium">
            {{ formatBytes(utilization.memory.stats.mean) }}
          </p>
        </div>
        <div>
          <p class="text-on-surface-secondary mb-1">
            Median
          </p>
          <p class="text-on-surface font-medium">
            {{ formatBytes(utilization.memory.stats.median) }}
          </p>
        </div>
        <div>
          <p class="text-on-surface-secondary mb-1">
            P95
          </p>
          <p class="text-on-surface font-medium">
            {{ formatBytes(utilization.memory.stats.percentile.p95) }}
          </p>
        </div>
        <div>
          <p class="text-on-surface-secondary mb-1">
            Max
          </p>
          <p class="text-on-surface font-medium">
            {{ formatBytes(utilization.memory.stats.max) }}
          </p>
        </div>
      </div>
      <div class="flex items-center gap-2 pt-2">
        <span class="text-sm text-on-surface-secondary">Trend:</span>
        <span
          v-if="utilization.memory.trend.direction === 'increasing'"
          class="flex items-center gap-1 text-sm text-error-light"
        >
          <Icon
            name="lucide:trending-up"
            class="text-base"
          />
          <span>Increasing</span>
        </span>
        <span
          v-else-if="utilization.memory.trend.direction === 'decreasing'"
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
          v-if="utilization.memory.anomalies.anomaly_count > 0"
          class="text-sm font-medium text-error-light"
        >
          {{ utilization.memory.anomalies.anomaly_count }} detected
        </span>
        <span
          v-else
          class="text-sm text-on-surface-secondary"
        >
          None
        </span>
      </div>
    </div>

    <!-- Stability -->
    <div class="panel p-4 space-y-3">
      <p class="text-lg font-semibold font-heading text-on-surface">
        Stability
      </p>
      <div>
        <p class="text-sm text-on-surface-secondary mb-1">
          Health Score
        </p>
        <p
          class="text-2xl font-bold"
          :class="scoreColor(stability.stability_score, { midThreshold: 0.6, type: 'text' })"
        >
          {{ formatPercentage(stability.stability_score) }}
        </p>
      </div>
      <div class="grid grid-cols-2 gap-3 text-sm">
        <div>
          <p class="text-on-surface-secondary mb-1">
            CPU Throttling
          </p>
          <p class="text-on-surface font-medium">
            {{ formatPercentage(stability.cpu_throttling) }}
          </p>
        </div>
        <div>
          <p class="text-on-surface-secondary mb-1">
            CPU Pressure
          </p>
          <p class="text-on-surface font-medium">
            {{ formatPercentage(stability.cpu_pressure) }}
          </p>
        </div>
        <div>
          <p class="text-on-surface-secondary mb-1">
            Memory Fail
          </p>
          <p class="text-on-surface font-medium">
            {{ stability.memory_fail_cnt }}
          </p>
        </div>
        <div>
          <p class="text-on-surface-secondary mb-1">
            Memory Pressure
          </p>
          <p class="text-on-surface font-medium">
            {{ formatPercentage(stability.memory_pressure) }}
          </p>
        </div>
      </div>
      <div class="flex items-center gap-2 pt-2">
        <span class="text-sm text-on-surface-secondary">OOM:</span>
        <span
          class="text-sm font-medium"
          :class="stability.memory_oom > 0 ? 'text-error-light' : 'text-on-surface'"
        >
          {{ stability.memory_oom }}
        </span>
      </div>
      <div class="flex items-center gap-2">
        <span class="text-sm text-on-surface-secondary">Restarts:</span>
        <span
          class="text-sm font-medium py-1 rounded"
          :class="stability.restarts > 0 ? 'text-error-light' : 'text-on-surface'"
        >
          {{ stability.restarts }}
        </span>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { formatCPU } from '#shared/utils/compute/format'
import type { UtilizationResult, StabilityResult } from '#shared/types/compute'

defineProps<{
  utilization: UtilizationResult
  stability: StabilityResult
}>()
</script>
