export const RECOMMENDATION_STATUS_OPTIONS: { value: string, label: string }[] = [
  { value: 'pending', label: 'Pending' },
  { value: 'applied', label: 'Applied' },
  { value: 'rejected', label: 'Rejected' },
  { value: 'superseded', label: 'Superseded' },
]

export const RECOMMENDATION_MODE_OPTIONS: { value: string, label: string }[] = [
  { value: 'cost_optimized', label: 'Cost Optimized' },
  { value: 'burstable', label: 'Burstable' },
  { value: 'guaranteed', label: 'Guaranteed' },
]

export const recommendationStatusLabel = (status: string): string =>
  RECOMMENDATION_STATUS_OPTIONS.find(option => option.value === status)?.label ?? status

export const recommendationModeLabel = (mode: string): string =>
  RECOMMENDATION_MODE_OPTIONS.find(option => option.value === mode)?.label ?? mode
