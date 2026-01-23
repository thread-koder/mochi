import type { FetchError } from 'ofetch'
import type { NuxtError } from '#app'

interface ParsedError {
  status: number
  statusText: string
  message: string
}

export const useApiError = () => {
  /**
   * Parses an API error and returns status, statusText, and message
   * @param error - NuxtError or FetchError
   * @param fallback - Default message
   * @returns Object with status, statusText, and message
   */
  const parseError = (error: NuxtError | FetchError | null | undefined, fallback = 'Request failed'): ParsedError => {
    if (!error) {
      return {
        status: 500,
        statusText: 'Internal Server Error',
        message: fallback,
      }
    }

    const status = error.status ?? error.statusCode ?? 500
    const statusText = error.statusText ?? error.statusMessage ?? 'Fetch Failed'
    const errorData = error.data as { error?: string, details?: string } | undefined

    let message = fallback

    if (errorData) {
      if (errorData.details) {
        message = `${errorData.error || fallback}: ${errorData.details}`
      }
    }
    else {
      message = error.statusText ?? error.statusMessage ?? error.message ?? fallback
    }

    return {
      status,
      statusText,
      message,
    }
  }

  return {
    parseError,
  }
}
