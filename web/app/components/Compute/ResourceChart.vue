<template>
  <div class="h-96 relative">
    <canvas
      ref="chartCanvas"
      class="w-full h-full"
    />
  </div>
</template>

<script setup lang="ts">
import {
  Chart,
  LineController,
  LineElement,
  PointElement,
  TimeScale,
  LinearScale,
  Decimation,
  Legend,
  Tooltip,
  Filler,
} from 'chart.js'
import 'chartjs-adapter-date-fns'

Chart.register(
  LineController,
  LineElement,
  PointElement,
  TimeScale,
  LinearScale,
  Decimation,
  Legend,
  Tooltip,
  Filler,
)

const props = defineProps<{
  data: Array<{ timestamp: string, value: number }>
  type: 'cpu' | 'memory'
  title: string
}>()

const chartCanvas = ref<HTMLCanvasElement | null>(null)
let chartInstance: Chart | null = null

const cssVariableColor = (variableName: string, opacity?: number): string => {
  if (!import.meta.client) return ''
  const root = document.documentElement
  const computedStyle = getComputedStyle(root)
  const value = computedStyle.getPropertyValue(variableName).trim()
  if (!value) return ''

  if (opacity !== undefined) {
    return value.replace(/\)$/, ` / ${opacity})`)
  }
  return value
}

const initChart = () => {
  if (!chartCanvas.value || !props.data || props.data.length === 0) {
    return
  }

  if (chartInstance) {
    chartInstance.destroy()
    chartInstance = null
  }

  const ctx = chartCanvas.value.getContext('2d')
  if (!ctx) return

  // Convert to Chart.js format
  const chartData = props.data.map(dp => ({
    x: new Date(dp.timestamp).getTime(),
    y: dp.value,
  }))

  const isCPU = props.type === 'cpu'
  const maxValue = Math.max(...chartData.map(d => d.y))
  const useMillicores = isCPU && maxValue < 1

  // Get theme colors
  const primaryColor = isCPU
    ? cssVariableColor('--color-primary-light')
    : cssVariableColor('--color-secondary-light')
  const primaryColorWithOpacity = isCPU
    ? cssVariableColor('--color-primary-light', 0.1)
    : cssVariableColor('--color-secondary-light', 0.1)
  const gridColor = isCPU
    ? cssVariableColor('--color-primary', 0.1)
    : cssVariableColor('--color-secondary', 0.1)
  const textColor = cssVariableColor('--color-on-surface-secondary')

  chartInstance = new Chart(ctx, {
    type: 'line',
    data: {
      datasets: [{
        label: isCPU
          ? (useMillicores ? 'CPU Utilization (millicores)' : 'CPU Utilization (cores)')
          : 'Memory Utilization',
        data: chartData,
        parsing: false,
        borderColor: primaryColor,
        backgroundColor: primaryColorWithOpacity,
        borderWidth: 2,
        fill: true,
        tension: 0.6,
        pointRadius: 0,
        pointHoverRadius: 5,
        normalized: true,
      }],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      normalized: true,
      parsing: false,
      interaction: {
        intersect: false,
        mode: 'index',
      },
      plugins: {
        decimation: {
          enabled: true,
          algorithm: 'lttb',
          threshold: 300,
        },
        legend: {
          labels: {
            color: textColor,
            font: { family: 'Inconsolata', size: 14 },
          },
        },
        tooltip: {
          backgroundColor: cssVariableColor('--color-surface-elevated'),
          borderColor: isCPU
            ? cssVariableColor('--color-primary', 0.3)
            : cssVariableColor('--color-secondary', 0.3),
          borderWidth: 1,
          titleColor: textColor,
          bodyColor: cssVariableColor('--color-on-surface'),
          titleFont: { family: 'Inconsolata', size: 14 },
          bodyFont: { family: 'Inconsolata', size: 14 },
          cornerRadius: 8,
          padding: 12,
          displayColors: true,
          boxWidth: 12,
          boxHeight: 12,
          boxPadding: 2,
          intersect: false,
          mode: 'index',
          callbacks: {
            title: (context) => {
              const xValue = context[0]?.parsed?.x
              if (xValue === null || xValue === undefined) return ''
              const timestamp = new Date(xValue)
              return timestamp.toLocaleString('en-US', {
                month: 'short',
                day: 'numeric',
                hour: '2-digit',
                minute: '2-digit',
                hour12: false,
              })
            },
            label: (context) => {
              const yValue = context.parsed.y
              if (yValue === null || yValue === undefined) return 'N/A'
              return isCPU
                ? formatCPU(yValue)
                : formatBytes(yValue)
            },
          },
        },
      },
      scales: {
        x: {
          type: 'time',
          time: {
            minUnit: 'minute',
            displayFormats: {
              millisecond: 'HH:mm:ss.SSS',
              second: 'HH:mm:ss',
              minute: 'HH:mm',
              hour: 'MMM d, HH:mm',
              day: 'MMM d, HH:mm',
              week: 'MMM d',
              month: 'MMM yyyy',
              quarter: 'MMM yyyy',
              year: 'yyyy',
            },
          },
          ticks: {
            color: textColor,
            font: { family: 'Inconsolata', size: 13 },
            maxTicksLimit: 10,
            maxRotation: 0,
            autoSkip: true,
          },
          grid: { color: gridColor },
        },
        y: {
          ticks: {
            color: textColor,
            font: { family: 'Inconsolata', size: 13 },
            callback: (value) => {
              return isCPU ? formatCPU(value as number) : formatBytes(value as number)
            },
          },
          grid: { color: gridColor },
          grace: '10%',
          title: {
            display: true,
            text: isCPU
              ? (useMillicores ? 'CPU (millicores)' : 'CPU (cores)')
              : 'Memory',
            color: textColor,
            font: { family: 'Inconsolata', size: 14 },
          },
        },
      },
    },
  })
}

watch(() => props.data, () => {
  nextTick(() => {
    initChart()
  })
}, { immediate: true })

// Watch the color mode and re-initialize the chart if it changes
const colorMode = useColorMode({ storageKey: 'mochi-theme' })
watch(colorMode, () => {
  if (chartInstance) {
    nextTick(() => {
      initChart()
    })
  }
})

onBeforeUnmount(() => {
  if (chartInstance) {
    chartInstance.destroy()
    chartInstance = null
  }
})
</script>
