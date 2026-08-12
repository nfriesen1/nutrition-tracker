# 03 — API Contract

The Go service exposes ten endpoints. This document is the authority on their shapes; the zod
schemas in `app/src/schemas/` are the client-side mirror of it, and a mismatch between the two
should show up as a parse error rather than as `undefined` three components deep.

## Conventions

**Base path.** Everything is under `/api/v1` except `GET /healthz`. On web the browser calls
same-origin `/api/...` and Netlify proxies it; on native the app calls the Fly hostname
directly. See [06-deployment.md](06-deployment.md).

**Content type.** Requests and responses are `application/json; charset=utf-8`. A request with
a body and no JSON content type is `415`.

**Auth.** Every `/api/v1` endpoint requires `Authorization: Bearer <access_token>`. There are no
optional-auth endpoints. `/healthz` takes no auth. See [04-auth.md](04-auth.md).

**`user_id` is never accepted from the client.** Not in a body, not in a query string, not in a
path. It comes from the verified JWT and nowhere else. Any request body field named `user_id`
is ignored — silently, because rejecting it would tell a prober that the field means something.

**Field naming is `snake_case`,** matching the database columns. The app's zod schemas keep the
same names rather than camel-casing on the boundary, so a field can be traced from Postgres to
JSX without a rename table.

**Numbers are JSON numbers, not strings.** `numeric(7,2)` columns serialise as `157.5`, not
`"157.50"`. Trailing zeroes are not preserved — the client formats for display.

**Dates are `YYYY-MM-DD` strings.** Never timestamps, never epoch seconds. `created_at` and
`updated_at` are not returned by any endpoint in slice 1; nothing in the UI shows them.

**Unknown fields in a request body are ignored,** not rejected. This keeps a newer client
working against an older server during a deploy.

## Error envelope

Every non-2xx response has exactly this shape, produced by `httpx/respond.go`:

```json
{
  "error": {
    "code": "validation_failed",
    "message": "quantity must be greater than 0",
    "details": { "quantity": "must be greater than 0" }
  }
}
```

- `code` — a stable machine-readable string. The client switches on this, never on `message`.
- `message` — one human-readable sentence, safe to show to a user.
- `details` — optional. Present only for `validation_failed`, where it maps field name to
  problem so a form can highlight the right input.

### Status codes

| Status | `code` | When |
|---|---|---|
| `400` | `bad_request` | Malformed JSON, wrong content type, unparseable query parameter |
| `401` | `unauthenticated` | Missing, malformed, expired, or unverifiable token |
| `404` | `not_found` | No such resource — **or it exists and is not yours** |
| `409` | `conflict` | Duplicate custom food (unique index violation) |
| `422` | `validation_failed` | Well-formed JSON that breaks a rule |
| `500` | `internal_error` | Anything unexpected. `message` is generic; the detail is logged with the request id, not returned |

`400` versus `422` is the distinction between "I cannot read this" and "I read it and it is
wrong." A body that is not JSON is `400`; `{"quantity": -1}` is `422`.

**There is no `403`.** A resource that exists but belongs to another user returns `404`, with
the same body as a resource that does not exist at all. A `403` would confirm that the id is
real, which is a slow enumeration oracle over other users' custom foods. This costs nothing —
there is no legitimate flow where a user needs to be told "that exists but is not yours."

These map to Go sentinel errors in `httpx/errors.go` (`ErrNotFound`, `ErrValidation`, …); a
handler returns a sentinel and the responder chooses the status. Handlers do not write status
codes directly.

---

## `GET /healthz`

Liveness and readiness for Fly's health check. No auth.

```
200 OK
{ "status": "ok", "version": "a1b2c3d" }
```

`version` is the git SHA, injected at build time with `-ldflags`. No database round-trip: a
health check that fails when Postgres blips would have Fly restart a service that is fine, and
restarting does not fix a database. Database reachability is a monitoring concern, not a
liveness one.

---

## `GET /api/v1/me`

The caller's profile. **Creates the row if it does not exist** — this is the lazy profile
creation described in [02-data-model.md](02-data-model.md), and it is why the app calls this
endpoint once on session start.

