# Nutrition Tracker

A MyFitnessPal-style nutrition app. Log foods against a date and a meal, see calories add up
against a daily goal.

The app is an Expo (React Native + TypeScript) codebase that ships to iOS, Android, and the
web from one source tree. The API is a Go service backed by PostgreSQL. Identity comes from
Supabase Auth. The web build is hosted on Netlify; the API runs on Fly.io.

## Status

**Specification complete, implementation not started.** No application code has been written
yet. Everything in `docs/` describes what is going to be built, and
[docs/tasks/README.md](docs/tasks/README.md) is the ordered work queue — 21 tasks, starting with
[T-01](docs/tasks/T-01-supabase-project.md).

## Where to start

| If you want to know… | Read |
|---|---|
| What we're building and why the stack looks like this | [docs/00-overview.md](docs/00-overview.md) |
| How the pieces fit together at runtime | [docs/01-architecture.md](docs/01-architecture.md) |
| What the tables look like | [docs/02-data-model.md](docs/02-data-model.md) |
| What the API accepts and returns | [docs/03-api-contract.md](docs/03-api-contract.md) |
| How login and token verification work | [docs/04-auth.md](docs/04-auth.md) |
| What screens exist | [docs/05-app-ui.md](docs/05-app-ui.md) |
| How it gets deployed | [docs/06-deployment.md](docs/06-deployment.md) |
| How it gets tested | [docs/07-testing.md](docs/07-testing.md) |
| **What to build next** | **[docs/tasks/README.md](docs/tasks/README.md)** |

## Layout

```
app/    Expo app (React Native + TypeScript)   — scaffolded in T-02
api/    Go HTTP service                        — scaffolded in T-05
docs/   Specifications and the task queue
```

The two build independently — `app/` has a `package.json`, `api/` has a `go.mod` — and agree on
the contract in [docs/03-api-contract.md](docs/03-api-contract.md).

## Running it (once T-01 through T-05 are done)

```sh
# API
cd api && go run ./cmd/api

# App — web
cd app && npx expo start --web

# App — native
cd app && npx expo start
```
