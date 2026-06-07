export const RECOMMENDATION_STATUS_OPTIONS: { value: string, label: string }[] = [
  { value: 'pending', label: 'Pending' },
  { value: 'applied', label: 'Applied' },
  { value: 'rejected', label: 'Rejected' },
  { value: 'superseded', label: 'Superseded' },
]

export const RECOMMENDATION_MODE_OPTIONS: {
  value: string
  label: string
  description: string
  icon: string
}[] = [
  {
    value: 'cost_optimized',
    label: 'Cost Optimized',
    description: 'Aggressive rightsizing from P95 usage, with larger allowed cuts.',
    icon: 'lucide:dollar-sign',
  },
  {
    value: 'burstable',
    label: 'Burstable',
    description: 'P95-based sizing with burst headroom (limit > request). Recommended default.',
    icon: 'lucide:activity',
  },
  {
    value: 'guaranteed',
    label: 'Guaranteed',
    description: 'Peak-based sizing with request equal to limit (Guaranteed QoS).',
    icon: 'lucide:circle-check',
  },
]

export const recommendationStatusLabel = (status: string): string =>
  RECOMMENDATION_STATUS_OPTIONS.find(option => option.value === status)?.label ?? status

export const recommendationModeLabel = (mode: string): string =>
  RECOMMENDATION_MODE_OPTIONS.find(option => option.value === mode)?.label ?? mode
