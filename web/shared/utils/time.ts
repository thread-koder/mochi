/**
 * Returns a time ago string for a given timestamp.
 * @param timestamp - The timestamp to format.
 * @returns A human-readable time ago string.
 */
export const timeAgo = (timestamp: string) => {
  const date = new Date(timestamp)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffSecs = Math.floor(diffMs / 1000)
  const diffMins = Math.floor(diffSecs / 60)
  const diffHours = Math.floor(diffMins / 60)
  const diffDays = Math.floor(diffHours / 24)

  if (diffSecs < 60) {
    return 'just now'
  }
  if (diffMins < 60) {
    return `${diffMins} ${diffMins === 1 ? 'minute' : 'minutes'} ago`
  }
  if (diffHours < 24) {
    return `${diffHours} ${diffHours === 1 ? 'hour' : 'hours'} ago`
  }
  if (diffDays < 7) {
    return `${diffDays} ${diffDays === 1 ? 'day' : 'days'} ago`
  }

  return date.toLocaleDateString()
}

/**
 * Formats a duration string to a human-readable format.
 * @param duration - The duration string to format.
 * @returns A human-readable duration string.
 */
export const formatDuration = (duration: string): string => {
  const hoursMatch = duration.match(/(\d+)h/)
  const minutesMatch = duration.match(/(\d+)m/)
  const secondsMatch = duration.match(/(\d+)s/)

  const hours = hoursMatch && hoursMatch[1] ? parseInt(hoursMatch[1], 10) : 0
  const minutes = minutesMatch && minutesMatch[1] ? parseInt(minutesMatch[1], 10) : 0
  const seconds = secondsMatch && secondsMatch[1] ? parseInt(secondsMatch[1], 10) : 0

  if (minutes > 0 || seconds > 0) {
    return duration
  }

  if (hours >= 24 && hours % 24 === 0) {
    const days = Math.floor(hours / 24)
    return `${days} ${days === 1 ? 'day' : 'days'}`
  }

  if (hours > 0) {
    return `${hours} ${hours === 1 ? 'hour' : 'hours'}`
  }

  return duration
}
