# T-17 — Add-food screen

**Goal.** A user can search for a food, pick a quantity, and log it to a meal.

**Depends on.** [T-15](T-15-diary-screen.md).

## Steps

1. `app/(app)/add-food.tsx`. Read `?date=` and `?meal=` from the params — the screen knows where
   the entry is going before a food is chosen.
2. Search input, **debounced 300 ms**, querying `['foods', q]`. **Do not fire below two
   characters**; the server also guards this, and both checks are wanted.
3. `FoodResultRow` — name, brand, serving label, calories per serving, and a "yours" badge when
   `is_custom`.
4. Selecting a food reveals `QuantityInput`, defaulted to `1`, with a live client-side preview
   of the scaled calories. **The preview is cosmetic** — the `201` response is authoritative and
   is what gets displayed after adding.
5. Add → `POST /api/v1/diary/entries` → invalidate `['diary', date]` → `router.back()`.
6. The states from [05-app-ui.md](../05-app-ui.md): a hint line when idle or under two
   characters; a slim progress line while searching with previous results still visible;
   a no-results state carrying a **Create "<query>" as a new food** button that navigates to
   `/create-food?name=<query>`.
7. `keyboardType="decimal-pad"` on native, `inputMode="decimal"` on web, for the quantity field.

## Done when

- [ ] Typing "ban" finds the banana within a moment of stopping typing.
- [ ] One character fires no request — check the network tab.
- [ ] Typing quickly produces one request, not one per keystroke.
- [ ] Adding a banana at `1.5` returns to the diary with the entry under the right meal and a
      total that includes `158`.
- [ ] The entry lands on the **date the screen was opened with**, not today, when arriving from
      a past day.
- [ ] Searching for nonsense shows the create button, and it carries the query text through.
- [ ] The quantity field opens a numeric keyboard on a phone.
- [ ] Adding the same food twice creates two rows.

## Notes

The empty-state create button is the entire discovery path for custom foods. A user who searched
and found nothing is exactly the user who needs one; a separate menu item goes unused.

The date checkbox is the one worth being careful about — it is easy to write `todayLocal()` in
the mutation instead of threading the param through, and the bug only shows up when logging
retroactively.

## Spec references

- [05-app-ui.md](../05-app-ui.md) — the screen, its states, and the debounce
- [03-api-contract.md](../03-api-contract.md) — search and entry creation
- [02-data-model.md](../02-data-model.md) — what the preview is approximating
