'use client';

import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useState } from 'react';

export function Providers({ children }: { children: React.ReactNode }) {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            // M12.5: resilience defaults. Prefer a small number of retries with exponential backoff.
            retry: 2,
            retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 10_000),
            // Avoid surprising refetches while the operator is reading.
            refetchOnWindowFocus: false
          },
          mutations: {
            // Mutations should fail fast; callers can retry explicitly.
            retry: false
          }
        }
      })
  );

  return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
}


