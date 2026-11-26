# Nutrition Tracker

A MyFitnessPal-style nutrition app. Log foods against a date and a meal, see calories add up
against a daily goal.

The app is an Expo (React Native + TypeScript) codebase that ships to iOS, Android, and the
web from one source tree. The API is a Go service backed by PostgreSQL. Identity comes from
Supabase Auth. The web build is hosted on Netlify; the API runs on Fly.io.

## Status

**In progress: 2 of 21 tasks done.** The project is built spec-first: everything in `docs/`
describes what is going to be built, and [docs/tasks/README.md](docs/tasks/README.md) is the
ordered work queue.

- [x] [T-01](docs/tasks/T-01-supabase-project.md) Supabase project setup (checked by `scripts/verify-t01.sh`)
- [x] [T-02](docs/tasks/T-02-expo-scaffold.md) Expo app scaffold: every route exists and renders a placeholder screen
- [ ] **Next:** [T-03](docs/tasks/T-03-supabase-client-session.md) Supabase client, session provider, route gate

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
app/    Expo app (React Native + TypeScript)   — scaffolded (T-02); screens are placeholders
api/    Go HTTP service                        — not started (T-05)
docs/   Specifications and the task queue
```

The two build independently — `app/` has a `package.json`, `api/` will have a `go.mod` — and agree on
the contract in [docs/03-api-contract.md](docs/03-api-contract.md).

## Running it

The app scaffold runs today; the API runs once T-05 lands.

```sh
# API (after T-05)
cd api && go run ./cmd/api

# App — web
cd app && npx expo start --web

# App — native
cd app && npx expo start
```
