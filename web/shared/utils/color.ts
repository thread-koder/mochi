type ScoreColorOptions = {
  highThreshold?: number
  midThreshold?: number
  type?: 'text' | 'bg'
}

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
