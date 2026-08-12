# 05 — App UI

Six screens. The route tree is fixed in [01-architecture.md](01-architecture.md); this document
says what each screen does, what it calls, and what it looks like when things are loading,
empty, or broken.

## Navigation map

```
                    ┌─────────────┐
        no session  │   /login    │◀────── sign out
       ┌───────────▶│             │
       │            └──────┬──────┘
       │                   │ "Create one"
       │            ┌──────▼──────┐
       │            │  /signup    │
       │            └──────┬──────┘
       │                   │ session established
  ┌────┴──────────────────▼────────────────────┐
  │              /diary?date=YYYY-MM-DD         │  ← home
  │                                             │
  └──┬──────────────┬───────────────┬───────────┘
     │ "+" on meal  │ gear icon     │ tap an entry
     ▼              ▼               ▼
 /add-food     /settings        (edit sheet, in place)
     │
     │ "Create a new food"
     ▼
 /create-food ──────▶ back to /add-food with the new food selected
```

Expo Router's two groups do the gating: `(auth)` renders when there is no session, `(app)`
redirects to `/login` when there is none. Group names in parentheses do not appear in the URL,
so the address is `/diary`, not `/(app)/diary`.

## The session gate

`app/(app)/_layout.tsx` is the only place that redirects on auth state:

```
session loading  →  render a splash, redirect NOTHING
no session       →  <Redirect href="/login" />
session          →  <Stack />
```

The loading state is not cosmetic. `supabase.auth.getSession()` reads from storage
asynchronously, so for the first frame after a cold start there is no session yet. Redirecting
on that frame bounces every returning user through the login screen. See
[04-auth.md](04-auth.md).

`app/_layout.tsx` above it mounts the providers, in this order: `QueryClientProvider`, then
`SessionProvider`. Session lives inside the query client because signing out clears the query
cache.

---

## `(auth)/login.tsx`

Email and password, a submit button, and a link to signup.

- **Calls:** `supabase.auth.signInWithPassword`. Not the Go API — login never touches it.
- **Form:** react-hook-form + zod. Email must parse as an email; password non-empty. Client
  validation exists to avoid a pointless round trip, not to enforce anything.
- **Success:** no navigation code. `onAuthStateChange` fires, the session context updates, the
  `(app)` gate stops redirecting, and the router lands on `/diary`. Navigating imperatively on
  success as well produces a double navigation.
- **Error:** Supabase returns a deliberately vague "Invalid login credentials" for both a wrong
  password and an unknown email — do not improve on that, it is what stops the form being an
  account-existence oracle. Shown above the form, not attached to a field.
- **Submitting:** button disabled, label "Signing in…". The email/password inputs stay enabled
  so a user can correct a typo they spot mid-request.

## `(auth)/signup.tsx`

Same shape, plus password confirmation.

- **Calls:** `supabase.auth.signUp`.
- **Validation:** password minimum 8 characters, matching confirmation. **Supabase's default
  minimum is 6, not 8** — verified against the live project in T-04 — so the form is stricter
  than the server. That direction is safe: everything the form accepts, the server accepts.
  The number lives in `PASSWORD_MIN_LENGTH` in `src/auth/schemas.ts`; raise the dashboard's
  "Minimum password length" to 8 to make them agree, or drop the constant to 6.

  The broken direction is the other one. A client looser than the server accepts a password,
  sends it, and gets back a server error the user cannot act on from the form.

  Login deliberately has **no** minimum — only non-empty. Login validates against what the
  account already has, and imposing the signup rule would lock out anyone whose password
  predates it.
- **Success:** email confirmation is off (T-01), so signup returns a live session and the flow
  is identical to login — the gate does the navigating.
- **Error:** "User already registered" is passed through. This *is* an account-existence oracle,
  and it is the standard trade every consumer signup makes: the alternative is a form that
  appears to succeed and sends nothing, which users read as a broken app.

No profile row is created here. The first `GET /api/v1/me` does it — see
[03-api-contract.md](03-api-contract.md).

---

## `(app)/diary.tsx` — home

The screen the app exists for.

```
┌────────────────────────────────────────────┐
│  ‹      Tuesday, 11 August      ›      ⚙   │   DateHeader
├────────────────────────────────────────────┤
│  1,488 / 2,000 kcal          512 remaining │   TotalsBar
│  ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓░░░░░░░░             │
├────────────────────────────────────────────┤
│  BREAKFAST                      158    +   │   MealSection
│    Banana, raw          1.5 × medium   158 │   EntryRow
│  ─────────────────────────────────────────  │
│  LUNCH                          640    +   │
│    …                                        │
│  ─────────────────────────────────────────  │
│  DINNER                         690    +   │
│  ─────────────────────────────────────────  │
│  SNACK                            0    +   │
│    Nothing logged                           │
└────────────────────────────────────────────┘
```

**Data:** one call, `GET /api/v1/diary?date=…`. The response already contains the goal, the
totals, all four meals in order, and each subtotal — so the screen does no arithmetic and needs
no second request. Meals with no entries still render their header and a "Nothing logged" row,
because `meals` is always four objects.