```
200 OK
{
  "user_id": "9f8e7d6c-...",
  "daily_calorie_goal": 2000
}
```

The upsert is `insert ... on conflict (user_id) do nothing` followed by the select, in one
transaction. Two concurrent first-requests therefore both succeed rather than one hitting a
primary key violation.

## `PATCH /api/v1/me`

```json
{ "daily_calorie_goal": 2200 }
```

```
200 OK
{ "user_id": "9f8e7d6c-...", "daily_calorie_goal": 2200 }
```

| Rule | Failure |
|---|---|
| `daily_calorie_goal` required, integer | `422` |
| `1 ≤ goal ≤ 20000` | `422` |

Also upserts, so a user who never called `GET /api/v1/me` can still set a goal. `PATCH` rather
than `PUT` because the profile will grow more fields and partial update is the shape that
survives that.

---

## `GET /api/v1/foods`

Prefix search over seed foods and the caller's own foods.

| Parameter | Type | Default | Notes |
|---|---|---|---|
| `q` | string | `""` | Prefix to match against the food name |
| `limit` | integer | `25` | Clamped to `1..50`; out-of-range is clamped, not rejected |

```
GET /api/v1/foods?q=ban&limit=25

200 OK
{
  "foods": [
    {
      "id": "a1b2c3d4-...",
      "name": "Banana, raw",
      "brand": null,
      "serving_label": "medium (118 g)",
      "calories": 105,
      "protein_g": 1.29,
      "carbs_g": 26.95,
      "fat_g": 0.39,
      "is_custom": false
    }
  ]
}
```

`is_custom` is `owner_id is not null` — computed, not stored. The client uses it to show a
"yours" badge and to decide whether editing is offered. `owner_id` itself is never returned;
the client has no use for a uuid it already knows.

**`q` shorter than two characters returns `{"foods": []}` without touching the database.** A
one-character prefix matches a large fraction of the table and the result is useless to a user
anyway. This is a guard against the search-as-you-type call that fires on the first keystroke.

Ordering: caller's own foods first, then seed foods, each alphabetical by name. Custom foods
first because a user who bothered to create one is usually looking for it.

The result set is the union of `owner_id is null` and `owner_id = caller`. There is no
pagination — with a curated seed list and a `limit` of 50, there is nothing to page through.
Adding a third-party food database is when a cursor appears.

## `POST /api/v1/foods`

Create a custom food. Owned by the caller.

```json
{
  "name": "Mum's granola",
  "brand": null,
  "serving_label": "1/2 cup (55 g)",
  "calories": 240,
  "protein_g": 6.1,
  "carbs_g": 31.4,
  "fat_g": 10.2
}
```

```
201 Created
{ "id": "...", "name": "Mum's granola", ..., "is_custom": true }
```

| Rule | Failure |
|---|---|
| `name` required, 1–200 chars after trimming | `422` |
| `brand` optional, ≤ 100 chars, `null` and `""` both mean absent | `422` |
| `serving_label` required, 1–100 chars | `422` |
| `calories` required, `0 ≤ n ≤ 99999.99` | `422` |
| `protein_g` / `carbs_g` / `fat_g` optional, default `0`, same range | `422` |
| Name+brand not already used by this caller | `409` |

Values are rounded to two decimals on write; the response echoes what was stored, so a client
that sends `10.239` gets `10.24` back and can display the truth.

The `409` comes from the partial unique index in [02-data-model.md](02-data-model.md), caught
by inspecting the pgx error's constraint name rather than by a pre-flight `select` — the
pre-flight version has a race, the constraint does not.

There is no macro/calorie consistency check. `4×protein + 4×carbs + 9×fat` will not equal
`calories` for most real foods (fibre, alcohol, rounding on the label), and rejecting on that
basis would refuse correct data.

## `GET /api/v1/foods/{id}`

```
200 OK
{ "id": "...", "name": "Banana, raw", ..., "is_custom": false }
```

`404` if the id does not exist, is not a uuid, **or belongs to another user**. All three
produce a byte-identical response.

