# 00 — Overview

## The product

A nutrition tracker in the mold of MyFitnessPal. A person signs in, picks a day, and records
what they ate at each meal. The app tells them how many calories that adds up to and how far
they are from their daily goal.

## Slice 1 scope

This is the first vertical slice. It is deliberately narrow — it must work end to end, in
production, before anything else gets built.

**In scope:**

- Email + password signup, login, and logout
- A diary screen for a single calendar day, split into breakfast / lunch / dinner / snack
- Searching a database of foods and adding one to a meal with a quantity
- Creating a custom food when search comes up empty
- Per-meal calorie subtotals, a day total, and remaining-against-goal
- Editing the quantity of a logged entry and deleting an entry
- Navigating between days
- Setting a daily calorie goal

**Explicitly not in scope** (these are real features, just not now):

- Barcode scanning
- Recipes, meals-as-templates, or copying a previous day
- Exercise, water, or weight logging
- Macro *goals* — macros are stored and summed, but only calories are goal-tracked
- Social features, streaks, reminders, notifications
- Offline-first sync — the app requires a network connection
- Password reset via email (Supabase supports it; wiring the redirect is slice 2)
- Third-party food databases (USDA FoodData Central, Open Food Facts). Slice 1 ships a
  curated seed list of common foods; see [T-14](tasks/T-14-seed-foods.md).

## Success criterion

One sentence, checkable in production: **a new user can sign up, log a banana to breakfast,
and see the calorie total on their diary — and it is still there after a hard refresh and on
a different device.**

## Stack decisions

The user set two constraints: React Native + TypeScript on the front end, and Netlify
hosting. The user also asked for a backend language other than C#, which they already know.
Everything below follows from those.

### Frontend: Expo (React Native + react-native-web) + TypeScript

React Native alone has no web target. Expo bundles `react-native-web` and can export a static
web build (`npx expo export --platform web`) that Netlify serves directly. The same source
tree runs on iOS and Android, which means the "web app" requirement and any future mobile app
share one codebase.

- **Expo Router** for navigation. It is file-based and produces real URLs on web —
  `/diary?date=2026-08-11` is a linkable address, which a stack navigator would not give us.
- **TanStack Query** for server state. The diary screen is a cache-and-invalidate problem:
  add an entry, invalidate the day, re-render totals. Doing that by hand with `useEffect` is
  where this kind of app usually rots.
- **react-hook-form + zod** for forms. The zod schemas double as runtime validation of API
  responses, so a contract drift shows up as a clear parse error instead of `undefined`
  appearing three components deep.

### Backend language: Go

The brief was "something different from C#, and I want to learn it."

Go is a good answer for someone with C# muscle memory. It is statically typed, compiled, and
garbage-collected, so the fundamentals transfer. But the things that *don't* transfer are the
interesting part of the learning: explicit error values instead of exceptions, structural
interfaces instead of declared implementation, composition instead of inheritance, and
goroutines/channels instead of `async`/`await`. The language spec is small enough to hold in
your head in a weekend, and the toolchain (`go build`, `go test`, `go fmt`) has no
configuration to fight.

Considered and set aside:

- **TypeScript on the backend** — would let the whole thing run on Netlify Functions, but
  teaches nothing new and shares no types-at-runtime guarantees anyway.
- **Rust** — more to learn, but the borrow checker would dominate the project. Learning Rust
  and shipping a nutrition app are two projects; this is the nutrition app.
- **Elixir/Phoenix** — genuinely different and excellent, but the ecosystem is smaller and
  the deployment story is heavier than a single Go binary in a scratch container.
- **Python/FastAPI** — fast to write, but dynamic typing after C# feels like a downgrade
  rather than a lesson.

### HTTP layer: Go standard library

Go 1.22 taught `http.ServeMux` method-aware patterns with path parameters
(`mux.HandleFunc("GET /api/v1/foods/{id}", …)`). That covers everything this API needs, so
there is no framework. Middleware is a `func(http.Handler) http.Handler` — writing that chain
by hand is one of the better ways to actually understand Go's interfaces.

