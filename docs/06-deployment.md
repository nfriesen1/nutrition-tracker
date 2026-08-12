# 06 — Deployment

Three hosted pieces: the Expo web export on Netlify, the Go binary on Fly.io, and Postgres plus
Auth on Supabase. Native builds are out of slice 1's scope — the app runs on a device through
Expo Go during development, and App Store distribution is a later concern.

## Environments

Two, and no staging tier:

| | Local | Production |
|---|---|---|
| App | `npx expo start` | Netlify |
| API | `go run ./cmd/api` on `:8080` | Fly.io |
| Database | **The same Supabase project** | Supabase |

Slice 1 shares one Supabase project between local and production. That is a real compromise —
a bad local migration hits production data — and it is accepted because a second project means
a second set of auth keys, a second seed run, and twice the configuration for a project with
one developer and no users. The mitigations: migrations are additive and reviewed, and the
production acceptance run in [07-testing.md](07-testing.md) happens after every deploy.

The moment there is a second user, this splits. Doing it later is cheap: a new Supabase project,
a different `DATABASE_URL` in `.env.local`, and the migrations re-run from scratch.

---

## Environment variables

Three groups, and the split matters because one of them is public.

### Go service — set as Fly secrets

| Variable | Example | Notes |
|---|---|---|
| `DATABASE_URL` | `postgresql://postgres.<ref>:<pw>@aws-0-<region>.pooler.supabase.com:6543/postgres` | **Secret.** Use the pooler URL, not the direct one — see below |
| `SUPABASE_JWKS_URL` | `https://<ref>.supabase.co/auth/v1/.well-known/jwks.json` | Not secret, but set here so it moves with the environment |
| `SUPABASE_JWT_ISS` | `https://<ref>.supabase.co/auth/v1` | Checked against the token's `iss` |
| `SUPABASE_JWT_AUD` | `authenticated` | Checked against the token's `aud` |
| `PORT` | `8080` | Fly sets this; the config must read it rather than hardcoding |
| `ENV` | `production` | Switches log output to JSON and suppresses error detail in responses |

If T-01 finds the project is on the legacy shared-secret signing mode, `SUPABASE_JWKS_URL` is
replaced by `SUPABASE_JWT_SECRET` — a real secret, and the reason the asymmetric mode is
preferred. See [04-auth.md](04-auth.md).

`internal/config/config.go` loads these once at startup and **fails fast on anything required
and missing**, listing every missing variable in one error rather than dying on the first. A
service that boots with no `DATABASE_URL` and returns `500`s is worse than one that refuses to
start.

### Expo app — set in Netlify's build environment

| Variable | Notes |
|---|---|
| `EXPO_PUBLIC_SUPABASE_URL` | `https://<ref>.supabase.co` |
| `EXPO_PUBLIC_SUPABASE_ANON_KEY` | The anon/publishable key |
| `EXPO_PUBLIC_API_URL` | Empty on web (same-origin), the Fly hostname for native builds |

> **`EXPO_PUBLIC_*` values are compiled into the client bundle. They are public.** Anyone can
> read them out of the shipped JavaScript. The anon key is designed for that. **The Supabase
> service-role key must never be given this prefix, never appear under `app/`, and never enter
> this repository** — it bypasses every rule in [04-auth.md](04-auth.md).

Locally these live in `app/.env.local`, which `.gitignore` already covers.

### Netlify build

| Variable | Value |
|---|---|
| `NODE_VERSION` | `24` — already pinned in `netlify.toml` |

---

## The API base URL problem

The one place the app must know what platform it is on.

**On web** the browser calls `/api/v1/...` — same origin. Netlify's redirect proxies it to Fly.
No CORS preflight, no cross-site cookie question, and the Fly hostname never appears in the
client bundle.

**On native** there is no origin to preserve and no proxy in front of the app. The request must
go to the absolute Fly hostname.

`src/api/client.ts` resolves it once:

```ts
export const API_BASE_URL =
  Platform.OS === 'web'
    ? '/api'
    : `${process.env.EXPO_PUBLIC_API_URL}/api`;
```

