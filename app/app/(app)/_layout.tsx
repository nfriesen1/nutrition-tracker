import { Stack } from 'expo-router';

/**
 * Authenticated group.
 *
 * T-03 turns this into the session gate, which has three states and must get the
 * first one right:
 *
 *   loading      -> render a splash and redirect NOTHING
 *   no session   -> <Redirect href="/login" />
 *   session      -> <Stack />
 *
 * getSession() reads from storage asynchronously, so for the first frame after a
 * cold start there is no session yet. Redirecting on that frame bounces every
 * returning user through the login screen. See docs/04-auth.md.
 */
export default function AppLayout() {
  return <Stack screenOptions={{ headerShown: false }} />;
}
