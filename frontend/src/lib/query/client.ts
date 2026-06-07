/**
 * TanStack Query client.
 *
 * Defaults:
 *   - 5s staleTime — UI doesn't refetch on every navigation, but the
 *     top status bar's polling queries override this.
 *   - 1 retry on network errors, 0 on 4xx (don't retry auth failures).
 *   - refetchOnWindowFocus: false — enterprise dashboards don't get
 *     yanked under the operator's mouse.
 */

import { QueryClient } from "@tanstack/react-query";
import { ApiError } from "@/lib/api/client";

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5_000,
      gcTime: 5 * 60_000,
      refetchOnWindowFocus: false,
      retry: (failureCount, error) => {
        if (error instanceof ApiError) {
          if (error.status >= 400 && error.status < 500) return false;
        }
        return failureCount < 1;
      },
    },
    mutations: {
      retry: false,
    },
  },
});
