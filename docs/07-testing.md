# 07 — Testing

## What this is trying to buy

This is a personal project with one developer. The test suite exists to catch three specific
things, not to hit a coverage number:

1. **Arithmetic that is wrong in a way nobody notices.** Scaling and rounding produce plausible
   numbers when they are broken.
2. **The two cross-cutting rules silently regressing.** Local-calendar dates and log-time
   snapshotting are both easy to undo with a reasonable-looking change.
3. **A missing ownership filter.** [04-auth.md](04-auth.md) states it plainly: there is no RLS
   backstop, so a forgotten `where user_id = $1` is a data breach. That deserves a test per
   owned resource.

Everything else is tested by using the app. There is no coverage target, and no E2E framework in
slice 1.

## The shape of it

```
        few │   production acceptance run (manual, after every deploy)
            │   ───────────────────────────────────────────────
            │   integration: migrations + real Postgres     (build-tagged)
            │   ───────────────────────────────────────────────
            │   handler tests: httptest + stub store
            │   component tests: RTL on the arithmetic-facing components
            │   ───────────────────────────────────────────────
       many │   unit: nutrition/scale.go table tests
```

---

## Go: unit tests

### `internal/nutrition/scale.go`

The one piece of real logic in slice 1, isolated as pure functions precisely so it can be tested
exhaustively — the rationale is already in [01-architecture.md](01-architecture.md).

Go's table test is the idiom, and it is a good first thing to write in the language:

```go
func TestScale(t *testing.T) {
    tests := []struct {
        name     string
        food     Nutrients
        quantity decimal.Decimal
        want     Nutrients
    }{
        {"whole serving", banana, dec("1"), Nutrients{105.00, 1.29, 26.95, 0.39}},
        {"half serving",  banana, dec("0.5"), ...},
        {"the worked example", banana, dec("1.5"), Nutrients{157.50, 1.94, 40.43, 0.59}},
        // ...
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := Scale(tt.food, tt.quantity)
            if got != tt.want { t.Errorf("Scale() = %v, want %v", got, tt.want) }
        })
    }
}
```

The cases that matter:

| Case | Why |
|---|---|
| The worked example from [02-data-model.md](02-data-model.md) | It is the documented contract; if the doc and the code disagree, one of them is wrong |
| `quantity = 1` | Must return the input unchanged, not a rounded version of it |
| Exact half-cent, `x.xx5` | Pins the rounding mode. Banker's rounding and half-away-from-zero differ here, and the doc says half-away-from-zero |
| A food with zero macros | Zeroes stay zeroes, no `NaN`, no `-0` |
| Large quantity (`1000`, the cap) | No overflow past `numeric(7,2)`'s range |
| Small quantity (`0.01`, two decimals) | A tiny fraction of a small food rounds to `0.00` and does not error |
| Fractions that do not terminate in binary (`0.1`, `0.3`) | The reason `numeric`/`decimal` is used rather than `float64` — this test fails loudly if someone "simplifies" it back |

That last row is the one worth being deliberate about. Use `shopspring/decimal` (or pgx's
`pgtype.Numeric`) end to end rather than converting to `float64` in the middle; a test with
`0.1 + 0.2` in it makes the reason permanent.

### `internal/auth`

Verification is security-critical and must be testable without a network.

Generate an ES256 key pair in the test, serve a JWKS from an `httptest.Server`, mint tokens with
the private key, and point the verifier at the test server. Then:

| Test | Expected |
|---|---|
| Valid token | Passes, `sub` lands in context |
| Expired `exp` | `401` |
| `exp` 20s past, within leeway | Passes |
| Wrong `iss` | `401` |
| Wrong `aud` | `401` |
| Signature from a different key | `401` |
| **`alg: none`** | `401` — the forgery case from [04-auth.md](04-auth.md) |
| **`alg` swapped to HS256, signed with the public key as the secret** | `401` — the other forgery case |
| Unknown `kid`, key present after refetch | Passes, and the refetch happened exactly once |
| Missing header, `Basic` scheme, `Bearer` with no token | `401` each |

The two algorithm-confusion tests are the reason this section exists. They are cheap to write
and they are what stops a refactor from quietly accepting forged tokens.

---

## Go: handler tests

