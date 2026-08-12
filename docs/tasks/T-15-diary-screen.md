# T-15 — Diary screen

**Goal.** The home screen renders a real day: totals against the goal, four meal sections with
subtotals, and working date navigation.

**Depends on.** [T-13](T-13-app-api-client.md), [T-14](T-14-seed-foods.md).

## Steps

1. `src/components/` — `DateHeader`, `TotalsBar`, `MealSection`, `EntryRow`, `ErrorNotice`,
   `ScreenSkeleton`.
2. `app/(app)/diary.tsx` — one `useQuery` on `['diary', date]`. **No arithmetic on this screen**;
   the response already carries totals and subtotals.
3. **The date lives in the URL.** `useLocalSearchParams()` reads `?date=`; the arrows call
   `router.setParams`. A missing or unparseable param is replaced with `todayLocal()` on mount.
4. `placeholderData` (previous day's data, dimmed) so arrowing between days does not flash
   empty.
5. The states from [05-app-ui.md](../05-app-ui.md): skeleton rows on first load — not a spinner,
   so the layout does not jump; "Nothing logged" per empty meal, **never a full-screen empty
   state**; retryable inline error with the date header still interactive.
6. Over-goal: `remaining_calories` negative renders "N over" with a full amber bar. Amber, not
   red — going over is information.
7. Formatting — calories whole with a thousands separator, macros to one decimal with `g`.
8. The `+` on each meal header navigates to `/add-food?date=…&meal=…`. A gear navigates to
   `/settings`. Both get `accessibilityLabel`s, both at least 44 pt.
9. Component tests: `TotalsBar` including the over-goal state, `MealSection` including the empty
   state, `EntryRow`'s `1.5 × medium (118 g)` formatting.

## Done when

- [ ] A day with entries renders them under the right meals with correct subtotals and total.
- [ ] An empty day renders four meal headers with zeroes, not a blank screen.
- [ ] The arrows move a day at a time and **the URL changes with them**.
- [ ] `/diary?date=2026-08-11` pasted into the browser opens that day directly.
- [ ] Arrowing quickly through days does not flash an empty screen.
- [ ] Setting the goal below the day's total shows the over-goal state.
- [ ] Killing the API mid-session shows the error state, and Retry recovers once it is back.
- [ ] The screen renders correctly on a phone.

## Notes

The bookmarkable URL is the payoff for choosing Expo Router over a stack navigator — worth
confirming by hand.

There is no way to add an entry yet; [T-17](T-17-add-food-screen.md) supplies that. Seed the
database directly, or `curl` the create endpoint, to get rows on screen.

## Spec references

- [05-app-ui.md](../05-app-ui.md) — the layout sketch, every state, and the formatting rules
- [03-api-contract.md](../03-api-contract.md) — the day-view response
- [07-testing.md](../07-testing.md) — which components get tests and why only those
