type RecommendationFieldState = 'change' | 'new' | 'unchanged' | 'empty'

const hasValue = (value: string | null | undefined): boolean => {
  return value != null && value !== ''
}

export const recommendationFieldState = (
  current: string | null | undefined,
  recommended: string | null | undefined,
  changePercent: number | null | undefined,
): RecommendationFieldState => {
  if (!hasValue(current) && !hasValue(recommended)) {
    return 'empty'
  }

  if (!hasValue(current) && hasValue(recommended)) {
    return 'new'
  }

  if (changePercent != null && changePercent !== 0) {
    return 'change'
  }

  return 'unchanged'
}
