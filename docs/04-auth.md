# 04 — Auth

## The split

**Supabase answers "who is this?". The Go service answers "what are they allowed to do?".**

That line is the whole design. Supabase Auth (GoTrue) owns password hashing, session issuance,
refresh-token rotation, and the token format. The Go service owns every rule about what a
verified user may read and write, and it enforces those rules in code that lives in this
repository.

What is explicitly *not* outsourced: authorization. There are no row-level security policies.
The Go service holds a direct Postgres connection with full table access and is trusted to
filter by `user_id` itself. This is deliberate — [00-overview.md](00-overview.md) rejects the
PostgREST-plus-RLS design because the rules belong somewhere they can be read and tested.

The consequence is worth stating out loud: **a missing `where user_id = $1` is a data breach,
not a bug.** There is no second line of defence. The mitigations are that every query lives in
`internal/store/queries/*.sql` where it can be reviewed as a set, and that
[07-testing.md](07-testing.md) requires a cross-user test for every owned resource.

## Token flow

```
  App                     Supabase Auth              Go API              Supabase JWKS
   │                           │                       │                       │
   │──signUp / signIn─────────▶│                       │                       │
   │◀─access_token + refresh───│                       │                       │
   │  (stored by supabase-js)  │                       │                       │
   │                           │                       │                       │
   │──GET /api/v1/diary───────────────────────────────▶│                       │
   │   Authorization: Bearer <access_token>            │──fetch keys (cached)─▶│
   │                                                   │◀──public keys─────────│
   │                                                   │                       │
   │◀──200 + day view──────────────────────────────────│  verify sig, exp,     │
   │                                                   │  iss, aud → sub       │
   │                           │                       │                       │
   │──refresh (automatic,──────▶                       │                       │
   │  before expiry)           │                       │                       │
```

The Go service never talks to Supabase Auth's API and holds no Supabase credentials beyond the
public key material. It cannot mint a token, revoke one, or look a user up. It can only verify.

## Client side

`@supabase/supabase-js`, constructed once in `src/auth/supabase.ts`.

### Session persistence

The client library persists the session and refreshes it automatically. Storage differs by
platform, and this is the one place the app branches on `Platform.OS`:

| Platform | Storage | Notes |
|---|---|---|
| Web | `localStorage` | The library's default; no configuration |
| iOS / Android | `AsyncStorage` | Passed explicitly as the `storage` option |

```ts
createClient(url, anonKey, {
  auth: {
    storage: Platform.OS === 'web' ? undefined : AsyncStorage,
    autoRefreshToken: true,
    persistSession: true,
    detectSessionInUrl: false,
  },
})
```

`detectSessionInUrl: false` because slice 1 has no OAuth or magic-link redirect to parse, and
leaving it on means the library inspects every URL the web app loads.

`localStorage` is readable by any JavaScript on the origin, which is the standard XSS exposure
for a browser SPA. Slice 1 accepts it: the alternative — an httpOnly refresh cookie — needs a
server-side session exchange that Supabase's client library is not built around, and would mean
writing the auth flow this project deliberately did not write. Nothing else on the origin loads
third-party script.

### The anon key is public

`EXPO_PUBLIC_SUPABASE_ANON_KEY` ships inside the client bundle. That is what it is for — it
identifies the project and permits calls to the auth endpoints. It is not a secret and does not
grant table access.

**The service-role key must never appear in the app,** in any environment variable, in any
config file under `app/`, or in this repository. It bypasses everything. The Go service does
not need it either — it uses the database connection string.

### `SessionProvider`

`src/auth/SessionProvider.tsx` holds the session in React context and keeps it current:

1. On mount, `supabase.auth.getSession()` reads the persisted session. Until it resolves, the
   provider reports a `loading` state — the route gate must render nothing rather than
   redirecting, or a returning user gets flashed to `/login` on every cold start.
2. `supabase.auth.onAuthStateChange` subscribes for the rest of the app's life, so a refresh, a
   sign-out, or a sign-out in another tab updates the context.
3. The subscription is unsubscribed on unmount.

The gate lives in `app/(app)/_layout.tsx`: no session and not loading, redirect to `/login`.
Details in [05-app-ui.md](05-app-ui.md).

### Attaching the token

`src/api/client.ts` reads the current session immediately before each request and sets the
header. It does **not** cache the token in a module variable — `supabase-js` may have rotated it
since the last call, and a stale token produces a `401` that looks like a server bug.

On a `401`, the wrapper calls `supabase.auth.refreshSession()` once and retries the request
once. If the retry also `401`s, it signs out and lets the route gate redirect. One retry, not a
loop: a genuinely revoked session would otherwise spin forever.

## Server side

`internal/auth/verifier.go`. The middleware runs on every `/api/v1` route and on none of
`/healthz`.

### Verification steps

1. Read the `Authorization` header. It must be exactly `Bearer <token>` — one space, case
   insensitive on the scheme. Missing or malformed is `401` with no further work.
2. Parse the JWT header to get the `kid` and the algorithm.
3. Look up the key by `kid` in the cached key set.
4. Verify the signature.
5. Check the registered claims:
   - `exp` in the future, with a small leeway (30s) for clock skew
   - `iss` exactly `https://<project-ref>.supabase.co/auth/v1`
   - `aud` contains `authenticated`
   - `sub` present and a valid uuid
6. Store the `sub` in the request context via `internal/auth/context.go`.

Any failure at any step is `401` with `{"error":{"code":"unauthenticated", ...}}`. The response
does not say which step failed. Handlers never see an unauthenticated request.

