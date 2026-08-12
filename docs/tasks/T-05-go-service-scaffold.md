# T-05 — Go service scaffold

**Goal.** `go run ./cmd/api` starts an HTTP server that answers `/healthz`, logs every request,
survives a panic, and shuts down cleanly.

**Depends on.** Nothing. ([T-01](T-01-supabase-project.md) is only needed from
[T-06](T-06-schema-migrations.md) onward.)

## Steps

1. `go mod init` in `api/`. Module path can be `github.com/<user>/nutrition-tracker/api`.
2. `internal/config/config.go` — load from the environment, **fail fast listing every missing
   required variable at once**, not just the first. `PORT` defaults to `8080`; `ENV` defaults to
   `development`. `DATABASE_URL` is not required yet.
3. `internal/httpx/respond.go` — `writeJSON` and `writeError`, producing the single error
   envelope from [03-api-contract.md](../03-api-contract.md).
4. `internal/httpx/errors.go` — sentinel errors (`ErrNotFound`, `ErrValidation`, `ErrConflict`,
   `ErrUnauthenticated`) and the mapping from sentinel to status code. **Handlers return
   sentinels; only the responder writes a status.**
5. `internal/httpx/middleware.go`:
   - **recovery** — outermost. A panic becomes a `500` plus a logged stack trace, not a dead
     process.
   - **request logging** — assign a request id, log method, path, status, and duration on the
     way out. Needs a `ResponseWriter` wrapper to capture the status.
   - CORS, **development only**, driven off `ENV` — for the Expo web dev server on `:8081`
     calling `:8080`. Production web is same-origin and must not get a permissive header.
6. `internal/httpx/router.go` — the route table. Go 1.22 method-aware patterns
   (`mux.HandleFunc("GET /healthz", ...)`). One place, every endpoint.
7. `internal/httpx/health.go` — `GET /healthz` returning `{"status":"ok","version":"..."}` with
   **no database call**.
8. `cmd/api/main.go` — wire config → router → `http.Server` with read/write timeouts, and
   graceful shutdown on `SIGINT`/`SIGTERM` via `signal.NotifyContext` and `srv.Shutdown(ctx)`.
9. Bind `0.0.0.0`, not `127.0.0.1`, so a phone on the LAN can reach it in development.
10. Set `version` from `-ldflags="-X main.version=..."`, defaulting to `dev`.

## Done when

- [ ] `go run ./cmd/api` starts and `curl localhost:8080/healthz` returns `200` with the JSON
      body.
- [ ] A log line appears for that request with a request id, status, and duration.
- [ ] A temporary handler that panics returns `500` in the envelope shape and **the server is
      still up afterwards**.
- [ ] `Ctrl-C` logs a shutdown line and exits `0` rather than dying mid-request.
- [ ] Unsetting a required environment variable produces one clear startup error naming it.
- [ ] `go vet ./...` and `gofmt -l .` are clean.

## Notes

Writing the middleware chain by hand is the point — `func(http.Handler) http.Handler` is the
best available introduction to Go's interfaces, and [00-overview.md](../00-overview.md) picked
the standard library for exactly this reason.

Order matters: recovery outermost, then logging, then (from
[T-08](T-08-jwt-middleware.md)) auth. A panic inside the logger should still be caught, and an
auth rejection should still be logged.

**Milestone:** T-01 through T-05 done means both processes run locally.

## Spec references

- [01-architecture.md](../01-architecture.md) — the package layout and middleware order
- [03-api-contract.md](../03-api-contract.md) — the error envelope, the status table, `/healthz`
- [06-deployment.md](../06-deployment.md) — the environment variables config must load
