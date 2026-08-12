# T-14 — Seed foods

**Goal.** The `foods` table ships with a curated list of common foods, so search returns
something on the first day.

**Depends on.** [T-06](T-06-schema-migrations.md).

## Steps

1. Assemble roughly 150–250 common foods. Aim for coverage of what someone actually logs rather
   than breadth:
   - Fruit and vegetables — banana, apple, avocado, broccoli, potato
   - Proteins — chicken breast, egg, salmon, mince, tofu, tinned tuna
   - Grains and starches — bread, rice, pasta, oats, tortilla
   - Dairy — milk (whole and semi), cheddar, greek yoghurt, butter
   - Nuts, seeds, oils
   - Store-cupboard staples — peanut butter, honey, olive oil, pasta sauce
   - A handful of common snacks
2. For each: `name`, `serving_label`, `calories`, `protein_g`, `carbs_g`, `fat_g`. `brand` and
   `owner_id` both `null`.
3. **`serving_label` must describe a real, recognisable portion** with the gram weight in
   parentheses: `medium (118 g)`, `1 slice (28 g)`, `1 cup (240 ml)`. This is the field users
   reason about, and a vague one makes every log of that food wrong.
4. Source the numbers from USDA FoodData Central and round to two decimals. Note the source in a
   comment at the top of the migration.
5. Write `db/migrations/0004_seed_foods.sql` as a single `insert`. The down migration deletes
   `where owner_id is null`.
6. Make it re-runnable: `on conflict do nothing`, or a guard so re-applying is harmless.
7. Name things the way a person searches. **Prefix matching means the first word is what
   matters** — "Banana, raw" is findable by typing "ban"; "Raw banana" is not.

## Done when

- [ ] The migration applies cleanly on an empty database and inserts the full list.
- [ ] Every row has `owner_id is null`.
- [ ] `GET /api/v1/foods?q=ban` returns the banana; `?q=chic` returns chicken breast;
      `?q=egg` returns the egg.
- [ ] Every `serving_label` includes a weight or volume.
- [ ] Spot-check five rows against the label or FoodData Central.
- [ ] Re-running the migration on a seeded database does not duplicate rows.
- [ ] The down migration removes exactly the seed rows and leaves custom foods alone.

## Notes

Point 7 is the one that decides whether search feels usable. The index is a prefix index by
design — [02-data-model.md](../02-data-model.md) explains the trade — so "Greek yoghurt, plain"
is found by "gre" or "gree", never by "yog". Where a food is genuinely known by its second word,
add the obvious alternative as a second row rather than fighting the index.

Third-party food databases are explicitly out of scope for slice 1 — see
[00-overview.md](../00-overview.md). This list is what stands in for one.

## Spec references

- [00-overview.md](../00-overview.md) — the curated-seed-list decision
- [02-data-model.md](../02-data-model.md) — `owner_id is null`, the prefix index, per-serving
  nutrition
- [03-api-contract.md](../03-api-contract.md) — how these rows surface in search
