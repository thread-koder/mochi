export const recommendationStatusTextClass = (status: string): string => {
  switch (status.toLowerCase()) {
    case 'applied':
      return 'text-success-light'
    case 'rejected':
      return 'text-error-light'
    case 'pending':
      return 'text-warning-light'
    case 'superseded':
      return 'text-on-surface-muted'
    default:
      return 'text-on-surface-secondary'
  }
}

export const recommendationStatusBadgeClass = (status: string): string => {
  switch (status.toLowerCase()) {
    case 'applied':
      return 'bg-success/20 text-success-light border-success/30'
    case 'rejected':
      return 'bg-error/20 text-error-light border-error/30'
    case 'pending':
      return 'bg-warning/20 text-warning-light border-warning/30'
    case 'superseded':
      return 'bg-on-surface-muted/20 text-on-surface-muted border-on-surface-muted/30'
    default:
      return 'badge-neutral'
  }
}