Every call goes through this module, so this is the only branch. A component that builds a URL
itself will work on web and fail on device, which is a slow bug to find — hence one wrapper.

**Local development on a physical device** is the sharp edge: `localhost` on a phone means the
phone. Set `EXPO_PUBLIC_API_URL` to the development machine's LAN address
(`http://192.168.1.x:8080`) and make sure the Go service binds `0.0.0.0`, not `127.0.0.1`.

Because native calls Fly directly rather than through the proxy, **the Go service needs CORS
handling for exactly one case**: the Expo web dev server on `http://localhost:8081` calling a
local API on `:8080`. In production, web is same-origin and native sends no `Origin` header at
all. Keep the CORS middleware development-only, driven off `ENV`, rather than shipping a
permissive header to production.

---

## Netlify

`netlify.toml` is already committed at the repository root; T-16 replaces the placeholder
hostname. It does three things:

**Build.** `base = "app"`, `command = "npx expo export --platform web"`, `publish = "dist"`. The
`publish` path is relative to `base`, so the real directory is `app/dist`. Expo's web export
writes there by default.

**Proxy `/api/*` to Fly** with `status = 200` and `force = true`. Status 200 makes it a rewrite,
not a redirect — the browser never learns Fly exists. `force = true` makes the rule win over
any file that happens to exist at that path.

The hostname is hardcoded because Netlify does not interpolate environment variables inside
`netlify.toml`. The alternative, if this becomes annoying, is generating a `_redirects` file
during the build from an env var.

**SPA fallback last.** `/* → /index.html` with status 200, so a deep link to
`/diary?date=2026-08-11` serves the app rather than a 404.

> **Redirect order is load-bearing.** Netlify applies the first matching rule. A catch-all above
> the API proxy swallows every API call and answers it with the HTML shell — which surfaces as
> "Unexpected token '<' in JSON at position 0" and looks like an API bug. The comment in
> `netlify.toml` says this; leave it there.

Deploys are from `main`, and Netlify builds a preview for every pull request. Previews share the
same `/api` proxy and therefore the same production API and database — fine for a solo project,
and another thing that changes when the environments split.

---

## Fly.io

### Dockerfile

Multi-stage, in `api/Dockerfile`:

```dockerfile
FROM golang:1.23 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath \
      -ldflags="-s -w -X main.version=${GIT_SHA}" \
      -o /out/api ./cmd/api

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/api /api
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/api"]
```

- `go.mod`/`go.sum` are copied before the source so the dependency layer caches across builds.
- `CGO_ENABLED=0` gives a static binary, which is what makes the distroless base work.
- `distroless/static` over `scratch` because it brings CA certificates — the JWKS fetch and the
  TLS connection to Supabase both need them, and their absence produces an
  `x509: certificate signed by unknown authority` error that reads like a Supabase problem.
- `-X main.version` fills the `version` field of `/healthz`, so it is possible to tell what is
  actually running.

### `fly.toml`

```toml
app = "nutrition-tracker-api"
primary_region = "iad"   # match the Supabase project's region

[build]

[http_service]
  internal_port = 8080
  force_https = true
  auto_stop_machines = false
  auto_start_machines = true
  min_machines_running = 1

  [[http_service.checks]]
    path     = "/healthz"
    interval = "15s"
    timeout  = "2s"

[[vm]]
  size = "shared-cpu-1x"
  memory = "256mb"
```

`primary_region` should match the Supabase project's region. Every request does at least one
database round trip, and putting the API on a different continent from the database adds that
latency to every call for no benefit.

`min_machines_running = 1` and `auto_stop_machines = false` keep a machine warm. A scale-to-zero
cold start is several seconds on a request the user is waiting for, and this is the single
setting that decides whether the app feels responsive.

The health check hits `/healthz`, which — per [03-api-contract.md](03-api-contract.md) — does
**not** touch the database. A check that fails on a database blip would have Fly restart a
healthy process, and restarting does not fix Postgres.

Secrets are set with `fly secrets set` and are injected as environment variables at runtime.
They are never in `fly.toml`, which is committed.

### Database connections

