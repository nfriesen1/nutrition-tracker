# T-18 — Create-food screen

**Goal.** When search comes up empty, a user can create the food and log it without losing their
place.

**Depends on.** [T-17](T-17-add-food-screen.md).

## Steps

1. `app/(app)/create-food.tsx` — name, optional brand, serving label, calories, protein, carbs,
   fat.
2. Pre-fill `name` from `?name=` when arriving from the empty search state.
3. zod validation mirroring [03-api-contract.md](../03-api-contract.md). **Only name, serving
   label, and calories are required**; macros default to `0`. Demanding a full macro breakdown
   to log a homemade soup is how a user gives up.
4. `serving_label` gets a real placeholder — `1 cup (240 g)` — not a bare label. This is the
   field people get wrong, and getting it wrong makes every future log of that food wrong.
5. `POST /api/v1/foods` → invalidate `['foods']` → **return to `/add-food` with the new food
   already selected and the quantity input showing.** The user came here to log something.
6. `409` renders "You already have a food called that", attached to the name field.
7. Numeric inputs get `decimal-pad` / `inputMode="decimal"`.

## Done when

- [ ] Arriving from a failed search pre-fills the name.
- [ ] Creating a food returns to add-food with it selected, and one more tap logs it.
- [ ] The new food carries a "yours" badge and sorts above seed results.
- [ ] Submitting with only name, serving label, and calories succeeds; macros store as `0`.
- [ ] Creating the same name twice shows the `409` message on the name field.
- [ ] Entering `10.239` calories stores `10.24`, and the screen shows what was stored.
- [ ] Navigating here directly, with no `?name=`, works with an empty form.

## Notes

The "return with it selected" behaviour is what makes this feel like part of the logging flow
rather than a database admin screen. Dropping the user on the diary after creating a food means
they have to search for it again.

Editing and deleting custom foods is not in slice 1 — there is no endpoint for it. A mistyped
food is corrected by creating a better one; the duplicate constraint means the name has to
change.

## Spec references

- [05-app-ui.md](../05-app-ui.md) — the screen and its success behaviour
- [03-api-contract.md](../03-api-contract.md) — `POST /api/v1/foods`, validation, the `409`
- [02-data-model.md](../02-data-model.md) — per-serving nutrition and `serving_label`
