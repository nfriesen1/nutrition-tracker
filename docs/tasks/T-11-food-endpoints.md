# T-11 — Food endpoints

**Goal.** Foods can be searched, fetched by id, and created — with the seed/own visibility rule
enforced on every path.

**Depends on.** [T-10](T-10-profile-endpoints.md).

## Steps

1. `internal/store/queries/foods.sql`:
   - `SearchFoods` — **`where lower(name) like lower($1) || '%'`**, exactly this shape or the
     index in [02-data-model.md](../02-data-model.md) is not used. Filtered by
     `(owner_id is null or owner_id = $2)`. Ordered custom-first then alphabetical, limited.
   - `GetFoodForUser` — by id, with the same visibility filter in the `where` clause.
   - `CreateFood` — `owner_id` always the context user.
2. `internal/httpx/foods.go` — the three handlers.
3. `GET /api/v1/foods`: clamp `limit` to `1..50` rather than rejecting it; **return
   `{"foods":[]}` without a database call when `q` is under two characters.**
4. `GET /api/v1/foods/{id}`: an id that is not a uuid, does not exist, or belongs to another
   user all return the **same `404` body**. Never `403`.
5. `POST /api/v1/foods`: validate per the contract's rule table, round nutrient values to two
   decimals on write, echo what was stored.
6. Map the unique-index violation to `409` by **inspecting the pgx error's constraint name**,
   not by a pre-flight `select` — the pre-flight has a race, the constraint does not.
7. Compute `is_custom` as `owner_id is not null` in the response mapping. **Never return
   `owner_id`.**
8. Handler tests: one per status branch, plus the cross-user `404` test from
   [07-testing.md](../07-testing.md).

## Done when

- [ ] `GET /api/v1/foods?q=b` returns `[]` and produces no SQL log line.
- [ ] With a seed row present, `?q=ban` finds it and `?q=nana` does not — prefix, not substring.
- [ ] `explain` on the search query shows an index scan on `foods_name_prefix_idx`, not a
      sequential scan.
- [ ] A custom food created by user A sorts above seed foods **for A** and is invisible to B.
- [ ] `GET /api/v1/foods/<A's food id>` as user B returns `404` with a body identical to a
      random uuid's `404`.
- [ ] Creating the same name+brand twice returns `409`.
- [ ] `owner_id` appears in no response body.
- [ ] Sending `10.239` as calories stores and returns `10.24`.

## Notes

The `explain` check is worth actually running once. It is the only way to know the functional
index is being used, and `ilike` against a plain column — the natural thing to write — silently
does not use it.

There is no macro/calorie consistency check. Real foods do not satisfy `4p + 4c + 9f = kcal`.

## Spec references

- [03-api-contract.md](../03-api-contract.md) — all three endpoints and the `404`-not-`403` rule
- [02-data-model.md](../02-data-model.md) — the index and the exact query shape it needs
- [04-auth.md](../04-auth.md) — the foods authorization rules
