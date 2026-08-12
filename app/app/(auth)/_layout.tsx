import { Redirect, Stack } from 'expo-router';

import { useSession } from '@/auth/SessionProvider';
import { SessionSplash } from '@/components/session-splash';

/**
 * Auth group. The mirror of the (app) gate: a signed-in user cannot sit on the login
 * screen.
 *
 * This is also what makes login and signup need no imperative navigation on success.
 * The session updates, this gate stops matching, and the router lands on /diary.
 * Navigating manually as well produces a double navigation and a visible flicker —
 * see T-04.
 *
 * The group name is in parentheses, so it does not appear in the URL: the address is
 * /login, not /(auth)/login.
 */
export default function AuthLayout() {
  const { session, loading } = useSession();

  if (loading) return <SessionSplash />;
  if (session) return <Redirect href="/diary" />;

  return <Stack screenOptions={{ headerShown: false }} />;
}
