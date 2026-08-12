# Task Queue — Slice 1

The ordered build queue for [00-overview.md](../00-overview.md)'s first vertical slice. Work
them in numerical order unless a task's **Depends on** says otherwise.

## The queue

| # | Task | Area |
|---|---|---|
| [T-01](T-01-supabase-project.md) | Supabase project setup | infra |
| [T-02](T-02-expo-scaffold.md) | Expo app scaffold | app |
| [T-03](T-03-supabase-client-session.md) | Supabase client, `SessionProvider`, route gate | app |
| [T-04](T-04-auth-screens.md) | Login / signup / sign-out screens | app |
| [T-05](T-05-go-service-scaffold.md) | Go service scaffold — config, router, middleware, `/healthz` | api |
| [T-06](T-06-schema-migrations.md) | Schema migrations (goose) | api |
| [T-07](T-07-sqlc-store.md) | sqlc + pgx store wiring | api |
| [T-08](T-08-jwt-middleware.md) | JWT verification middleware + auth context | api |
| [T-09](T-09-nutrition-scaling.md) | `nutrition/scale.go` + table tests | api |
| [T-10](T-10-profile-endpoints.md) | Profile endpoints (`/me`) | api |
| [T-11](T-11-food-endpoints.md) | Food endpoints (search, get, create) | api |
| [T-12](T-12-diary-endpoints.md) | Diary endpoints (day view, create, patch, delete) | api |
| [T-13](T-13-app-api-client.md) | App API client + zod schemas | app |
| [T-14](T-14-seed-foods.md) | Seed foods | api |
| [T-15](T-15-diary-screen.md) | Diary screen | app |
| [T-16](T-16-first-deploy.md) | First production deploy — Fly + Netlify | infra |
| [T-17](T-17-add-food-screen.md) | Add-food screen (search + quantity) | app |
| [T-18](T-18-create-food-screen.md) | Create-food screen | app |
| [T-19](T-19-edit-delete-entry.md) | Edit quantity / delete entry | app |
| [T-20](T-20-settings-screen.md) | Settings screen (calorie goal) | app |
| [T-21](T-21-acceptance-run.md) | Test pass + production acceptance run | both |

## Milestones

**T-01 → T-05: both processes run locally.** The app boots, a user can sign up and log in, and
the Go service answers `/healthz`. Nothing is connected to anything yet.

**T-06 → T-12: the API is complete.** Every endpoint in
[03-api-contract.md](../03-api-contract.md) works and is testable with `curl` and a real token.

**T-13 → T-16: it is live.** The diary screen renders real data, and the whole chain —
Netlify → Fly → Supabase — is deployed and proven.

**T-17 → T-21: the slice closes.** The remaining screens ship continuously to the live
environment, and T-21 runs the acceptance check that
[00-overview.md](../00-overview.md)'s success criterion demands.

## Why T-16 deploys before the app is finished

The success criterion is a *production* criterion. Finding out on day one whether the
Netlify redirect, the Fly health check, the JWKS fetch, and the Supabase pooler all work
together is worth far more than discovering it after building four more screens — every one of
those is a configuration problem that takes an hour to find and a minute to fix, and they do not
get easier by being discovered together.

From T-17 onward, each task ships to the live environment as it is finished.

## Ordering notes

- **T-09 before T-10 to T-12.** The scaling functions are pure and testable; writing them before
  the handlers that call them means the arithmetic is settled before it is embedded in HTTP.
- **T-10 before T-11 and T-12.** `/me` is the simplest authenticated endpoint. It is where a
  real token first travels the whole path — app → Netlify → Fly → JWKS → Postgres — and it is a
  much better place to debug that path than a handler with real logic in it.
- **T-13 before T-15.** The diary screen is built against typed, validated calls, not raw
  `fetch`.
- **T-14 before T-15.** An empty `foods` table makes the diary screen impossible to test by
  hand.
- **T-03 and T-04 before T-05** is a soft ordering. They are independent — the app's auth
  touches Supabase directly and never the Go service. Swap them if you would rather start in Go.

## Working a task

Each file is: **Goal** (one sentence), **Depends on**, **Steps**, **Done when** (checkable), and
**Spec references** (deep links into the docs).

The specs in [docs/](../) are the authority. If a task and a spec disagree, the spec wins — and
if the spec is what is wrong, fix the spec first, then the task. A decision made in code and not
written down is a decision that gets re-litigated in three weeks.

A task is done when every **Done when** box is checkable, not when the code compiles.
