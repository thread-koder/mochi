import type { UtilizationResult } from '#shared/types/compute'

type UtilizationResource = 'cpu' | 'memory'

const utilizationResource = (resource: string): UtilizationResource => {
  return resource === 'memory' ? 'memory' : 'cpu'
}

export const utilizationSortMetricValue = (
  utilization: UtilizationResult,
  metric: string,
  resource: string,
): number => {
  const util = utilization[utilizationResource(resource)]

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
