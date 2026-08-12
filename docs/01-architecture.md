# 01 — Architecture

## Runtime topology

```
                    ┌──────────────────────────────┐
                    │  Browser / iOS / Android     │
                    │  Expo app (React Native TS)  │
                    └──────┬────────────────┬──────┘
                           │                │
           auth calls      │                │  data calls
    (signup/login/refresh) │                │  Authorization: Bearer <access_token>
                           │                │
                           ▼                ▼
              ┌─────────────────┐    ┌──────────────────┐
              │ Supabase Auth   │    │ Netlify (CDN)    │
              │ GoTrue          │    │ static web build │
              └────────┬────────┘    │ /api/* proxy ────┼──┐
                       │             └──────────────────┘  │
                       │ JWKS (public keys, cached)         │
                       │                                    ▼
                       │                        ┌────────────────────────┐
                       └───────────────────────▶│ Fly.io                 │
                                                │ Go API (net/http)      │
                                                │  ├ auth middleware     │
                                                │  ├ handlers            │
                                                │  └ sqlc store (pgx)    │
                                                └───────────┬────────────┘
                                                            │ Postgres wire
                                                            ▼
                                                ┌────────────────────────┐
                                                │ Supabase PostgreSQL    │
                                                │ profiles, foods,       │
                                                │ diary_entries          │
                                                └────────────────────────┘
```

Three things worth noticing in that picture:

1. **The app talks to two backends.** Supabase Auth for identity, the Go API for data. They
   are connected only by the JWT: Supabase mints it, the Go service verifies it.
2. **On web, the app never calls Fly directly.** Netlify's `/api/*` redirect proxies to Fly,
   so the browser sees one origin. On native there is no origin to preserve, so the app hits
   the Fly hostname directly. The API base URL is therefore a build-time config value that
   differs per platform — see [06-deployment.md](06-deployment.md).
3. **The Go service is the only writer to the application tables.** Nothing else inserts a
   diary entry.

## Request lifecycle

A representative request, `POST /api/v1/diary/entries`:

1. **App** — user taps "Add" on the add-food screen. A TanStack Query mutation fires.
2. **App** — the fetch wrapper reads the current Supabase session, attaches
   `Authorization: Bearer <access_token>`, and sends the request to `/api/v1/diary/entries`
   (web: same-origin, proxied by Netlify; native: the Fly hostname).
3. **Go — recovery middleware** turns a panic into a `500` with a logged stack trace instead
   of killing the process.
4. **Go — request logging middleware** assigns a request id and records method, path, status,
   and duration on the way out.
5. **Go — auth middleware** parses the bearer token, verifies its signature against the
   cached Supabase JWKS, checks `exp`/`iss`/`aud`, extracts the `sub` claim as the user id,
   and stores it in the request `context`. A bad or missing token short-circuits with `401`.
6. **Go — handler** decodes the JSON body, validates it (`quantity > 0`, `meal` is one of the
   four, `date` parses as `YYYY-MM-DD`), and rejects with `422` if not.
7. **Go — handler** loads the referenced food, confirming it is either a seed food or one
   this user owns. If not, `404`.
8. **Go — handler** computes the nutrition snapshot: each nutrient scaled by `quantity` and
   rounded to two decimals.
9. **Go — store (sqlc)** inserts the row with `user_id` taken from the context — never from
   the request body.
10. **Go** — responds `201` with the created entry.
11. **App** — the mutation's `onSuccess` invalidates the `['diary', date]` query key; the
    diary screen refetches and the subtotals update.

## Go service package layout

```
api/
  cmd/
    api/
      main.go              wire config → db pool → migrations → router → server; graceful shutdown
  internal/
    config/
      config.go            env var loading, with fail-fast on anything required and missing
    httpx/
      router.go            route table — the one place every endpoint is listed
      middleware.go        recovery, request logging, auth
      respond.go           writeJSON / writeError helpers; the single error envelope
      errors.go            sentinel API errors mapped to status codes
      health.go            GET /healthz
      me.go                profile handlers
      foods.go             food handlers
      diary.go             diary handlers
    auth/
      verifier.go          JWKS fetch + cache, token parse and claim validation
      context.go           UserIDFrom(ctx) / withUserID(ctx, id)
    nutrition/
      scale.go             quantity × per-serving nutrients → snapshot; pure and unit-tested
    store/
      store.go             pool construction, transaction helper
      queries/*.sql        hand-written SQL — the sqlc input
      generated/*.go       sqlc output; never edited by hand
  db/
    migrations/*.sql       goose migrations
  sqlc.yaml
  go.mod
```

Why it is shaped this way:

- **`internal/`** makes every package unimportable from outside the module. For a single
  service that is exactly right — nothing here is a library.
- **`cmd/api/main.go` is the only place that knows how the pieces connect.** Everything else
  receives its dependencies as parameters, which is what makes handlers testable without a
  running server.
- **`nutrition/` is pure functions with no I/O.** Scaling and rounding is the one piece of
  real logic in slice 1, and isolating it means it can be tested exhaustively with a table
  test — the idiomatic Go pattern, and a good first thing to write in the language.
- **`store/generated/` is machine-written.** Editing it by hand is a bug; regenerate with
  `sqlc generate` instead.

## Expo app layout

```
app/
  app/                            Expo Router — the file tree IS the route tree
    _layout.tsx                   root: providers + session gate
    (auth)/
      _layout.tsx
      login.tsx
      signup.tsx
    (app)/
      _layout.tsx                 redirects to /login when there is no session
      diary.tsx                   home
      add-food.tsx
      create-food.tsx
      settings.tsx
  src/
    api/
      client.ts                   fetch wrapper: base URL, bearer token, error envelope, 401 retry
      diary.ts                    typed diary calls + query keys
      foods.ts                    typed food calls + query keys
      me.ts
    auth/
      supabase.ts                 client construction and platform storage
      SessionProvider.tsx         session context + auth state subscription
    schemas/                      zod schemas — the runtime shape of the API contract
    components/                   MealSection, EntryRow, DateHeader, TotalsBar, …
    lib/
      date.ts                     local-calendar-date helpers (see below)
  app.json
  tsconfig.json
  package.json
```

## Two cross-cutting rules

These are called out here because they cut across every layer, and getting either wrong
produces bugs that look like something else entirely.

### Dates are local calendar dates

A diary day is the user's own calendar day. It is **not** a UTC timestamp, and it is not
derived from one on the server.

- The client computes `YYYY-MM-DD` from the device's local clock and sends it as a string.
- The server stores it in a Postgres `date` column and does no timezone conversion, ever.
- The server never calls `time.Now()` to decide which day an entry belongs to.

Get this wrong and a user in UTC−6 logging a 7pm dinner sees it land on tomorrow. Every date
that crosses the wire is a `YYYY-MM-DD` string; `src/lib/date.ts` is the only place that
converts between a `Date` object and that string.

### Nutrition is snapshotted at log time

When an entry is created, the scaled calories and macros are copied onto the `diary_entries`
row. Reads sum those columns; they do not join back to `foods` for numbers.

If a user later corrects a custom food's calorie count, last week's totals must not silently
change. Historical logs are a record of what was believed at the time. The join to `foods`
still happens on read, but only to fetch the display name — never the nutrition values.

Full details in [02-data-model.md](02-data-model.md).
