const MIN_POINTS_FOR_ANALYSIS = 2

export const hasEnoughPoints = (count: number | undefined): boolean =>
  (count ?? 0) >= MIN_POINTS_FOR_ANALYSIS