Used when the add-food screen is opened via a deep link and the food is not already in the
search cache.

---

## `GET /api/v1/diary`

The day view. This is the endpoint the diary screen is built directly against, so it returns
everything that screen renders and requires no second call.

| Parameter | Type | Notes |
|---|---|---|
| `date` | `YYYY-MM-DD` | **Required.** No default — see below |

```
GET /api/v1/diary?date=2026-08-11

200 OK
{
  "date": "2026-08-11",
  "daily_calorie_goal": 2000,
  "totals": {
    "calories": 1487.5,
    "protein_g": 82.4,
    "carbs_g": 160.13,
    "fat_g": 54.9
  },
  "remaining_calories": 512.5,
  "meals": [
    {
      "meal": "breakfast",
      "subtotal": { "calories": 157.5, "protein_g": 1.94, "carbs_g": 40.43, "fat_g": 0.59 },
      "entries": [
        {
          "id": "b2c3d4e5-...",
          "food_id": "a1b2c3d4-...",
          "food_name": "Banana, raw",
          "food_brand": null,
          "serving_label": "medium (118 g)",
          "quantity": 1.5,
          "calories": 157.5,
          "protein_g": 1.94,
          "carbs_g": 40.43,
          "fat_g": 0.59
        }
      ]
    },
    { "meal": "lunch",  "subtotal": { ... }, "entries": [] },
    { "meal": "dinner", "subtotal": { ... }, "entries": [] },
    { "meal": "snack",  "subtotal": { ... }, "entries": [] }
  ]
}
```

Four things about this shape:

1. **`meals` is always an array of exactly four objects, in fixed order** — breakfast, lunch,
   dinner, snack — even on a day with no entries at all. The client never has to construct an
   empty meal section or decide the order. An empty day is four objects with zero subtotals and
   empty `entries`.
2. **`meals` is an array, not an object keyed by meal name.** Order in a JSON object is not
   guaranteed by the format, and relying on it is the kind of thing that works until it does
   not.
3. **The nutrition numbers on each entry are the snapshot columns**, read straight from
   `diary_entries`. `food_name`, `food_brand`, and `serving_label` come from the join to
   `foods` — display only. This is the read-side half of the rule in
   [02-data-model.md](02-data-model.md).
4. **`remaining_calories` is `goal − totals.calories` and may be negative.** The server does not
   clamp it; going over is real information and the UI shows it.

`entries` within a meal are ordered by `created_at` ascending — the order they were logged,
which is the order a user expects to see them in.

**`date` has no default.** Omitting it is `422`, not "today". The server deciding what "today"
means is exactly the timezone bug that [02-data-model.md](02-data-model.md) exists to prevent;
only the client knows the user's calendar day. A `date` that is not a valid `YYYY-MM-DD` is
`422` — including `2026-02-30`, which is well-formed but not a date.

Subtotals and totals are computed in SQL (`sum(...) group by meal`), not in Go. The query
returns four rows plus the entry list; the handler assembles them. This is the `GROUP BY` that
[00-overview.md](00-overview.md) cites as the reason for a relational store.

## `POST /api/v1/diary/entries`

Log a food.

```json
{
  "food_id": "a1b2c3d4-...",
  "entry_date": "2026-08-11",
  "meal": "breakfast",
  "quantity": 1.5
}
```

```
201 Created
{
  "id": "b2c3d4e5-...",
  "food_id": "a1b2c3d4-...",
  "food_name": "Banana, raw",
  "food_brand": null,
  "serving_label": "medium (118 g)",
  "entry_date": "2026-08-11",
  "meal": "breakfast",
  "quantity": 1.5,
  "calories": 157.5,
  "protein_g": 1.94,
  "carbs_g": 40.43,
  "fat_g": 0.59
}
```

| Rule | Failure |
|---|---|
| `food_id` required, valid uuid | `422` |
| Food exists and is seed-or-caller-owned | `404` |
| `entry_date` required, valid `YYYY-MM-DD` | `422` |
| `meal` required, one of the four | `422` |
| `quantity` required, `0 < n ≤ 1000` | `422` |

