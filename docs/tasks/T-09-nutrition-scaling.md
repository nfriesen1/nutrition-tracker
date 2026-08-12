# T-09 — Nutrition scaling

**Goal.** A pure function turns a food's per-serving nutrients and a quantity into the snapshot
values a diary entry stores, and it is tested exhaustively.

**Depends on.** [T-07](T-07-sqlc-store.md), for the decimal type decision.

## Steps

1. `internal/nutrition/scale.go`. **No I/O, no database, no context parameter.** Types plus one
   function:
   ```go
   type Nutrients struct {
       Calories, ProteinG, CarbsG, FatG decimal.Decimal
   }

   func Scale(per Nutrients, quantity decimal.Decimal) Nutrients
   ```
2. Multiply each field by `quantity`, round to **two decimals, half away from zero**. Rounding
   happens once, here, at write time.
3. Add `RoundQuantity` for the same two-decimal treatment of the incoming quantity, so the
   stored `quantity` and the stored nutrients always agree.
4. Use decimal arithmetic end to end. **A `float64` anywhere in this file is a bug.**
5. Write the table test from [07-testing.md](../07-testing.md), including every row of its case
   table — most importantly the worked example from
   [02-data-model.md](../02-data-model.md), the exact-half-cent rounding case, and the
   non-terminating-binary-fraction case.

## Done when

- [ ] `go test ./internal/nutrition/...` passes.
- [ ] **The worked example produces exactly `157.50 / 1.94 / 40.43 / 0.59`.** If it does not,
      either the code or [02-data-model.md](../02-data-model.md) is wrong — resolve which before
      moving on.
- [ ] `quantity = 1` returns the input unchanged.
- [ ] `x.xx5` rounds away from zero, not to even.
- [ ] Quantity `1000` stays inside `numeric(7,2)`'s range.
- [ ] Quantity `0.01` on a small food gives `0.00` and does not error.
- [ ] The package imports nothing from `store`, `httpx`, or `database/sql`.

## Notes

This is the only real business logic in slice 1, and doing it before the handlers means the
arithmetic is settled before it is tangled up with HTTP. Writing the table test is also the
single best introduction to Go's testing idiom — subtests, `t.Run`, no assertion library.

The client computes a preview of the same arithmetic in
[T-17](T-17-add-food-screen.md). That preview is cosmetic; **this** is authoritative, and the
client always displays what the API returns.

## Spec references

- [02-data-model.md](../02-data-model.md) — the worked example and the rounding rule
- [01-architecture.md](../01-architecture.md) — why this package is pure
- [07-testing.md](../07-testing.md) — the case table in full