**The date lives in the URL.** `useLocalSearchParams()` reads `?date=`; the arrows call
`router.setParams`. On web `/diary?date=2026-08-11` is a real, linkable, bookmarkable address —
which is the reason [00-overview.md](00-overview.md) chose Expo Router over a stack navigator.
A missing or unparseable `date` param is replaced with today's local date on mount.

**`src/lib/date.ts` is the only converter** between a `Date` and a `YYYY-MM-DD` string. Every
other file passes the string around. This is the client half of the rule in
[02-data-model.md](02-data-model.md), and centralising it is what stops someone reaching for
`toISOString()` — which converts to UTC and silently shifts the date for anyone west of
Greenwich.

```ts
todayLocal(): string              // YYYY-MM-DD from the device clock
addDays(date: string, n: number): string
formatForDisplay(date: string): string   // "Tuesday, 11 August"
isToday(date: string): boolean
```

**States:**

| State | Render |
|---|---|
| Loading, first visit to a day | Skeleton rows in each meal section — not a spinner, so the layout does not jump |
| Loading, day already cached | Previous data, dimmed slightly. TanStack Query's `placeholderData` keeps the old day on screen while the new one loads, so arrowing through days does not flash empty |
| Empty day | Four meal headers, zero subtotals, "Nothing logged" per meal. Never a full-screen empty state — the meal headers are the affordance for adding |
| Error | Inline message with a Retry that refetches. The date header stays interactive so the user can navigate away from a day that is failing |
| Over goal | `remaining_calories` is negative; the bar fills and turns amber, the label reads "512 over". Not red — going over is information, not an error |

**Actions:** `+` on a meal header goes to `/add-food?date=…&meal=…`. Tapping an entry opens the
edit sheet (below). The gear opens `/settings`.

## The edit sheet

Not a route — a bottom sheet over the diary, opened by tapping an `EntryRow`. It has a quantity
input, Save, and Delete.

Deliberately not a screen: it is a one-field edit, and on web a route would put an entry id in
the address bar for something that is not worth linking to.

- **Save:** `PATCH /api/v1/diary/entries/{id}` with the new quantity.
- **Delete:** `DELETE /api/v1/diary/entries/{id}`, with a confirm on native and on web alike.
  There is no undo — [03-api-contract.md](03-api-contract.md) hard-deletes.
- Both invalidate `['diary', date]` and close the sheet.

Editing the quantity re-snapshots from the food's current values, so the numbers can change by
more than the quantity ratio. Worth a one-line note in the sheet only if it ever confuses
someone in practice — do not pre-emptively explain it.

---

## `(app)/add-food.tsx`

Reached from a meal's `+`. Carries `?date=` and `?meal=` through, so the screen knows where the
entry is going before a food is chosen.

**Search:** a text input that queries `GET /api/v1/foods?q=`, **debounced 300 ms**. The server
returns `[]` for fewer than two characters, and the client does not send the request at all
below that threshold — the check exists in both places because the server one is a guarantee
and the client one saves a round trip.

Results list the name, brand, serving label, and calories per serving. Custom foods sort first
and carry a "yours" badge, driven by `is_custom`.

**Selecting a food** reveals a quantity input, defaulted to `1`, with a live preview of the
scaled calories. The preview is computed on the client for responsiveness; the authoritative
numbers come back in the `201` response, and the client uses those, not its own. If the two
ever disagree, the server is right and the difference is a bug in
`nutrition/scale.go`'s client-side twin.

**Add** posts to `POST /api/v1/diary/entries`, then `router.back()` to the diary.

**States:**

| State | Render |
|---|---|
| Idle (no query) | The caller's recent foods would go here in a later slice; slice 1 shows a hint line |
| Fewer than 2 characters | The hint stays, no request fires |
| Searching | A slim progress line under the input. The previous results stay visible |
| No results | "No foods match '<query>'" plus a **Create "<query>" as a new food** button that navigates to `/create-food?name=<query>` with the name pre-filled |
| Error | Inline, retryable, input stays usable |

That empty-state button is the whole discovery path for custom foods. A user who searches for
something and finds nothing is exactly the user who needs to create one, and making them find a
separate menu item is how the feature goes unused.

## `(app)/create-food.tsx`

A form: name, optional brand, serving label, calories, protein, carbs, fat.

- **Pre-filled** from `?name=` when arriving from the empty search state.
- **Calls:** `POST /api/v1/foods`.
- **Validation:** mirrors the rules in [03-api-contract.md](03-api-contract.md) in zod. Macros
  default to `0` and are optional; only name, serving label, and calories are required, because
  demanding a full macro breakdown to log a homemade soup is how a user gives up.
- **`serving_label` needs a real hint**, not a bare label. Placeholder: `1 cup (240 g)`. This is
  the field people get wrong, and getting it wrong makes every future log of that food wrong.
- **Success:** invalidates `['foods']` and returns to `/add-food` with the new food selected and
  the quantity input already showing — the user came here to log something, not to curate a
  database.
- **`409` conflict:** "You already have a food called that." Attached to the name field.

