# T-06 — Schema migrations

**Goal.** The three tables from [02-data-model.md](../02-data-model.md) exist in Supabase,
created by goose migrations that run automatically at service boot.

**Depends on.** [T-01](T-01-supabase-project.md), [T-05](T-05-go-service-scaffold.md).

## Steps

1. Add goose as a library dependency (`github.com/pressly/goose/v3`), not just a CLI.
2. Write `db/migrations/`, transcribing [02-data-model.md](../02-data-model.md) exactly:
   - `0001_create_profiles.sql`
   - `0002_create_foods.sql` — table plus **both** indexes
   - `0003_create_diary_entries.sql` — table plus `(user_id, entry_date)`
   Every file gets a working `-- +goose Down`.
3. Embed the directory with `//go:embed migrations/*.sql` and apply it from `cmd/api/main.go`
   **before** the server starts listening.
4. Add `DATABASE_URL` to the required set in `internal/config`. Use the **pooler** connection
   string, port 6543.
5. Confirm every `CHECK` constraint from the spec is present — the meal values, the positive
   quantity, the non-negative nutrients, the goal bounds, the length limits.

## Done when

- [ ] `go run ./cmd/api` on an empty database creates all three tables and logs each migration.
- [ ] Running it a second time applies nothing and logs that it is already current.
- [ ] `\d foods` in `psql` shows both indexes, including
      `foods_name_prefix_idx` on `lower(name)`.
- [ ] `insert into diary_entries (..., meal) values (..., 'brunch')` is rejected by the CHECK.
- [ ] `insert into diary_entries (..., quantity) values (..., 0)` is rejected.
- [ ] Down migrations run cleanly in reverse and leave no orphaned indexes.
- [ ] **A migration deliberately broken with a syntax error stops the service from starting**
      rather than leaving it up with a half-applied schema.

## Notes

Write and test the down migrations even though production will never run them. Writing the down
is how you notice you forgot to drop an index.

`gen_random_uuid()` needs no extension on Supabase's Postgres version.

The last checkbox is testing the deploy-safety property in
[06-deployment.md](../06-deployment.md): a bad migration must fail the deploy, not corrupt the
schema.

## Spec references

- [02-data-model.md](../02-data-model.md) — the authoritative DDL and the migration order table
- [06-deployment.md](../06-deployment.md) — boot-time migration and the risk it accepts
