# T-13 — App API client and zod schemas

**Goal.** The app talks to the Go API through one typed, validated wrapper.

**Depends on.** [T-03](T-03-supabase-client-session.md), [T-12](T-12-diary-endpoints.md).

## Steps

1. `src/api/client.ts`:
   - `API_BASE_URL` — the single `Platform.OS` branch from
     [06-deployment.md](../06-deployment.md). **No other file constructs a URL.**
   - Read the session and attach `Authorization: Bearer …` **per request**. Never cache the
     token in a module variable; `supabase-js` rotates it.
   - Parse the error envelope into a typed `ApiError` carrying `status`, `code`, `message`, and
     `details`.
   - On `401`: `refreshSession()` once, retry once, and on a second `401` sign out. **One retry,
     not a loop.**
2. `src/schemas/` — zod schemas mirroring [03-api-contract.md](../03-api-contract.md):
   `profileSchema`, `foodSchema`, `diaryEntrySchema`, `diaryDaySchema`. Export the inferred
   types; **do not hand-write a parallel `interface`.**
3. `src/api/me.ts`, `foods.ts`, `diary.ts` — typed call functions plus the query keys from
   [05-app-ui.md](../05-app-ui.md). Each parses its response through the schema before
   returning.
4. `src/lib/date.ts` — `todayLocal`, `addDays`, `formatForDisplay`, `isToday`. **The only place
   that converts between a `Date` and a `YYYY-MM-DD` string.** Never `toISOString()`.
5. Add `isClientError` for the query client's no-retry-on-4xx rule.
6. Tests: the schema fixtures and the `date.ts` table test from
   [07-testing.md](../07-testing.md), including the two extreme timezones and the DST case.

## Done when

- [ ] Every schema parses a fixture recorded from the running API.
- [ ] The negative fixtures — `meals` as an object, `calories` as a string — both **fail** to
      parse.
- [ ] `todayLocal()` returns the local day under `TZ=Pacific/Midway` and
      `TZ=Pacific/Kiritimati`.
- [ ] `addDays` moves by one calendar day across a DST transition, not by 24 hours.
- [ ] A deliberately expired token triggers exactly one refresh and one retry, and the call then
      succeeds.
- [ ] `grep -r "toISOString" src/` finds nothing outside `date.ts`.
- [ ] `grep -r "fetch(" src/` finds it only in `client.ts`.

## Notes

The two `grep` checkboxes are cheap and they are the enforcement mechanism for two rules that
are otherwise pure convention. Both failure modes — a UTC date shift, a URL that works on web
and breaks on device — are slow to diagnose after the fact.

Recording the fixtures by hand from the real API, rather than generating them, is what makes the
schema tests a genuine contract test rather than a tautology.

## Spec references

- [03-api-contract.md](../03-api-contract.md) — the shapes being mirrored
- [05-app-ui.md](../05-app-ui.md) — query keys and the zod-at-the-boundary rationale
- [06-deployment.md](../06-deployment.md) — the base URL branch
- [04-auth.md](../04-auth.md) — the token attachment and refresh rules
