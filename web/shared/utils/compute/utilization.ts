import type { UtilizationResult } from '#shared/types/compute'
import { formatCPU } from '#shared/utils/compute/format'
import { formatBytes } from '#shared/utils/format'
import { hasEnoughPoints } from '#shared/utils/timeseries'

type UtilizationResource = 'cpu' | 'memory'

const utilizationResource = (resource: string): UtilizationResource => {
  return resource === 'memory' ? 'memory' : 'cpu'
}

export const utilizationMetricClass = (
  sampleSize: number | undefined,
  variant: 'primary' | 'secondary' = 'primary',
): string => {
  if (!hasEnoughPoints(sampleSize)) return 'text-on-surface-muted'
  return variant === 'secondary' ? 'text-on-surface-secondary' : 'text-on-surface'
}

export const utilizationSortMetricValue = (
  utilization: UtilizationResult,
  metric: string,
  resource: string,
): number => {
  const util = utilization[utilizationResource(resource)]
  if (!hasEnoughPoints(util.sample_size)) {
    return -1
  }

  switch (metric) {
    case 'current':
      return util.current
    case 'p95':
      return util.stats.percentile.p95
    case 'mean':
      return util.stats.mean
    case 'max':
      return util.stats.max
    default:
      return util.stats.percentile.p95
  }
}

export const formatUtilizationCPU = (
  value: number | undefined,
  sampleSize: number | undefined,
): string => (hasEnoughPoints(sampleSize) ? formatCPU(value) : 'N/A')

export const formatUtilizationBytes = (
  value: number | undefined,
  sampleSize: number | undefined,
): string => (hasEnoughPoints(sampleSize) ? formatBytes(value) : 'N/A')
