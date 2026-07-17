<template>
  <div class="panel p-4 space-y-6">
    <div>
      <p class="text-base font-semibold font-heading text-on-surface text-center mb-2">
        CPU
      </p>
      <ComputeResourceChart
        :data="cpu"
        type="cpu"
        title="CPU"
        :group="chartGroup"
        @ready="onChartReady"
      />
    </div>
    <div>
      <p class="text-base font-semibold font-heading text-on-surface text-center mb-2">
        Memory
      </p>
      <ComputeResourceChart
        :data="memory"
        type="memory"
        title="Memory"
        :group="chartGroup"
        @ready="onChartReady"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { connect, disconnect } from 'echarts/core'
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
