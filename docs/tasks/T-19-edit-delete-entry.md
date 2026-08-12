# T-19 — Edit quantity and delete entry

**Goal.** Tapping a logged entry opens a sheet where its quantity can be changed or the entry
removed.

**Depends on.** [T-15](T-15-diary-screen.md).

## Steps

1. A bottom sheet over the diary, opened from `EntryRow`. **Not a route** — it is a one-field
   edit, and on web a route would put an entry id in the address bar for something not worth
   linking to.
2. Contents: the food name and serving label as context, a `QuantityInput` seeded with the
   current quantity, Save, and Delete.
3. Save → `PATCH /api/v1/diary/entries/{id}` → invalidate `['diary', date]` → close.
4. Delete → confirm → `DELETE /api/v1/diary/entries/{id}` → invalidate → close. Confirm on web
   as well as native; **there is no undo.**
5. Validation matches the server: `0 < quantity ≤ 1000`. Save is disabled while the field is
   empty or invalid.
6. Errors render inside the sheet without closing it, so a typed value is not lost.

## Done when

- [ ] Changing `1` to `2` doubles the entry's calories and updates both the meal subtotal and
      the day total.
- [ ] Deleting an entry removes it and the subtotals drop accordingly.
- [ ] Delete asks for confirmation, and cancelling leaves the entry alone.
- [ ] A quantity of `0` or a negative is refused before a request is sent.
- [ ] Dismissing the sheet without saving changes nothing.
- [ ] Deleting an entry already deleted in another tab shows the `404` gracefully and the diary
      refetches.
- [ ] The sheet is usable on a phone — reachable buttons, no keyboard covering the input.

## Notes

Editing the quantity **re-snapshots from the food's current values**, so if the food was
corrected since logging, the numbers can move by more than the quantity ratio.
[03-api-contract.md](../03-api-contract.md) explains why that is the right trade. Do not
pre-emptively add UI explaining it — add a note only if it confuses someone in practice.

Moving an entry between meals or days is not supported. Delete and re-add.

## Spec references

- [05-app-ui.md](../05-app-ui.md) — the sheet and why it is not a route
- [03-api-contract.md](../03-api-contract.md) — `PATCH` and `DELETE`, and the re-snapshot rule
