import { QueryClientProvider } from '@tanstack/react-query';
import { Stack } from 'expo-router';
import { useState } from 'react';

import { SessionProvider } from '@/auth/SessionProvider';
import { createQueryClient } from '@/lib/query-client';

/**
 * Root layout: providers, then the route tree.
 *
 * Order matters. SessionProvider sits *inside* QueryClientProvider because signing
 * out clears the query cache — see docs/05-app-ui.md.
 */
export default function RootLayout() {
  // useState rather than a module-level client: a client created at module scope is
  // shared across requests during static rendering, which would leak one visitor's
  // cache into another's HTML.
  const [queryClient] = useState(createQueryClient);

  return (
    <QueryClientProvider client={queryClient}>
      <SessionProvider>
        <Stack screenOptions={{ headerShown: false }} />
      </SessionProvider>
    </QueryClientProvider>
  );
}
