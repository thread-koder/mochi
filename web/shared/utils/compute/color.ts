/**
 * Returns Tailwind classes for a recommendation status badge
 * @param status - The recommendation status string.
 * @returns Tailwind CSS class string for the badge
 */
export const recommendationStatusBadgeClass = (status: string): string => {
  switch (status.toLowerCase()) {
    case 'applied':
      return 'bg-success-light/20 text-success-light border-success-light/30'
    case 'rejected':
      return 'bg-error-light/20 text-error-light border-error-light/30'
    case 'pending':
      return 'bg-warning-light/20 text-warning-light border-warning-light/30'
    case 'superseded':
      return 'bg-on-surface-muted/20 text-on-surface-muted border-on-surface-muted/30'
    default:
      return 'bg-primary/20 text-primary-light border-primary/30'
  }
}