## `(app)/settings.tsx`

Two things: the daily calorie goal, and sign out.

- **Data:** `GET /api/v1/me`. **Saves:** `PATCH /api/v1/me`.
- The goal input is a number field, `1`–`20000`, matching the server's bound.
- Saving invalidates `['me']` **and** `['diary']` — every day view embeds the goal, so a stale
  diary would show the old target next to the new one. This is the one non-obvious invalidation
  in the app.
- **Sign out** calls `supabase.auth.signOut()`, then `queryClient.clear()`. Clearing the cache
  is not optional: without it, the next user to sign in on that device sees the previous user's
  diary for a frame before the refetch lands.

---

## Server state: query keys and invalidation

TanStack Query owns everything that comes from the API. No `useEffect`-plus-`useState` fetching
anywhere — [00-overview.md](00-overview.md) names that as where this kind of app rots.

### Keys

| Key | Source | Notes |
|---|---|---|
| `['me']` | `GET /api/v1/me` | Fetched once on session start; contains the goal |
| `['diary', date]` | `GET /api/v1/diary?date=` | One entry per day, keyed by the `YYYY-MM-DD` string |
| `['foods', query]` | `GET /api/v1/foods?q=` | Keyed by the debounced query text |
| `['foods', 'id', id]` | `GET /api/v1/foods/{id}` | Deep-link path only |

Keying the diary by the date **string** rather than a `Date` object is what makes the cache work
at all — two `Date` objects for the same day are different values and would never hit the same
cache entry.

### Invalidation

| Mutation | Invalidates | Why |
|---|---|---|
| `POST /diary/entries` | `['diary', entry_date]` | The day it landed on |
| `PATCH /diary/entries/{id}` | `['diary', date]` | Subtotals and totals move |
| `DELETE /diary/entries/{id}` | `['diary', date]` | Same |
| `POST /foods` | `['foods']` (prefix) | Every cached search may now have a new match |
| `PATCH /me` | `['me']` **and** `['diary']` (prefix) | The goal is embedded in every day view |
| Sign out | `queryClient.clear()` | Not invalidation — removal |

Invalidating `['diary']` as a prefix hits every cached day; invalidating `['diary', date]` hits
one. The goal change is the only case that needs the prefix, and using it everywhere would
refetch days the user is not looking at.

**Mutations invalidate; they do not write to the cache optimistically.** Slice 1 requires a
network connection, the API is fast, and an optimistic diary update has to reproduce the
server's scaling and rounding on the client to be correct — which means two implementations of
[02-data-model.md](02-data-model.md)'s arithmetic that can drift. Not worth it here.

### Client defaults

```ts
new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry: (count, err) => !isClientError(err) && count < 2,
      refetchOnWindowFocus: true,
    },
  },
})
```

`retry` must not retry a `4xx`. Retrying a `422` three times is three identical rejections and
a slower error message. The `isClientError` helper reads the status off the error thrown by
`src/api/client.ts`.

## Contract validation with zod

Every response is parsed by a zod schema in `src/schemas/` before it reaches a component. The
inferred types are the app's types — there is no hand-written `interface DiaryDay` that could
drift from the schema.

A parse failure throws, surfacing as the screen's error state with a message naming the field.
That is the intended behaviour: a contract mismatch should be loud and immediate at the
boundary, not a crash somewhere further in when `entries.map` is called on `undefined`.

The schemas are also the app-side half of the contract test in
[07-testing.md](07-testing.md) — they parse recorded fixtures, so a server change that breaks
them fails in CI rather than in production.

## Presentation notes

Slice 1 uses React Native primitives and `StyleSheet`. No UI kit, no NativeWind. The screen
count is six and the component list is short; a design system would be more configuration than
code.

- **Components** (`src/components/`): `DateHeader`, `TotalsBar`, `MealSection`, `EntryRow`,
  `FoodResultRow`, `QuantityInput`, `ErrorNotice`, `ScreenSkeleton`.
- **Numbers:** calories display as whole numbers with a thousands separator (`1,488`); macros
  display to one decimal with a `g` suffix. The stored precision is two decimals — see
  [02-data-model.md](02-data-model.md) — and the extra digit is deliberately not shown.
- **Touch targets** are at least 44 pt. The `+` on a meal header and the date arrows are the
  ones that end up too small if this is not stated.
- **Numeric inputs** set `keyboardType="decimal-pad"` on native and `inputMode="decimal"` on
  web. A quantity field that opens a QWERTY keyboard is the most common way this kind of app
  feels wrong on a phone.
- **Accessibility:** every icon-only control (`+`, `‹`, `›`, gear) needs an `accessibilityLabel`.
  On web these render as real buttons, so keyboard focus order follows the DOM and needs no
  extra work.

## Related documents

- [01-architecture.md](01-architecture.md) — the file tree these routes live in
- [03-api-contract.md](03-api-contract.md) — the exact response shapes consumed here
- [04-auth.md](04-auth.md) — `SessionProvider` and the token attachment these screens rely on
- [07-testing.md](07-testing.md) — which of these components get tests