`httptest` against the **real router**, with the store replaced by a stub. Routing through the
real `internal/httpx/router.go` means the test exercises the actual path pattern and the actual
middleware chain — a handler tested in isolation passes while the route is registered under the
wrong method.

This is where Go's structural interfaces earn their keep: the handlers take a store interface,
and a test struct with the right method set satisfies it with no mocking framework and no
registration.

```go
type stubStore struct {
    getFood func(ctx context.Context, id, userID uuid.UUID) (Food, error)
}
```

**One test per status branch in [03-api-contract.md](03-api-contract.md).** For
`POST /api/v1/diary/entries` that is:

- `201` happy path — and assert the response body's nutrition matches the scaled values
- `401` no token
- `422` missing `food_id`, bad `meal`, `quantity` of `0`, `quantity` negative, malformed
  `entry_date`, and `2026-02-30`
- `404` food id that the store reports as absent
- `400` body that is not JSON

Plus the two rule tests that belong here and nowhere else:

**Ownership.** For every owned resource, a test where user A requests user B's row and gets
`404` — asserting the status is `404`, **not** `403`, and that the body is byte-identical to a
genuinely-absent row's. Applies to `GET /foods/{id}`, `PATCH /diary/entries/{id}`, and
`DELETE /diary/entries/{id}`.

**`user_id` is never accepted from the body.** Post a diary entry with an extra
`"user_id": "<some other uuid>"` field and assert the row was created for the *authenticated*
user. This is a one-line test guarding the rule in [03-api-contract.md](03-api-contract.md).

---

## Go: integration test

One test, guarded by a build tag so `go test ./...` stays fast:

```go
//go:build integration
```

Run with `go test -tags=integration ./...` against a scratch Postgres — Docker locally, or a
throwaway Supabase project. It:

1. Runs the goose migrations from empty. **This is the only thing that tests the migrations at
   all**, and it is the main reason the test exists.
2. Inserts a food and three diary entries across two meals.
3. Reads the day view and asserts subtotals, totals, and ordering.
4. Reads the *adjacent* day and asserts it is empty — the boundary condition for `entry_date`.
5. Asserts the down migrations run cleanly in reverse.

Two rule tests belong here because they are about the database, not the handler:

**Dates do not shift.** Insert an entry for `2026-08-11`, read it back, assert the string is
`2026-08-11` — with the test process's `TZ` set to something aggressive like
`Pacific/Kiritimati` (UTC+14) or `Pacific/Midway` (UTC−11). If anything in the stack is doing a
timezone conversion, this fails. Run the same assertion under both.

**Snapshots do not move.** Insert an entry, then `update foods set calories = calories * 2`,
then read the day view. The total must be unchanged. This is the read-side half of
[02-data-model.md](02-data-model.md)'s central rule, and it is the test that catches a query
that joins to `foods` for nutrition values instead of names.

`sqlc` also provides a compile-time check for free: if a query and the schema disagree,
`sqlc generate` fails before any test runs.

---

## App tests

Vitest (or Jest, whichever `create-expo-app` scaffolds) plus `@testing-library/react-native`.

### Schema contract tests

The highest-value app test, and the cheapest.

Record a real response from each endpoint into `src/schemas/__fixtures__/*.json`, then assert
each zod schema parses its fixture. Because the schemas are also what the app parses responses
with at runtime, this is a genuine contract test: a server field rename fails here.

```ts
it('parses a diary day', () => {
  expect(() => diaryDaySchema.parse(diaryDayFixture)).not.toThrow();
});
```

Refresh the fixtures by hand when the contract changes — regenerating them automatically from
the running server would make the test tautological.

Include the negative cases too: a fixture with `meals` as an object rather than an array, and
one with `calories` as a string, both of which must fail to parse. Those are the two shapes a
careless server change would produce.

### `src/lib/date.ts`

Pure functions, so a table test again. The important cases:

| Case | Why |
|---|---|
| `todayLocal()` under `TZ=Pacific/Midway` and `TZ=Pacific/Kiritimati` | The two ends. Must return the *local* day, not the UTC one |
| `addDays` across a month boundary, a year boundary, and 29 February 2028 | Off-by-one territory |
| `addDays` across a DST transition | The result must move by one calendar day, not by 24 hours |
| `formatForDisplay` on a known date | Guards against `toISOString()` sneaking in |

The DST case is the one that catches a naive `new Date(ms + 86400000)` implementation.

### Component tests

