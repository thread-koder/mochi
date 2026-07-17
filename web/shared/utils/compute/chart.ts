import type { ComposeOption } from 'echarts/core'
import type { LineSeriesOption } from 'echarts/charts'
import type {
  GridComponentOption,
  TooltipComponentOption,
} from 'echarts/components'
import { formatCPU } from '#shared/utils/compute/format'
import { formatBytes } from '#shared/utils/format'
import { cssVariableColor } from '#shared/utils/color'
import type { DataPoint } from '#shared/types/timeseries'

type ECOption = ComposeOption<
  | LineSeriesOption
  | GridComponentOption
  | TooltipComponentOption
>

const ONE_DAY_MS = 24 * 60 * 60 * 1000
const ONE_YEAR_MS = 365 * ONE_DAY_MS

const TWO_HOURS_SEC = 2 * 60 * 60
const TWENTY_TWO_HOURS_SEC = 22 * 60 * 60
const TWENTY_EIGHT_DAYS_SEC = 28 * 24 * 60 * 60

const timeAxisTickCount = (width: number): number => {
  return Math.max(2, Math.floor(width / 100))
}

const formatTimeAxisLabel = (
  timestamp: number,
  rangeMs: number,
  secPerTick: number,
): string => {
  const date = new Date(timestamp)

  if (secPerTick <= 45) {
    return date.toLocaleString('en-US', {
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: false,
    })
  }
  if (secPerTick <= TWO_HOURS_SEC || rangeMs <= ONE_DAY_MS) {
    if (date.getHours() === 0 && date.getMinutes() === 0) {
      return date.toLocaleString('en-US', {
        month: 'short',
        day: 'numeric',
      })
    }
    return date.toLocaleString('en-US', {
      hour: '2-digit',
      minute: '2-digit',
      hour12: false,
    })
  }
  if (secPerTick <= TWENTY_TWO_HOURS_SEC) {
    return date.toLocaleString('en-US', {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
      hour12: false,
    })
  }
  if (secPerTick <= TWENTY_EIGHT_DAYS_SEC || rangeMs <= ONE_YEAR_MS) {
    return date.toLocaleString('en-US', {
      month: 'short',
      day: 'numeric',
    })
  }
  return date.toLocaleString('en-US', {
    month: 'short',
    year: 'numeric',
  })
}

const buildTimeAxisLabelFormatter = (minMs: number, maxMs: number, tickCount: number) => {
  const rangeMs = Math.max(maxMs - minMs, 1)
  const secPerTick = rangeMs / tickCount / 1000

  return (value: number) => formatTimeAxisLabel(value, rangeMs, secPerTick)
}

export const buildLineChartOption = (params: {
  type: 'cpu' | 'memory'
  data: DataPoint[]
  title: string
  width: number
}): ECOption => {
  const { type, data, title, width } = params
  const isCPU = type === 'cpu'

  const seriesData: [number, number][] = data.map(dp => [
    new Date(dp.timestamp).getTime(),
    dp.value,
  ])

  const maxValue = seriesData.length > 0
    ? Math.max(...seriesData.map(d => d[1]))
    : 0
  const yMax = maxValue > 0 ? maxValue * 1.1 : 1

  const timestamps = seriesData.map(d => d[0])
  const timeMin = timestamps.length > 0 ? Math.min(...timestamps) : Date.now()
  const timeMax = timestamps.length > 0 ? Math.max(...timestamps) : Date.now()
  const xSplitNumber = timeAxisTickCount(width)

  const fontFamily = 'Inconsolata'
  const seriesColor = isCPU ? '--color-primary' : '--color-info'
  const lineColor = cssVariableColor(seriesColor)
  const fillColor = cssVariableColor(seriesColor, 0.1)
  const gridColor = cssVariableColor(seriesColor, 0.1)
  const textColor = cssVariableColor('--color-on-surface-secondary')
  const tooltipBgColor = cssVariableColor('--color-surface-elevated')
  const tooltipBorderColor = cssVariableColor(seriesColor, 0.3)
  const tooltipBodyColor = cssVariableColor('--color-on-surface')
  const axisPointerColor = cssVariableColor(seriesColor, 0.4)

  return {
    animation: false,
    grid: {
      left: 50,
      right: 8,
      top: 8,
      bottom: 8,
    },
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'line',
        snap: true,
        lineStyle: {
          color: axisPointerColor,
        },
        label: { show: false },
      },
      backgroundColor: tooltipBgColor,
      borderColor: tooltipBorderColor,
      borderWidth: 1,
      borderRadius: 8,
      padding: 12,
      textStyle: {
        color: tooltipBodyColor,
        fontFamily: fontFamily,
        fontSize: 14,
      },
      formatter: (params) => {
        const items = Array.isArray(params) ? params : [params]
        const first = items[0]
        if (!first || first.value == null) return ''

        const value = first.value as [number, number]
        const timestamp = new Date(value[0]).toLocaleString('en-US', {
          month: 'short',
          day: 'numeric',
          hour: '2-digit',
          minute: '2-digit',
          hour12: false,
        })
        const formatted = isCPU ? formatCPU(value[1]) : formatBytes(value[1])
        return `<div style="color:${textColor}">${timestamp}</div>
          <div style="margin-top:4px;color:${tooltipBodyColor}">${formatted}</div>`
      },
    },
    xAxis: {
      type: 'time',
      splitNumber: xSplitNumber,
      axisLabel: {
        color: textColor,
        fontFamily: fontFamily,
        fontSize: 13,
        hideOverlap: true,
        formatter: buildTimeAxisLabelFormatter(timeMin, timeMax, xSplitNumber),
      },
      axisLine: {
        lineStyle: { color: gridColor },
      },
      axisTick: { show: false },
      splitLine: {
        show: true,
        lineStyle: {
          color: gridColor,
          width: 1,
        },
      },
    },
    yAxis: {
      type: 'value',
      max: yMax,
      splitNumber: 6,
      axisLabel: {
        color: textColor,
        fontFamily: fontFamily,
        fontSize: 13,
        showMinLabel: true,
        showMaxLabel: true,
        formatter: (value: number) => {
          return isCPU ? formatCPU(value) : formatBytes(value)
        },
      },
      axisLine: { show: false },
      axisTick: { show: false },
      splitLine: {
        show: true,
        lineStyle: {
          color: gridColor,
          width: 1,
        },
      },
    },
    series: [
      {
        type: 'line',
        name: title,
        data: seriesData,
        showSymbol: false,
        sampling: 'lttb',
        smooth: 0.3,
        lineStyle: {
          color: lineColor,
          width: 2,
        },
        itemStyle: {
          color: lineColor,
        },
        areaStyle: {
          color: fillColor,
        },
        emphasis: {
          disabled: true,
        },
      },
    ],
  }
}
