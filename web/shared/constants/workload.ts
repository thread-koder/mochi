import type { Workload } from '#shared/types/namespace'
import type { WorkloadResponse } from '#shared/types/workload'

export const WORKLOAD_TYPE_OPTIONS: { value: string, label: string }[] = [
  { value: 'Deployment', label: 'Deployment' },
  { value: 'StatefulSet', label: 'StatefulSet' },
  { value: 'DaemonSet', label: 'DaemonSet' },
  { value: 'Job', label: 'Job' },
  { value: 'CronJob', label: 'CronJob' },
  { value: 'Pod', label: 'Pod' },
]

export const workloadTypeLabel = (workloadType: string): string =>
  WORKLOAD_TYPE_OPTIONS.find(option => option.value === workloadType)?.label ?? workloadType

export const workloadStatusLine = (workload: Workload | WorkloadResponse): string => {
  switch (workload.type) {
    case 'Deployment':
    case 'StatefulSet':
    case 'DaemonSet':
      return `Replicas: ${workload.status.ready}/${workload.status.replicas}`
    case 'Job':
      return `Active: ${workload.status.active} · Succeeded: ${workload.status.succeeded} · Failed: ${workload.status.failed}`
    case 'CronJob':
      return workload.status.suspend ? 'Suspended' : workload.status.schedule
    case 'Pod':
      return workload.status.phase
  }
}
