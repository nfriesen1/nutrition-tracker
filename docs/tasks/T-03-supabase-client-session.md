# T-03 — Supabase client, `SessionProvider`, route gate

**Goal.** The app knows whether someone is signed in, keeps that knowledge current, and sends
signed-out users to `/login`.

**Depends on.** [T-01](T-01-supabase-project.md), [T-02](T-02-expo-scaffold.md).

## Steps

1. `src/auth/supabase.ts` — construct the client once from `EXPO_PUBLIC_SUPABASE_URL` and
   `EXPO_PUBLIC_SUPABASE_ANON_KEY`, with the platform storage branch and options from
   [04-auth.md](../04-auth.md). Throw at module load if either variable is missing; a client
   built with `undefined` fails later with a confusing error.
2. `src/auth/SessionProvider.tsx` — context exposing `{ session, user, loading }`:
   - `getSession()` on mount, `loading` true until it resolves
   - `onAuthStateChange` subscription for the component's lifetime
   - unsubscribe on unmount
3. `app/_layout.tsx` — mount `QueryClientProvider` then `SessionProvider`, with the client
   defaults from [05-app-ui.md](../05-app-ui.md) (`staleTime`, the no-retry-on-4xx rule,
   `refetchOnWindowFocus`).
4. `app/(app)/_layout.tsx` — the gate:
   - `loading` → render a splash and **redirect nothing**
   - no session → `<Redirect href="/login" />`
   - session → `<Stack />`
5. `app/(auth)/_layout.tsx` — the mirror: an existing session redirects to `/diary`, so a
   signed-in user cannot sit on the login screen.

## Done when

- [ ] Opening `/diary` while signed out lands on `/login`.
- [ ] **After signing in and hard-refreshing the browser, the app stays on `/diary` and does not
      flash the login screen.** This is the `loading` state working; it is the whole reason step
      4 says "redirect nothing".
- [ ] Signing out in one browser tab signs out the other within a second.
- [ ] Force-quitting and reopening the app on a device keeps the session — confirms
      AsyncStorage is wired.

## Notes

There is nothing to sign in with yet; use a user created through the Supabase dashboard in
[T-01](T-01-supabase-project.md), or do this task and [T-04](T-04-auth-screens.md) back to back.

None of this touches the Go API. Login is entirely between the app and Supabase — see
[04-auth.md](../04-auth.md).

## Spec references

- [04-auth.md](../04-auth.md) — client construction, storage, session lifecycle
- [05-app-ui.md](../05-app-ui.md) — the gate's three states and the query client defaults