`chi` is the drop-in alternative if the routing ever outgrows the stdlib; it uses the same
`http.Handler` interface, so switching is mechanical.

### Database: PostgreSQL, hosted on Supabase

Postgres because the data is relational and the day-view query is a `GROUP BY` that a
document store would make awkward. Supabase because its free tier includes both Postgres and
the auth service, so there is one vendor, one dashboard, and one connection string.

### Data access: pgx + sqlc

`sqlc` reads hand-written `.sql` files and generates typed Go structs and methods from them.
For someone coming from Entity Framework this hits the same ergonomic note — typed results,
compile-time safety — but inverts where the truth lives: you write the SQL, and the types are
derived from it. That is the right direction for learning, and it means the query plan is
never a surprise.

`GORM` is the EF-Core-shaped alternative. It is rejected here on purpose: it would let the
user avoid learning any SQL, which is the opposite of the goal.

`pgx` is the Postgres driver sqlc targets, and it handles Postgres types (numeric, uuid,
date, enums) properly rather than flattening them to strings.

### Migrations: goose

Plain `.sql` files with `-- +goose Up` / `-- +goose Down` markers, applied in order. It can
be embedded in the API binary so migrations run at deploy time, which removes a manual step
from every release.

### Auth: Supabase Auth

Signup, login, session storage, refresh-token rotation, and password hashing are all solved
problems with sharp edges, and getting them subtly wrong is how accounts get compromised.
Supabase Auth handles them; `@supabase/supabase-js` handles session persistence on both
native (AsyncStorage) and web (localStorage) without branching code.

Crucially, this does **not** outsource authorization. The Go API verifies the JWT itself
against Supabase's JWKS endpoint and enforces every ownership rule in code the user writes.
Supabase answers "who is this?"; the Go service answers "what are they allowed to do?".

### Why there is a Go backend at all

Supabase can expose tables directly over PostgREST with row-level security, which would make
the Go service redundant — the app could talk to Supabase and be done.

We are not doing that, for two reasons. First, learning Go is an explicit goal of the
project, and an API with real business rules is how you actually learn a language. Second,
the rules here (nutrition snapshotting, quantity scaling, day totals, entry ownership) belong
in code that can be read, tested, and changed, not spread across RLS policies and client
queries.

Supabase is used for exactly two things: hosted Postgres, and identity.

### Hosting: Netlify for the web build, Fly.io for the API

Netlify was a requirement, and it is a good fit for what it is doing — serving a static
bundle from a CDN. Netlify Functions, though, are a JavaScript/TypeScript-first serverless
runtime, and a long-lived Go service with a database connection pool is not what they are
built for.

So the split is: Netlify serves the static web export, and a `netlify.toml` redirect proxies
`/api/*` through to the Go service on Fly.io. From the browser's perspective every request is
same-origin, which means no CORS preflights and no cross-site cookie problems. Fly runs the
Go binary in a container with a health check and keeps a machine warm.

Render is the fallback if Fly's pricing model becomes annoying; its free tier cold-starts,
which is survivable for a personal project. See
[06-deployment.md](06-deployment.md).

## Repository layout

```
nutrition-tracker/
  app/      Expo app (React Native + TypeScript)
  api/      Go HTTP service
  docs/     These specifications, plus the task queue in docs/tasks/
```

One git repo, no monorepo tooling. `app/` has a `package.json`; `api/` has a `go.mod`. They
are independent builds that agree on the contract in
[03-api-contract.md](03-api-contract.md).

## Related documents

- [01-architecture.md](01-architecture.md) — runtime topology and request lifecycle
- [02-data-model.md](02-data-model.md) — schema
- [03-api-contract.md](03-api-contract.md) — endpoints
- [04-auth.md](04-auth.md) — identity and authorization
- [05-app-ui.md](05-app-ui.md) — screens
- [06-deployment.md](06-deployment.md) — hosting and environment variables
- [07-testing.md](07-testing.md) — test strategy
- [tasks/README.md](tasks/README.md) — the ordered build queue