Use Supabase's **connection pooler** (port `6543`), not the direct connection (`5432`). Supabase
caps direct connections tightly, and a Go service holding a pgx pool against the direct port
will exhaust them once there is more than one machine.

Set the pgx pool's `MaxConns` conservatively — 5 to 10 — in `internal/store/store.go`. Two
poolers stacked on each other with generous limits on both is how a small project runs out of
Postgres connections.

`prepared statement already exists` errors under the pooler in transaction mode are the known
gotcha; if they appear, disable pgx statement caching with `statement_cache_mode=describe` or
`default_query_exec_mode=simple_protocol` in the connection string.

---

## Migrations run at boot

goose migrations are embedded in the binary with `//go:embed` and applied in `cmd/api/main.go`
before the HTTP server starts listening. No separate migration step, no deploy checklist item
that can be skipped.

The risk this accepts: **a migration that fails takes the deploy down.** The binary exits, Fly's
health check fails, and Fly holds the previous version — so the outcome is "the deploy did not
land", not "the service is broken", which is the right failure. But it does mean a bad migration
blocks all deploys until it is fixed.

Mitigations for slice 1: migrations are additive (create table, create index, insert seed data),
none of them drop or retype a column, and goose's version table means a partially applied set
resumes rather than re-running.

When a genuinely destructive migration is needed, it stops being a boot-time concern and becomes
a two-deploy dance: ship code that tolerates both shapes, migrate, then ship code that assumes
the new shape.

---

## Deploy sequence

Order matters on the first deploy, and matters again whenever the contract changes.

**First deploy (T-16):**

1. Supabase project exists with auth configured (T-01).
2. `fly launch --no-deploy`, then `fly secrets set` for every variable in the Go table above.
3. `fly deploy`. Confirm `curl https://<app>.fly.dev/healthz` returns `200` and that the
   migration log lines appear in `fly logs`.
4. Put the real Fly hostname into `netlify.toml`'s redirect.
5. Connect the repository to Netlify, set the three `EXPO_PUBLIC_*` variables and
   `NODE_VERSION`, deploy.
6. Confirm `curl https://<site>.netlify.app/api/v1/me` returns `401` — not `404`, and not HTML.
   A `404` means the proxy rule is not matching; HTML means the SPA fallback is above it.

**Every deploy after:** the API ships first, then the app. Additive API changes are backwards
compatible, so an old client keeps working against a new server for the minutes between the two
deploys. The reverse — app first — means a client calling an endpoint that does not exist yet.

There is no automated CI/CD for the API in slice 1; `fly deploy` is run by hand. Netlify deploys
on push to `main` because it is wired that way by default, which is fine for a static bundle.

---

## Rollback

| Piece | How |
|---|---|
| Netlify | Publish a previous deploy from the dashboard. Instant |
| Fly | `fly releases` then `fly deploy --image <previous>` |
| Database | **No rollback.** goose down migrations exist and are tested locally, but rolling one back in production is a manual, considered act, not a routine one |

Because migrations are additive, rolling the API back to a previous release against a
newer schema works — the old code ignores columns it does not know about. This is the practical
reason the additive rule is worth keeping.

---

## Costs

| Service | Slice 1 | Notes |
|---|---|---|
| Netlify | Free | Static bundle, well inside the bandwidth allowance |
| Supabase | Free | Free-tier projects pause after a week of inactivity — a paused project is an outage, and waking it is a dashboard click |
| Fly.io | A few dollars a month | `min_machines_running = 1` is the reason it is not zero. This is the deliberate trade for no cold starts |

Render is the fallback named in [00-overview.md](00-overview.md) if Fly's billing becomes
irritating. Its free tier cold-starts, so the app would feel slower on the first request after
idle, which is survivable for a personal project. The Dockerfile is portable to it unchanged.

## Related documents

- [00-overview.md](00-overview.md) — why the hosting is split this way
- [01-architecture.md](01-architecture.md) — the runtime topology this deploys
- [03-api-contract.md](03-api-contract.md) — `/healthz`, and why it does not check the database
- [04-auth.md](04-auth.md) — what the Supabase variables are verified against
- [07-testing.md](07-testing.md) — the acceptance run against the deployed URL
