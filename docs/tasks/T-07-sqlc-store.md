# T-07 — sqlc + pgx store wiring

**Goal.** The service holds a pgx connection pool, and hand-written SQL is compiled into typed
Go methods by sqlc.

**Depends on.** [T-06](T-06-schema-migrations.md).

## Steps

1. Install the sqlc CLI and write `api/sqlc.yaml`: engine `postgresql`, driver
   `pgx/v5`, schema pointed at `db/migrations`, queries at `internal/store/queries`, output at
   `internal/store/generated`.
2. `internal/store/store.go` — build the pool with `pgxpool.New`, **`MaxConns` set to 5–10**,
   sensible `MaxConnLifetime` and `MaxConnIdleTime`, and a `Ping` at startup that fails loudly.
3. Write the first queries in `internal/store/queries/profiles.sql` so there is something to
   generate from:
   ```sql
   -- name: UpsertProfile :one
   -- name: UpdateProfileGoal :one
   ```
4. Run `sqlc generate`. **Commit the output.** It is machine-written and never hand-edited —
   [01-architecture.md](../01-architecture.md) says so — but committing it means the build does
   not require the sqlc CLI.
5. Add a transaction helper to `store.go` that takes a `func(tx) error`, commits on nil, and
   rolls back via `defer` otherwise. `UpsertProfile` in
   [T-10](T-10-profile-endpoints.md) is its first user.
6. Define a store **interface** in `internal/httpx` listing only the methods the handlers use.
   The generated `Queries` struct satisfies it structurally, with no declaration — and the stub
   in [07-testing.md](../07-testing.md)'s handler tests satisfies it the same way.
7. Wire the pool through `cmd/api/main.go` and close it on shutdown.
8. Decide how `numeric` columns arrive in Go — `pgtype.Numeric` or `shopspring/decimal` via
   sqlc's type overrides — and use it consistently. **Do not let a `float64` into the nutrition
   path**; [07-testing.md](../07-testing.md) has a test that fails if one does.

## Done when

- [ ] `sqlc generate` produces compiling Go with no manual edits.
- [ ] The service starts, pings the database, and logs the pool size.
- [ ] A deliberate typo in a `.sql` file makes `sqlc generate` fail — this is the compile-time
      schema check.
- [ ] The handler-facing store interface is declared in `httpx`, not in `store`, so the consumer
      owns it.
- [ ] Nutrition columns surface as a decimal type, not `float64`.

## Notes

Declaring the interface in the consuming package is the Go idiom and it is what makes
[07-testing.md](../07-testing.md)'s stub store work with no mocking framework. Coming from C#,
the instinct is to declare the interface next to the implementation; resist it.

If `prepared statement already exists` appears, it is the pooler in transaction mode — see the
connection-string fix in [06-deployment.md](../06-deployment.md).

## Spec references

- [01-architecture.md](../01-architecture.md) — the `store/` layout and the never-edit rule
- [02-data-model.md](../02-data-model.md) — the column types being mapped
- [06-deployment.md](../06-deployment.md) — pooler port and `MaxConns`
- [07-testing.md](../07-testing.md) — why the interface shape matters
