# T-16 — First production deploy

**Goal.** The whole chain — Netlify → Fly → Supabase — is live and proven.

**Depends on.** [T-15](T-15-diary-screen.md).

## Steps

1. `api/Dockerfile` — the multi-stage build from
   [06-deployment.md](../06-deployment.md). `CGO_ENABLED=0`, distroless **static** base (it
   carries the CA certificates the JWKS fetch and the Supabase TLS connection both need),
   `-X main.version` from the git SHA.
2. Build and run it locally before touching Fly. A container that fails on Fly for a reason that
   was reproducible on the laptop is an hour wasted.
3. `api/fly.toml` — `primary_region` **matching the Supabase region**,
   `min_machines_running = 1`, `auto_stop_machines = false`, health check on `/healthz` at 15s.
4. `fly launch --no-deploy`, then `fly secrets set` for every Go variable in
   [06-deployment.md](../06-deployment.md)'s table. Secrets never go in `fly.toml`, which is
   committed.
5. `fly deploy`. Confirm the migration log lines appear in `fly logs` and
   `curl https://<app>.fly.dev/healthz` returns `200` with the real git SHA.
6. **Replace the placeholder hostname in `netlify.toml`'s redirect** with the real Fly hostname.
   The file at the repository root already carries the comment saying to do this here.
7. Connect the repository to Netlify. Set the three `EXPO_PUBLIC_*` variables and
   `NODE_VERSION=24` in the build environment. Deploy.
8. Verify the redirect ordering, which is the step that catches the most common
   misconfiguration.

## Done when

- [ ] `curl https://<app>.fly.dev/healthz` returns `200` and the version string is the deployed
      SHA, not `dev`.
- [ ] `fly logs` shows the goose migrations applied, and no panics.
- [ ] The Netlify site loads and the login screen renders.
- [ ] **`curl https://<site>.netlify.app/api/v1/me` returns `401` with the JSON error
      envelope** — not `404` (the proxy rule is not matching) and not HTML (the SPA catch-all is
      above the API rule).
- [ ] Signing up on the production URL works and lands on a real diary.
- [ ] The diary loads real seed-food data from the production database.
- [ ] The browser network tab shows API calls going to the **site's own origin**, not to
      `fly.dev` — same-origin proxying is working, and there are no CORS preflights.
- [ ] A second request after several minutes idle is fast — `min_machines_running` is doing its
      job.

## Notes

**This deploys before the app is feature-complete, deliberately.** Every checkbox above is a
configuration problem that takes an hour to find and a minute to fix, and they are far easier to
diagnose one at a time now than all at once after four more screens. From
[T-17](T-17-add-food-screen.md) onward, each task ships to this environment as it is finished.

The `401`-not-HTML check is the single most useful line in this file. Getting HTML back from an
API call surfaces in the app as `Unexpected token '<' in JSON at position 0` and reads like an
API bug when it is a redirect-ordering bug.

Deploy order from here on: **API first, then app.** Additive API changes are backwards
compatible; the reverse order means a client calling an endpoint that does not exist yet.

## Spec references

- [06-deployment.md](../06-deployment.md) — Dockerfile, `fly.toml`, secrets, the deploy sequence
- [03-api-contract.md](../03-api-contract.md) — `/healthz` and why it skips the database
- [07-testing.md](../07-testing.md) — the acceptance run this makes possible
