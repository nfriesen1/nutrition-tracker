# T-12 — Diary endpoints

**Goal.** The day view, and creating, editing, and deleting entries. This completes the API.

**Depends on.** [T-09](T-09-nutrition-scaling.md), [T-11](T-11-food-endpoints.md).

## Steps

1. `internal/store/queries/diary.sql`:
   - `ListDayEntries` — entries for `(user_id, entry_date)`, joined to `foods` **for
     `name`, `brand`, and `serving_label` only**, ordered by `created_at`.
   - `SumDayByMeal` — `sum(...) group by meal`. Computed in SQL, not in Go.
   - `CreateEntry`, `GetEntryForUser`, `UpdateEntryQuantity`, `DeleteEntry` — the last three
     filtering on **`where id = $1 and user_id = $2`, both columns, in one statement.**
2. `internal/httpx/diary.go`.
3. `GET /api/v1/diary`: `date` **required** — missing is `422`, never "today". Reject anything
   that is not a real `YYYY-MM-DD`, including `2026-02-30`. Assemble the response so `meals` is
   **always four objects in fixed order**, zero-filled for meals with no entries.
4. `remaining_calories` is `goal − total`, **not clamped at zero**.
5. `POST /api/v1/diary/entries`: validate, load the food under the visibility filter (`404` if
   absent or not visible), scale with `internal/nutrition`, insert with `user_id` from context.
6. `PATCH .../{id}`: quantity only. Reload the entry's **original `food_id`** and rescale from
   that food's **current** per-serving values — the deliberate re-snapshot in
   [03-api-contract.md](../03-api-contract.md).
7. `DELETE .../{id}`: `204`, `404` when zero rows were affected.
8. Handler tests: every status branch, the cross-user `404` for `PATCH` and `DELETE`, and the
   **`user_id`-in-the-body-is-ignored** test from [07-testing.md](../07-testing.md).
9. Write the integration test from [07-testing.md](../07-testing.md) behind the `integration`
   build tag — including the two rule tests: dates do not shift under an extreme `TZ`, and
   doubling a food's calories does not move an existing day's total.

## Done when

- [ ] A day with no entries returns four meal objects with zero subtotals and empty `entries`.
- [ ] Logging the worked example (banana, `1.5`, breakfast) returns `157.5` and the day total
      matches.
- [ ] Logging the same food twice to the same meal creates two rows.
- [ ] Omitting `date` is `422`. `2026-02-30` is `422`.
- [ ] **Under `TZ=Pacific/Kiritimati` and `TZ=Pacific/Midway`, an entry written for
      `2026-08-11` reads back as `2026-08-11`.**
- [ ] **Doubling a food's calories in SQL leaves an existing day's total unchanged**, and
      `PATCH`ing that entry's quantity then picks up the new values.
- [ ] User B gets `404` on `PATCH` and `DELETE` of user A's entry, and A's row is untouched.
- [ ] A body containing `"user_id": "<other uuid>"` creates the row for the authenticated user.
- [ ] `go test -tags=integration ./...` passes.

## Notes

The two rule checkboxes are the point of this task. They are the tests that hold up
[02-data-model.md](../02-data-model.md)'s two cross-cutting rules, and both are the kind of thing
a reasonable-looking refactor undoes silently.

**Milestone:** the API is now complete. Every endpoint in
[03-api-contract.md](../03-api-contract.md) exists.

## Spec references

- [03-api-contract.md](../03-api-contract.md) — all four endpoints and the day-view JSON in full
- [02-data-model.md](../02-data-model.md) — snapshotting and the date rule
- [07-testing.md](../07-testing.md) — the handler and integration test lists
