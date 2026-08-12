import { Redirect, Stack } from 'expo-router';

import { useSession } from '@/auth/SessionProvider';
import { SessionSplash } from '@/components/session-splash';

/**
 * Authenticated group. The session gate.
 *
 * Three states, and the first one is the one that is easy to get wrong:
 *
 *   loading    -> splash, and redirect NOTHING
 *   no session -> /login
 *   session    -> render
 *
 * getSession() reads from storage asynchronously, so for the first frame after a cold
 * start there is no session yet. That is not the same as being signed out, and
 * redirecting on it bounces every returning user through the login screen. On web the
 * static HTML is rendered with no storage at all, so this frame always happens.
 *
 * See docs/04-auth.md.
 */
export default function AppLayout() {
  const { session, loading } = useSession();

  if (loading) return <SessionSplash />;
  if (!session) return <Redirect href="/login" />;

  return <Stack screenOptions={{ headerShown: false }} />;
}
