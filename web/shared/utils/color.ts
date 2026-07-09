type ScoreColorOptions = {
  highThreshold?: number
  midThreshold?: number
  type?: 'text' | 'bg'
  /** When true, higher values are worse (neutral → warning → error). */
  invert?: boolean
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
    if (options?.invert) {
      if (value > highThreshold) return 'text-error-light'
      if (value > midThreshold) return 'text-warning-light'
      return 'text-on-surface'
    }

    if (value >= highThreshold) return 'text-success-light'
    if (value >= midThreshold) return 'text-warning-light'
    return 'text-error-light'
  }
  else {
    if (value >= highThreshold) return 'bg-success'
    if (value >= midThreshold) return 'bg-warning'
    return 'bg-error'
  }
}

type ScoreBadgeOptions = {
  highThreshold?: number
  midThreshold?: number
}

export const scoreBadgeClass = (
  value: number | undefined | null,
  options?: ScoreBadgeOptions,
): string => {
  const highThreshold = options?.highThreshold ?? 0.8
  const midThreshold = options?.midThreshold ?? 0.6

  if (value === undefined || value === null) {
    return 'bg-on-surface-muted/20 text-on-surface-muted border-on-surface-muted/30'
  }

  if (value >= highThreshold) return 'bg-success/20 text-success-light border-success/30'
  if (value >= midThreshold) return 'bg-warning/20 text-warning-light border-warning/30'
  return 'bg-error/20 text-error-light border-error/30'
}

/**
 * Returns an empty string if not on the client-side.
 * Design tokens are expected to use modern CSS syntax (e.g. oklch), not bare hex.
 */
export const cssVariableColor = (variableName: string, opacity?: number): string => {
  if (!import.meta.client) return ''
  const root = document.documentElement
  const computedStyle = getComputedStyle(root)
  const value = computedStyle.getPropertyValue(variableName).trim()
  if (!value) return ''

  if (opacity !== undefined) {
    return value.replace(/\)$/, ` / ${opacity})`)
  }
  return value
}
