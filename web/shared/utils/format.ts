/**
 * Formats a CPU value to a human-readable string.
 * @param value - The CPU value to format.
 * @returns A human-readable string.
 */
export const formatCPU = (value?: number): string => {
  if (value === undefined || value === null || isNaN(value)) {
    return 'N/A'
  }

  if (value < 1) {
    const millicores = value * 1000
    return `${millicores < 1 ? millicores.toFixed(2) : Math.round(millicores)}m`
  }

  return `${value.toFixed(2)} cores`
}

/**
 * Formats a byte value to a human-readable string.
 * @param value - The byte value to format.
 * @returns A human-readable string.
 */
export const formatBytes = (value?: number): string => {
  if (value === undefined || value === null || isNaN(value)) {
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

/**
 * Formats a value to a percentage.
 * @param value - The value to format.
 * @returns A percentage string.
 */
export const formatPercentage = (value: number): string => {
  if (value === undefined || value === null || isNaN(value)) {
    return 'N/A'
  }
  return `${(value * 100).toFixed(1)}%`
}

/**
 * Formats a change percentage to a human-readable string.
 * @param percent - The change percentage to format.
 * @returns A human-readable string.
 */
export const formatChangePercent = (percent: number | null | undefined): string => {
  if (percent === null || percent === undefined) {
    return ''
  }
  const sign = percent >= 0 ? '+' : ''
  return `${sign}${percent.toFixed(1)}%`
}

/**
 * Formats a snake_case or lowercase value to Title Case
 * @param value - The string to format.
 * @returns A human-readable label string.
 */
export const formatTitleCase = (value: string): string => {
  if (!value) return ''
  return value.replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase())
}
