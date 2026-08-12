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
7. Create `app/.env.local` with the three `EXPO_PUBLIC_*` values from
   [06-deployment.md](../06-deployment.md). Confirm `.gitignore` already covers it.

## Done when

- [ ] Signing up through the Supabase dashboard's auth UI creates a user with no email
      confirmation step.
- [ ] The JWKS URL returns a key set, **or** the project is confirmed as legacy-shared-secret and
      that fact is written down.
- [ ] `psql "<pooler connection string>" -c 'select 1'` succeeds.
- [ ] `app/.env.local` exists and `git status` does not list it.

## Notes

The service-role key is not needed by anything in slice 1. Do not copy it anywhere.

Free-tier projects pause after about a week with no traffic. A paused project looks exactly like
an outage; waking it is one click in the dashboard.

## Spec references

- [04-auth.md](../04-auth.md) — the signing-mode decision and why asymmetric is preferred
- [06-deployment.md](../06-deployment.md) — the full environment variable table
- [02-data-model.md](../02-data-model.md) — why nothing is added to the `auth` schema
