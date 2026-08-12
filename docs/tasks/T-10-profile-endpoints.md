# T-10 — Profile endpoints

**Goal.** `GET /api/v1/me` and `PATCH /api/v1/me` work against a real token, creating the
profile row on first touch.

**Depends on.** [T-07](T-07-sqlc-store.md), [T-08](T-08-jwt-middleware.md).

## Steps

1. `internal/store/queries/profiles.sql` — `UpsertProfile` using
   `insert ... on conflict (user_id) do nothing` followed by the select, inside the transaction
   helper. Two concurrent first-requests must both succeed.
2. `UpdateProfileGoal` — `update ... set daily_calorie_goal = $2, updated_at = now()
   where user_id = $1 returning *`, also upserting so a user who never called `GET` can still
   save a goal.
3. `internal/httpx/me.go` — both handlers, taking `user_id` from `auth.UserIDFrom(ctx)` and
   never from the request.
4. Validate `daily_calorie_goal`: required, integer, `1..20000`. `422` with a `details` map
   naming the field.
5. Register both routes in `router.go`. Delete the temporary `/whoami` from
   [T-08](T-08-jwt-middleware.md).
6. Handler tests: `200` happy path for each, `401` without a token, `422` for a missing goal, a
   zero, a negative, an over-max, and a non-integer.

## Done when

- [ ] `curl -H "Authorization: Bearer <real token>" localhost:8080/api/v1/me` returns the
      user id and `2000`, **and creates the row** — confirm with `select * from profiles`.
- [ ] Calling it a second time returns the same row and does not error.
- [ ] `PATCH` with `{"daily_calorie_goal": 2200}` returns `2200` and persists.
- [ ] `PATCH` from a user who has never called `GET` still works.
- [ ] Each `422` case returns the field name in `details`.
- [ ] No token returns `401`.

## Notes

This is the first endpoint where a real token travels the whole path — app → Fly → JWKS →
Postgres. Debugging that path here, with a two-column table and no business logic, is far
easier than debugging it inside the diary handler. That is why `/me` comes before
[T-11](T-11-food-endpoints.md) and [T-12](T-12-diary-endpoints.md).

Get a real token by logging into the app from [T-04](T-04-auth-screens.md) and reading
`access_token` out of `localStorage`.

## Spec references

- [03-api-contract.md](../03-api-contract.md) — both endpoints, validation rules, status codes
- [02-data-model.md](../02-data-model.md) — the `profiles` table and lazy creation
- [04-auth.md](../04-auth.md) — the profile authorization rule
