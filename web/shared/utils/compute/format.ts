export const formatCPU = (value?: number | null): string => {
  if (value === undefined || value === null || isNaN(value)) {
    return 'N/A'
  }

  if (value < 1) {
    const millicores = value * 1000
    if (millicores === 0) return '0m'

    const display = millicores >= 10
      ? Math.round(millicores).toString()
      : Number.isInteger(millicores) ? millicores.toString() : millicores.toFixed(1)

    return `${display}m`
  }

  const display = value >= 10
    ? Math.round(value).toString()
    : Number.isInteger(value) ? value.toString() : value.toFixed(1)

  return `${display} cores`
}
