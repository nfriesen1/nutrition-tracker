import { QueryClient } from '@tanstack/react-query';

/**
 * Returns true for a 4xx thrown by the API client.
 *
 * Structural rather than an `instanceof` check, so this works before T-13 defines
 * ApiError and keeps working after — all it needs is a numeric `status`.
 */
export function isClientError(error: unknown): boolean {
  if (typeof error !== 'object' || error === null) return false;
  const status = (error as { status?: unknown }).status;
  return typeof status === 'number' && status >= 400 && status < 500;
}

export function createQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 30_000,
        // Never retry a 4xx. Retrying a 422 three times is three identical rejections
        // and a slower error message. See docs/05-app-ui.md.
        retry: (failureCount, error) => !isClientError(error) && failureCount < 2,
        refetchOnWindowFocus: true,
      },
    },
  });
}