The algorithm must be checked against an allowlist, not read from the token and trusted. A
token whose header says `"alg": "none"`, or that swaps an asymmetric algorithm for `HS256` to
get the public key treated as an HMAC secret, is the classic JWT forgery, and libraries that
accept the header's word for it are how it lands. `github.com/golang-jwt/jwt/v5` requires the
caller to supply a keyfunc and to assert the method type inside it — do that assertion.

### `internal/auth/context.go`

```go
type ctxKey struct{}

func withUserID(ctx context.Context, id uuid.UUID) context.Context
func UserIDFrom(ctx context.Context) (uuid.UUID, bool)
```

The key is an unexported zero-size struct type, which is the Go idiom: no other package can
construct that key, so no other package can overwrite the user id in a context. A string key
could collide with any other library's.

`UserIDFrom` returning `false` inside a handler is a programming error — it means a route was
registered outside the authenticated group. Handlers treat it as `500`, not `401`, because
answering `401` would hide the misconfiguration.

### Which signing mode? — confirm this in T-01

Supabase projects sign JWTs in one of two ways, and **which one applies depends on the
project's JWT signing key configuration**:

| Mode | Verification | Key source |
|---|---|---|
| **Asymmetric** (current default for new projects) | Public key, `ES256` or `RS256` | JWKS endpoint at `https://<ref>.supabase.co/auth/v1/.well-known/jwks.json` |
| **Legacy shared secret** | `HS256` symmetric | The project's JWT secret, from the dashboard |

[01-architecture.md](01-architecture.md) assumes the asymmetric path, and it is the one to use:
the Go service then holds no secret at all, only public key material, and a leaked config file
lets nobody mint tokens. Supabase also supports rotating signing keys, which JWKS handles by
`kid` and a shared secret does not.

**T-01 must check the project's setting and record the answer** before T-08 is written. If the
project is on the legacy shared secret, either migrate it to asymmetric keys in the dashboard —
preferred, and it is a one-time action — or implement the HS256 path as a fallback:
`SUPABASE_JWT_SECRET` as a Fly secret, `HS256` as the only allowed algorithm, no JWKS fetch,
and no key cache. Everything else in this document is unchanged.

Do not implement both paths and switch on config. Pick one; the dead branch is a place for a
verification bug to hide.

### JWKS caching

Fetching the key set on every request would add a network round trip to every API call and make
Supabase's uptime our uptime. The verifier caches it:

- Fetch once at startup. **A failure here does not stop the service from booting** — it retries
  in the background and returns `401` until it succeeds. Refusing to start would mean a
  Supabase blip during a deploy leaves nothing running at all.
- Serve from cache for a TTL of 10 minutes.
- **On an unknown `kid`, refetch immediately** (rate-limited to at most once a minute) and try
  again before rejecting. This is what makes key rotation invisible: a token signed with a new
  key arrives, the cache misses, the refetch picks up the new key, the request succeeds.
- Guard the cache with a `sync.RWMutex`. Reads are frequent and writes are rare, which is
  exactly what `RWMutex` is for — and getting this right is a good first encounter with Go's
  concurrency model.

`github.com/MicahParks/keyfunc/v3` implements this pattern against `golang-jwt` and is the
recommended dependency; writing it by hand is also a reasonable learning exercise, and the
interface it must satisfy is one function.

## Authorization rules

Every rule the service enforces, in one list. Each is a `where` clause, not a policy file.

| Resource | Rule |
|---|---|
| `profiles` | Read and write only the row where `user_id` = context user. Upserted on first touch. |
| `foods` — read | `owner_id is null or owner_id = <context user>` |
| `foods` — create | `owner_id` is always set to the context user; never read from the body |
| `foods` — update/delete | Not in slice 1. No endpoint exists. |
| `diary_entries` — read | `user_id = <context user>`, always |
| `diary_entries` — create | `user_id` stamped from context; `food_id` must pass the food read rule |
| `diary_entries` — update/delete | `where id = $1 and user_id = $2` — **both**, in the same statement |

That last row matters. Loading the entry, checking ownership in Go, then updating by id is two
statements with a gap between them. One statement that filters on both columns has no gap, and
"zero rows affected" is the not-found signal.

A resource that exists but fails an ownership rule is `404`, never `403`. The reasoning is in
[03-api-contract.md](03-api-contract.md).

## Not in slice 1

Deferred, per [00-overview.md](00-overview.md), each with a note on what it costs later:

- **Password reset** — Supabase implements it; the work is configuring the redirect URL per
  platform and adding a screen to handle the recovery token. Slice 2.
- **Email confirmation** — off in T-01 so signup is one step. Turning it on later changes the
  signup screen's success state, nothing else.
- **OAuth providers** — needs `detectSessionInUrl` and deep-link configuration.
- **Rate limiting on the Go service** — Supabase rate-limits its own auth endpoints, which
  covers the credential-stuffing surface. The data endpoints require a valid token to reach, so
  the exposure is one authenticated user hammering their own data.
- **Token revocation checks** — an access token stays valid until `exp` even if the user signs
  out elsewhere. With a short access-token lifetime (Supabase's default is one hour) this is
  the normal, accepted property of stateless JWTs. Checking revocation would need a call to
  Supabase on every request, which is the round trip the JWKS cache exists to avoid.

## Related documents

- [01-architecture.md](01-architecture.md) — where the middleware sits in the chain
- [02-data-model.md](02-data-model.md) — why there is no FK to `auth.users`
- [03-api-contract.md](03-api-contract.md) — the `401` envelope and the `404`-not-`403` rule
- [06-deployment.md](06-deployment.md) — `SUPABASE_JWKS_URL` and the rest of the environment
- [07-testing.md](07-testing.md) — verifying the middleware without a network
