import { Redirect } from 'expo-router';

/**
 * `/` is not a screen — the diary is home.
 *
 * T-03 makes this conditional on the session: the (app) group's gate redirects to
 * /login when there is none. Until then this always lands on the diary placeholder.
 */
export default function Index() {
  return <Redirect href="/diary" />;
}
