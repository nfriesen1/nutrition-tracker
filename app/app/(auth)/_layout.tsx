import { Stack } from 'expo-router';

/**
 * Auth group. Renders when there is no session.
 *
 * T-03 adds the mirror of the (app) gate here: an existing session redirects to
 * /diary, so a signed-in user cannot sit on the login screen.
 *
 * The group name is in parentheses, so it does not appear in the URL — the address
 * is /login, not /(auth)/login.
 */
export default function AuthLayout() {
  return <Stack screenOptions={{ headerShown: false }} />;
}