Only where a component does arithmetic or formatting a human would not notice was wrong:

- **`TotalsBar`** — renders the goal, the total, and remaining; and renders the **over-goal**
  state correctly when `remaining_calories` is negative ("512 over", amber, bar full).
- **`MealSection`** — renders a subtotal, and renders the empty state when `entries` is `[]`.
- **`EntryRow`** — formats quantity and serving label together (`1.5 × medium (118 g)`) and
  rounds calories to a whole number.

Not tested: layout, styling, navigation wiring, anything that is visibly broken the moment the
app is opened.

### Not in slice 1

- **E2E (Playwright, Maestro, Detox).** The manual acceptance run below covers the same path for
  a fraction of the setup cost. This is the first thing to add if the app grows.
- **Mock Service Worker / a fake API.** The schema tests cover the contract, and the handler
  tests cover the server. A third layer of fakes between them would mostly test itself.
- **Snapshot tests.** They fail on every intentional change and get regenerated without being
  read.

---

## Production acceptance run

**This is a first-class test, not an afterthought.** It is the only thing that exercises
Netlify → Fly → Supabase together, and [00-overview.md](00-overview.md)'s success criterion is
explicitly a *production* criterion.

Run it against the deployed URL after every deploy that touches the API or the diary flow.

| # | Step | Pass condition |
|---|---|---|
| 1 | `curl https://<site>/api/v1/me` with no token | `401` with the JSON error envelope. **Not** `404`, **not** HTML — either means the redirect rules in `netlify.toml` are in the wrong order ([06-deployment.md](06-deployment.md)) |
| 2 | Sign up as a brand-new user | Lands on the diary, today's date, four empty meals |
| 3 | Search `ban`, add a banana to breakfast, quantity `1` | Entry appears; breakfast subtotal and day total both read `105` |
| 4 | **Hard refresh** | Entry still there. This is the "it actually persisted" check |
| 5 | Change the quantity to `1.5` | Reads `158`. Confirms the scale-and-round path in production |
| 6 | Navigate to yesterday, then back | Yesterday is empty; today still has the entry. The date-boundary check |
| 7 | Search for something absent, create it as a custom food, log it | Appears with a "yours" badge; totals include it |
| 8 | Set the calorie goal to `2200` in settings, return to the diary | Remaining recalculates against `2200` — the `['me']` + `['diary']` invalidation from [05-app-ui.md](05-app-ui.md) |
| 9 | Sign out, sign back in | Diary still shows the entries; no flash of the previous state |
| 10 | **Open the site on a different device or a private window**, sign in as the same user | Same diary. This is the second half of the success criterion, and the only step that proves the data is not in local storage |
| 11 | Check `fly logs` | No panics, no `500`s, request durations in the tens of milliseconds |

Step 10 is the one that is easy to skip and is the whole point. Step 1 is the one that catches
the most common deployment misconfiguration.

---

## Running the tests

```sh
# Go — unit and handler
cd api && go test ./...

# Go — including the integration test (needs a Postgres in DATABASE_URL_TEST)
cd api && go test -tags=integration ./...

# Go — race detector, worth running on the JWKS cache
cd api && go test -race ./...

# App
cd app && npm test
```

`go test -race` matters for exactly one thing: the JWKS cache in `internal/auth` is the only
shared mutable state in the service, guarded by an `RWMutex`. Run the auth tests under `-race`
at least once after writing it.

## CI

Slice 1 has no CI pipeline. It is one developer running two commands, and setting up GitHub
Actions before there is a suite worth gating on is ceremony.

When it arrives, it is one workflow: `go vet ./...`, `go test ./...`, `npm test`, and
`npx tsc --noEmit`. The integration test stays out of the default job until there is a Postgres
service container configured for it.

`go vet` and `gofmt -l` are worth running by hand from the start regardless — `vet` catches real
bugs (a `printf` verb mismatch, a lost `context`), and it costs nothing.

## Related documents

- [02-data-model.md](02-data-model.md) — the worked example the scale test pins
- [03-api-contract.md](03-api-contract.md) — the status branches the handler tests enumerate
- [04-auth.md](04-auth.md) — the forgery cases the verifier tests cover
- [05-app-ui.md](05-app-ui.md) — the invalidation behaviour step 8 checks
- [06-deployment.md](06-deployment.md) — what step 1 is really testing
