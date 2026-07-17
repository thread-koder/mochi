const ONE_DAY_MS = 24 * 60 * 60 * 1000
const ONE_YEAR_MS = 365 * ONE_DAY_MS

const TWO_HOURS_SEC = 2 * 60 * 60
const TWENTY_TWO_HOURS_SEC = 22 * 60 * 60
const TWENTY_EIGHT_DAYS_SEC = 28 * 24 * 60 * 60

export const timeAxisTickCount = (width: number): number => {
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

export const buildTimeAxisLabelFormatter = (minMs: number, maxMs: number, tickCount: number) => {
  const rangeMs = Math.max(maxMs - minMs, 1)
  const secPerTick = rangeMs / tickCount / 1000

  return (value: number) => formatTimeAxisLabel(value, rangeMs, secPerTick)
}
