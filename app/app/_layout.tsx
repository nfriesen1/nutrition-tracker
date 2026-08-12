import { Stack } from 'expo-router';

/**
 * Root layout.
 *
 * T-03 wraps this in QueryClientProvider then SessionProvider, in that order —
 * signing out clears the query cache, so the session must live inside the client.
 * See docs/05-app-ui.md.
 */
export default function RootLayout() {
  return <Stack screenOptions={{ headerShown: false }} />;
}
