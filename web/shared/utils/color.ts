type ScoreColorOptions = {
  highThreshold?: number
  midThreshold?: number
  type?: 'text' | 'bg'
}

/**
 * Returns the color for the score.
 * @param value - The score value (0-1).
 * @param options - The options for the score color.
 * @returns A Tailwind CSS class string.
 */
export const scoreColor = (
  value: number | undefined | null,
  options?: ScoreColorOptions,
): string => {
  const highThreshold = options?.highThreshold ?? 0.8
  const midThreshold = options?.midThreshold ?? 0.5
  const type = options?.type ?? 'text'

  if (value === undefined || value === null) {
    return type === 'text' ? 'text-on-surface-secondary' : 'bg-on-surface-muted'
  }

  if (type === 'text') {
    if (value >= highThreshold) return 'text-success-light'
    if (value >= midThreshold) return 'text-warning-light'
    return 'text-error-light'
  }
  else {
    if (value >= highThreshold) return 'bg-success-light'
    if (value >= midThreshold) return 'bg-warning-light'
    return 'bg-error-light'
  }
}

type ScoreBadgeOptions = {
  highThreshold?: number
  midThreshold?: number
}

/**
 * Returns the combined bg/text classes for the score badge.
 * @param value - The score value (0-1).
 * @param options - The options for the score color.
 * @returns A Tailwind CSS class string.
 */
export const scoreBadgeClass = (
  value: number | undefined | null,
  options?: ScoreBadgeOptions,
): string => {
  const highThreshold = options?.highThreshold ?? 0.8
  const midThreshold = options?.midThreshold ?? 0.6

  if (value === undefined || value === null) {
    return 'bg-on-surface-muted/20 text-on-surface-muted'
  }

  if (value >= highThreshold) return 'bg-success-light/20 text-success-light'
  if (value >= midThreshold) return 'bg-warning-light/20 text-warning-light'
  return 'bg-error-light/20 text-error-light'
}

/**
 * Returns Tailwind classes for a recommendation status badge
 * @param status - The recommendation status string.
 * @returns Tailwind CSS class string for the badge
 */
export const statusBadgeClass = (status: string): string => {
  const s = status?.toLowerCase() ?? ''
  switch (s) {
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
