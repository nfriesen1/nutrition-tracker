# T-20 — Settings screen

**Goal.** A user can set their daily calorie goal and sign out.

**Depends on.** [T-13](T-13-app-api-client.md), [T-15](T-15-diary-screen.md).

## Steps

1. `app/(app)/settings.tsx`, reached from the gear on the diary header.
2. Read `['me']`; render the goal in a number input bounded `1..20000`, matching the server.
3. Save → `PATCH /api/v1/me`.
4. **Invalidate `['me']` and `['diary']` as a prefix.** Every day view embeds the goal, so a
   stale diary shows the old target next to the new one. This is the one non-obvious
   invalidation in the app — see [05-app-ui.md](../05-app-ui.md).
5. Sign out → `supabase.auth.signOut()` then `queryClient.clear()`. Remove the temporary button
   [T-04](T-04-auth-screens.md) put on the diary.
6. Show the signed-in email as read-only context. There is nothing else to configure in slice 1;
   resist adding placeholder rows for features that do not exist.

## Done when

- [ ] The current goal loads into the input.
- [ ] Saving `2200` and returning to the diary shows remaining calculated against `2200` —
      **without a manual refresh**.
- [ ] Arrowing to a previously-viewed day also shows the new goal, confirming the prefix
      invalidation rather than a single-day one.
- [ ] `0`, a negative, and `99999` are all refused before a request is sent.
- [ ] Sign out returns to `/login`.
- [ ] **Signing in as a different user immediately afterwards shows their diary, with no flash
      of the previous user's data.**

## Notes

That last checkbox is what `queryClient.clear()` buys. Without it the next user sees the
previous user's cached diary for a frame — which is a privacy problem, not a cosmetic one, on a
shared device.

Macro goals are out of scope ([00-overview.md](../00-overview.md)). When they arrive they are
three more fields here and three nullable columns on `profiles`.

## Spec references

- [05-app-ui.md](../05-app-ui.md) — the screen and the invalidation table
- [03-api-contract.md](../03-api-contract.md) — `GET` and `PATCH /api/v1/me`
- [04-auth.md](../04-auth.md) — sign-out behaviour
