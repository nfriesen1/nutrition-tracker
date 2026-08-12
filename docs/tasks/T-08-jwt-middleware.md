# T-08 — JWT verification middleware

**Goal.** A Supabase access token is verified against the project's public keys, and the `sub`
claim reaches handlers through the request context.

**Depends on.** [T-01](T-01-supabase-project.md) — specifically its recorded answer about the
signing mode — and [T-05](T-05-go-service-scaffold.md).

## Steps

1. **Confirm the signing mode from [T-01](T-01-supabase-project.md) before writing anything.**
   Asymmetric (JWKS) is the expected path. If the project is on the legacy shared secret,
   implement the HS256 fallback in [04-auth.md](../04-auth.md) instead. **Implement one path
   only** — a dead branch is where a verification bug hides.
2. `internal/auth/context.go` — `withUserID` / `UserIDFrom`, keyed by an **unexported
   zero-size struct type** so no other package can overwrite it.
3. `internal/auth/verifier.go`:
   - JWKS fetch and cache, 10-minute TTL, guarded by a `sync.RWMutex`
   - refetch on unknown `kid`, rate-limited to once a minute
   - **startup fetch failure must not stop the service booting** — retry in the background and
     return `401` meanwhile
4. Verification, in order: parse header, find key by `kid`, verify signature, then check `exp`
   (30s leeway), `iss`, `aud`, and that `sub` is a uuid.
5. **Assert the algorithm inside the keyfunc against an allowlist.** Never trust the token
   header's `alg`. This is what stops `alg: none` and the RS256→HS256 confusion attack.
6. Add the middleware to the `/api/v1` group in `router.go` only. `/healthz` stays open.
7. Add a temporary `GET /api/v1/whoami` returning the context user id, for manual verification.
   Delete it in [T-10](T-10-profile-endpoints.md).
8. Write the verifier tests from [07-testing.md](../07-testing.md) — key pair generated in the
   test, JWKS served from an `httptest.Server`, no network.

## Done when

- [ ] A real token from the app returns the correct `sub` from `/api/v1/whoami`.
- [ ] No token, a malformed token, and an expired token each return `401` in the envelope shape,
      and the response body is identical for all three.
- [ ] **`alg: none` is rejected.**
- [ ] **A token re-signed as HS256 using the public key as the secret is rejected.**
- [ ] Wrong `iss` and wrong `aud` are each rejected.
- [ ] The JWKS endpoint is hit once, not once per request — confirm by counting hits on the test
      server.
- [ ] `go test -race ./internal/auth/...` is clean.
- [ ] Starting the service with the JWKS URL unreachable still boots, and requests get `401`.

## Notes

The two forgery tests are the reason this task has a test list at all. They cost ten minutes and
they are what keeps a later refactor honest.

`github.com/MicahParks/keyfunc/v3` implements the cache-and-refetch pattern against
`golang-jwt/jwt/v5`. Writing it by hand is a reasonable exercise — the interface is one
function — and the `RWMutex` is a good first encounter with Go concurrency.

## Spec references

- [04-auth.md](../04-auth.md) — the verification steps, caching rules, and both signing modes
- [03-api-contract.md](../03-api-contract.md) — the `401` envelope
- [07-testing.md](../07-testing.md) — the full verifier test table
