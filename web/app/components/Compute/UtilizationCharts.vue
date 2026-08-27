<template>
  <div class="panel p-4 space-y-6">
    <div>
      <p class="text-base font-semibold font-heading text-on-surface text-center mb-2">
        CPU
      </p>
      <ComputeResourceChart
        v-if="hasEnoughPoints(cpu.length)"
        :data="cpu"
        type="cpu"
        title="CPU"
        :group="chartGroup"
        @ready="onChartReady"
      />
      <div
        v-else
        class="h-72 flex flex-col items-center justify-center"
      >
        <UiEmptyState
          icon="lucide:chart-no-axes-column"
          title="No metrics"
        />
      </div>
    </div>
    <div>
      <p class="text-base font-semibold font-heading text-on-surface text-center mb-2">
        Memory
      </p>
      <ComputeResourceChart
        v-if="hasEnoughPoints(memory.length)"
        :data="memory"
        type="memory"
        title="Memory"
        :group="chartGroup"
        @ready="onChartReady"
      />
      <div
        v-else
        class="h-72 flex flex-col items-center justify-center"
      >
        <UiEmptyState
          icon="lucide:chart-no-axes-column"
          title="No metrics"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { connect, disconnect } from 'echarts/core'
import { hasEnoughPoints } from '#shared/utils/timeseries'
import type { DataPoint } from '#shared/types/timeseries'

defineProps<{
  cpu: DataPoint[]
  memory: DataPoint[]
}>()

const chartGroup = `utilization-${useId()}`

const onChartReady = () => {
  connect(chartGroup)
}

onBeforeUnmount(() => {
  disconnect(chartGroup)
})
</script>