The handler steps, matching the lifecycle in [01-architecture.md](01-architecture.md): validate
the body, load the food under the ownership filter, scale each nutrient by `quantity` via
`internal/nutrition`, insert with `user_id` from context.

`quantity` is rounded to two decimals before scaling, so `1.005` becomes `1.01` and the stored
numbers match the returned `quantity` exactly.

Logging the same food twice to the same meal creates two rows. That is intentional — two
bananas at different times of the morning are two entries, and merging them would make deleting
one impossible.

There is no bulk endpoint. One entry per request; the UI has no flow that adds several at once.

## `PATCH /api/v1/diary/entries/{id}`

The only mutable field is `quantity`.

```json
{ "quantity": 2 }
```

```
200 OK
{ "id": "b2c3d4e5-...", "quantity": 2, "calories": 210, ... }
```

Response is the same shape as `POST`. `404` if the entry does not exist or is not the caller's.
`422` on the same `quantity` rules.

**The re-snapshot wrinkle, stated plainly:** the handler reloads the entry's original `food_id`
and rescales from that food's **current** per-serving values. So if the food was corrected
between logging and editing, changing the quantity picks up the correction — and changing the
quantity from `1.5` to `1.5` is a legitimate way to force a refresh.

This is a deliberate choice and it is the less obvious one. The alternative — scaling from the
entry's own stored snapshot — would preserve history perfectly but would mean a user who fixes
a food's calories can never repair the entry that motivated the fix, short of deleting and
re-adding it. The rule that falls out is easy to state: *entries never change on their own, but
touching one brings it up to date.*

Moving an entry between meals or days is not supported. Delete and re-add. There is no UI
affordance for it, and adding `meal` and `entry_date` to this endpoint would need the day-view
cache to invalidate two days instead of one.

## `DELETE /api/v1/diary/entries/{id}`

```
204 No Content
```

`404` if it does not exist or is not the caller's. Not idempotent in the "always 204" sense —
deleting twice gives `204` then `404`, which is the honest answer and is what the client's
error handling already covers.

Hard delete. There is no undo in slice 1.

---

## Endpoint summary

| Method | Path | Auth | Success | Handler file |
|---|---|---|---|---|
| `GET` | `/healthz` | no | `200` | `httpx/health.go` |
| `GET` | `/api/v1/me` | yes | `200` | `httpx/me.go` |
| `PATCH` | `/api/v1/me` | yes | `200` | `httpx/me.go` |
| `GET` | `/api/v1/foods` | yes | `200` | `httpx/foods.go` |
| `POST` | `/api/v1/foods` | yes | `201` | `httpx/foods.go` |
| `GET` | `/api/v1/foods/{id}` | yes | `200` | `httpx/foods.go` |
| `GET` | `/api/v1/diary` | yes | `200` | `httpx/diary.go` |
| `POST` | `/api/v1/diary/entries` | yes | `201` | `httpx/diary.go` |
| `PATCH` | `/api/v1/diary/entries/{id}` | yes | `200` | `httpx/diary.go` |
| `DELETE` | `/api/v1/diary/entries/{id}` | yes | `204` | `httpx/diary.go` |

Every route is registered in `internal/httpx/router.go` — one table, one place, as laid out in
[01-architecture.md](01-architecture.md). Go 1.22's method-aware patterns mean
`mux.HandleFunc("PATCH /api/v1/diary/entries/{id}", ...)` is the whole registration, and a
request to a known path with an unregistered method gets `405` from the mux itself.

## Versioning

`/api/v1` is in the path so that a breaking change can ship as `/api/v2` while the old client
keeps working. Slice 1 will not need it — but the app is on the App Store's release cycle, not
the API's, and a native build in the wild cannot be forced to update. Additive changes (new
optional field, new endpoint) do not bump the version; removing or retyping a field does.

## Related documents

- [02-data-model.md](02-data-model.md) — the columns behind these fields
- [04-auth.md](04-auth.md) — how the bearer token becomes a `user_id`
- [05-app-ui.md](05-app-ui.md) — which screen calls what, and the query keys
- [07-testing.md](07-testing.md) — one handler test per status branch above
