<template>
  <div
    ref="chartEl"
    class="h-72 w-full relative"
  />
</template>

<script setup lang="ts">
import { init, type EChartsType } from 'echarts/core'
import { buildLineChartOption } from '#shared/utils/compute/chart'
import type { DataPoint } from '#shared/types/timeseries'

const props = defineProps<{
  data: DataPoint[]
  type: 'cpu' | 'memory'
  title: string
  group?: string
}>()

const emit = defineEmits<{
  ready: []
}>()

const chartEl = ref<HTMLDivElement | null>(null)
let chartInstance: EChartsType | null = null

const initChart = () => {
  if (!chartEl.value) return null

  if (!chartInstance) {
    chartInstance = init(chartEl.value, undefined, { renderer: 'canvas' })
    if (props.group) {
      chartInstance.group = props.group
    }
    emit('ready')
  }

  return chartInstance
}

const renderChart = () => {
  const chart = initChart()
  if (!chart || !chartEl.value) return

  chart.setOption(buildLineChartOption({
    type: props.type,
    data: props.data,
    title: props.title,
    width: chartEl.value.clientWidth,
  }), { notMerge: true })
}

watch(() => props.data, () => {
  nextTick(() => {
    renderChart()
  })
}, { immediate: true })

const colorMode = useColorMode({ storageKey: 'mochi-theme' })
watch(colorMode, () => {
  if (chartInstance) {
    nextTick(() => {
      renderChart()
    })
  }
})

useResizeObserver(chartEl, () => {
  if (!chartInstance) return
  chartInstance.resize()
  renderChart()
})

onBeforeUnmount(() => {
  if (chartInstance) {
    chartInstance.dispose()
    chartInstance = null
  }
})
</script>
