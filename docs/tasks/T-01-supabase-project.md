# T-01 — Supabase project setup

**Goal.** A Supabase project exists with email/password auth enabled, and every credential the
rest of the build needs is recorded somewhere safe.

**Depends on.** Nothing. This is the first task.

## Steps

1. Create a Supabase project. **Note the region** — [T-16](T-16-first-deploy.md) puts the Fly
   app in the matching one.
2. Under Authentication → Providers, enable **Email**. Disable "Confirm email" so signup returns
   a live session in one step ([04-auth.md](../04-auth.md) defers confirmation to slice 2).
3. Note the password minimum length. [T-04](T-04-auth-screens.md)'s zod schema must match it.
4. **Determine the JWT signing mode.** Under Authentication → JWT Keys (or JWT Settings), check
   whether the project signs with asymmetric keys or a legacy shared secret. If it offers a
   migration to asymmetric keys, take it. Record the answer — [T-08](T-08-jwt-middleware.md)
   is written against it and the two paths are different code.
5. Collect and store, in a password manager and not in this repository:
   - Project URL — `https://<ref>.supabase.co`
   - Anon / publishable key
   - Database connection string, **pooler variant, port 6543**
   - JWKS URL, or the JWT secret if the project is on the legacy mode
6. Confirm the JWKS endpoint is reachable and returns keys:
   `curl https://<ref>.supabase.co/auth/v1/.well-known/jwks.json`
7. Copy the templates and fill them in:
   ```sh
   cp app/.env.example app/.env.local
   cp api/.env.example api/.env.local
   ```
   Both are gitignored; the `.example` files are committed and must stay empty of real
   values. `api/.env.local` is not read by anything until [T-05](T-05-go-service-scaffold.md),
   but filling it in now means the credentials are recorded in one pass.
8. Write the answers into **Record** below — [T-08](T-08-jwt-middleware.md) reads them.
9. Verify: `./scripts/verify-t01.sh`

## Record

Fill this in as part of the task. These answers are inputs to later tasks, and an
undocumented decision is one that gets re-litigated in three weeks.

| | |
|---|---|
| Project ref | in `app/.env.local` / `api/.env.local` — not committed |
| Region | **TODO** — *(matched by `primary_region` in [T-16](T-16-first-deploy.md)'s `fly.toml`)* |
| **JWT signing mode** | **asymmetric (JWKS)** — verified 2026-08-12 |
| Signing algorithm | `ES256`, one key in the set |
| Password minimum length | **TODO** — *(the zod schema in [T-04](T-04-auth-screens.md) must match)* |
| Email provider | enabled — verified via `/auth/v1/settings` |
| Email confirmation | disabled (`mailer_autoconfirm` on) — verified |
| Date completed | 2026-08-12 (app side); API side outstanding, see below |

The signing mode is the load-bearing answer here. It came back **asymmetric**, which is the
path [01-architecture.md](../01-architecture.md) assumes and the one
[04-auth.md](../04-auth.md) prefers: the Go service then holds no secret at all, only public key
material. [T-08](T-08-jwt-middleware.md) implements the JWKS path and **not** the HS256
fallback — `SUPABASE_JWT_SECRET` stays unset.

### Outstanding

`api/.env.local` exists with the JWKS URL, issuer, and audience derived from the project URL,
but **`DATABASE_URL` is still blank** — it needs the database password, which is only visible in
the dashboard. This does not block [T-03](T-03-supabase-client-session.md) or
[T-04](T-04-auth-screens.md), which talk to Supabase directly and never to the Go service. It
does block [T-06](T-06-schema-migrations.md) onward.

Get it from Project Settings → Database → Connection string → **Transaction pooler** (port
6543, not 5432), and paste it into `api/.env.local`.

## Done when

- [x] Signing up through the Supabase dashboard's auth UI creates a user with no email
      confirmation step. *(email provider enabled, `mailer_autoconfirm` on)*
- [x] The JWKS URL returns a key set, **or** the project is confirmed as legacy-shared-secret and
      that fact is written down. *(one `ES256` key — asymmetric)*
- [ ] `psql "<pooler connection string>" -c 'select 1'` succeeds. **← `DATABASE_URL` not set yet**
- [x] `app/.env.local` and `api/.env.local` exist and `git status` lists neither.
- [ ] **Record** above is filled in. **← region and password minimum still TODO**
- [ ] `./scripts/verify-t01.sh` passes every check. **← 7 pass, 5 fail, all on the API side**

## Notes

The service-role key is not needed by anything in slice 1. Do not copy it anywhere.

Free-tier projects pause after about a week with no traffic. A paused project looks exactly like
an outage; waking it is one click in the dashboard.

## Spec references

- [04-auth.md](../04-auth.md) — the signing-mode decision and why asymmetric is preferred
- [06-deployment.md](../06-deployment.md) — the full environment variable table
- [02-data-model.md](../02-data-model.md) — why nothing is added to the `auth` schema
