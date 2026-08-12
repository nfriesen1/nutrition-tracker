# T-21 — Test pass and production acceptance run

**Goal.** The slice is verifiably finished: the suite is green, and
[00-overview.md](../00-overview.md)'s success criterion holds in production.

**Depends on.** Every preceding task.

## Steps

1. Run the full suite and fix what is broken:
   ```sh
   cd api && go test ./... && go test -tags=integration ./... && go test -race ./...
   cd api && go vet ./... && gofmt -l .
   cd app && npm test && npm run typecheck
   ```
2. Confirm the tests [07-testing.md](../07-testing.md) requires actually exist — it is easy for
   one to have been skipped under time pressure. Specifically:
   - the `alg: none` and algorithm-confusion tests in `internal/auth`
   - a cross-user `404` test for **each** owned resource
   - the `user_id`-in-the-body-is-ignored test
   - the timezone test and the snapshot-immutability test in the integration suite
   - the `date.ts` DST case
3. Deploy the current `main` — **API first, then the app.**
4. Run the eleven-step production acceptance run in
   [07-testing.md](../07-testing.md), against the deployed URL, in a clean browser profile.
5. Do step 10 properly: **a second device, or at minimum a private window on another machine.**
   It is the only step that proves the data is not sitting in local storage, and it is the
   half of the success criterion that is easiest to skip.
6. Check `fly logs` afterwards: no panics, no `500`s, request durations in the tens of
   milliseconds.
7. Walk [00-overview.md](../00-overview.md)'s in-scope list and confirm each bullet is real.
   Anything missing is a task, not a footnote.
8. Update `README.md`'s status section — the slice has shipped and the "specification phase"
   wording no longer applies.

## Done when

- [ ] Every command in step 1 exits zero.
- [ ] Every test in step 2 exists and passes.
- [ ] All eleven acceptance steps pass against production.
- [ ] **A new user can sign up, log a banana to breakfast, and see the calorie total on their
      diary — and it is still there after a hard refresh and on a different device.**
- [ ] `fly logs` is clean.
- [ ] Every in-scope bullet in [00-overview.md](../00-overview.md) works.
- [ ] `README.md` describes the app as shipped.

## Notes

That fourth checkbox is the success criterion, verbatim. It is the one that decides whether
slice 1 is done.

If something fails here, it is a bug to fix, not a caveat to record. The point of a vertical
slice is that it works end to end; a slice with an asterisk is not a slice.

Once this passes, the deferred list in [00-overview.md](../00-overview.md) becomes the input to
slice 2 — barcode scanning, recipes, macro goals, password reset, and a third-party food
database are all waiting there.

## Spec references

- [00-overview.md](../00-overview.md) — the success criterion and the in-scope list
- [07-testing.md](../07-testing.md) — the acceptance run and the full test inventory
- [06-deployment.md](../06-deployment.md) — the deploy sequence
