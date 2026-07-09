export const formatBytes = (value?: number): string => {
  if (value === undefined || isNaN(value)) {
    return 'N/A'
  }

  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let size = value
  let unitIndex = 0

  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024
    unitIndex++
  }

  if (unitIndex === 0) {
    return `${size} ${units[unitIndex]}`
  }

  return `${size.toFixed(2)} ${units[unitIndex]}`
}

export const formatPercentage = (value: number): string => {
  if (isNaN(value)) {
    return 'N/A'
  }
  return `${(value * 100).toFixed(1)}%`
}

export const formatChangePercent = (percent: number | null | undefined): string => {
  if (percent === null || percent === undefined) {
    return ''
  }
  const sign = percent >= 0 ? '+' : ''
  return `${sign}${percent.toFixed(1)}%`
}
