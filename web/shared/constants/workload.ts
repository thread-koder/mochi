export const WORKLOAD_TYPE_OPTIONS: { value: string, label: string }[] = [
  { value: 'Deployment', label: 'Deployment' },
  { value: 'StatefulSet', label: 'StatefulSet' },
  { value: 'DaemonSet', label: 'DaemonSet' },
  { value: 'Pod', label: 'Pod' },
]

export const workloadTypeLabel = (workloadType: string): string =>
  WORKLOAD_TYPE_OPTIONS.find(option => option.value === workloadType)?.label ?? workloadType
