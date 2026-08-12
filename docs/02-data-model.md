# 02 — Data Model

Three tables: `profiles`, `foods`, `diary_entries`. They live in the `public` schema of the
Supabase Postgres database, alongside Supabase's own `auth` schema — which this service reads
from never and writes to never.

## Relationship to `auth.users`

Supabase owns the `auth.users` table. We do not add foreign keys to it, and we do not add
triggers on it.

```
auth.users (Supabase)          public.profiles
  id ─────────────────────────▶  user_id        (same uuid, no FK)
```

The link is by convention: `profiles.user_id` holds the same uuid that Supabase puts in the
JWT's `sub` claim. There is no database-level foreign key, for two reasons. Cross-schema FKs
into `auth` are fragile across Supabase platform upgrades, and a trigger on `auth.users` would
make something other than the Go service a writer to our tables — which contradicts the rule
in [01-architecture.md](01-architecture.md).

Instead, **the profile row is created lazily**: the first authenticated request from a user
upserts it. See [03-api-contract.md](03-api-contract.md), `GET /api/v1/me`.

The cost of having no FK is that deleting a user in the Supabase dashboard leaves orphaned
rows. That is acceptable for slice 1 and is a known cleanup item.

## `profiles`

One row per user. Holds the only piece of user configuration slice 1 has: the calorie goal.

```sql
create table profiles (
    user_id            uuid        primary key,
    daily_calorie_goal integer     not null default 2000
                                   check (daily_calorie_goal > 0
                                          and daily_calorie_goal <= 20000),
    created_at         timestamptz not null default now(),
    updated_at         timestamptz not null default now()
);
```

| Column | Notes |
|---|---|
| `user_id` | The JWT `sub` claim. Never accepted from a request body. |
| `daily_calorie_goal` | Whole calories. The upper bound is a sanity check, not nutrition advice — it stops a fat-fingered `200000` from making the diary screen render nonsense. |
| `updated_at` | Set explicitly by the update query (`set updated_at = now()`). No trigger. |

Macro goals are deliberately absent — [00-overview.md](00-overview.md) puts them out of scope.
When they arrive they are three nullable integer columns here, which is an additive migration.

## `foods`

Seed foods and user-created custom foods share one table.

```sql
create table foods (
    id            uuid          primary key default gen_random_uuid(),
    owner_id      uuid          null,
    name          text          not null check (length(trim(name)) between 1 and 200),
    brand         text          null check (brand is null or length(trim(brand)) <= 100),
    serving_label text          not null check (length(trim(serving_label)) between 1 and 100),
    calories      numeric(7, 2) not null check (calories >= 0),
    protein_g     numeric(7, 2) not null default 0 check (protein_g >= 0),
    carbs_g       numeric(7, 2) not null default 0 check (carbs_g >= 0),
    fat_g         numeric(7, 2) not null default 0 check (fat_g >= 0),
    created_at    timestamptz   not null default now(),
    updated_at    timestamptz   not null default now()
);
```

### `owner_id` is the entire visibility model

```
owner_id IS NULL      →  seed food, readable by every user, writable by nobody
owner_id = <uuid>     →  custom food, readable and writable only by that user
```

Every read of `foods` is filtered by `owner_id is null or owner_id = $caller`. There is no
third state, no sharing, no public/private flag. This is the condition the handler ownership
check in [01-architecture.md](01-architecture.md) is testing, and it is why a food belonging to
another user returns `404` rather than `403` — see [03-api-contract.md](03-api-contract.md).

Seed foods are inserted by a migration with `owner_id` left null; see
[tasks/T-14-seed-foods.md](tasks/T-14-seed-foods.md).

### Nutrition is stored per serving

`serving_label` is free text describing one serving: `"medium (118 g)"`, `"1 cup"`,
`"2 slices"`. The four nutrient columns describe exactly that amount.

This is the decision that keeps the whole app simple. There is no unit table, no gram-weight
conversion, no density. A diary entry stores a decimal multiplier against this one serving, and
scaling is a single multiply. The cost is that a user who wants "37 g of almonds" has to do the
arithmetic themselves, or create a custom food. Alternate units are a later slice.

`numeric(7, 2)` rather than `real` or `double precision`: this is a quantity users read off a
label and expect to see back unchanged. Binary floating point turns `105.1` into
`105.09999...`, and `numeric` is exact. The precision allows up to `99999.99`, which is beyond
any plausible single serving.

### Indexes

```sql
create index foods_name_prefix_idx on foods (lower(name) text_pattern_ops);

create unique index foods_owner_name_brand_idx
    on foods (owner_id, lower(trim(name)), coalesce(lower(trim(brand)), ''))
    where owner_id is not null;
```

**`foods_name_prefix_idx` is the search index, and it only works if the query matches its
shape.** Search must be written:

```sql
where lower(name) like lower($1) || '%'
```

Not `name ilike $1 || '%'`, and not `like '%' || $1 || '%'`. A leading wildcard cannot use a
B-tree index at all, and `ilike` against a plain `text` column will not match the functional
`lower(name)` expression. The `text_pattern_ops` operator class is what makes `like` use the
index regardless of the database's collation.

This design is right for a curated seed list of a few hundred foods. It finds "banana" when you
type "ban", and does not find it when you type "nana". If the food catalogue later grows to
third-party scale, the upgrade path is `pg_trgm` with a GIN index — and because search lives in
exactly one `.sql` file, that is a one-file change.

**`foods_owner_name_brand_idx`** stops a user accumulating three foods called "Oatmeal". It is
partial (`where owner_id is not null`) so it constrains only custom foods; seed data is curated
by hand and does not need enforcing. `coalesce(..., '')` is needed because `null` brands would
otherwise never collide — in SQL, `null` is not equal to `null`.

## `diary_entries`

The log. One row per food logged to a meal on a day.

```sql
create table diary_entries (
    id         uuid          primary key default gen_random_uuid(),
    user_id    uuid          not null,
    food_id    uuid          not null references foods (id) on delete restrict,
    entry_date date          not null,
    meal       text          not null check (meal in ('breakfast', 'lunch', 'dinner', 'snack')),
    quantity   numeric(6, 2) not null check (quantity > 0 and quantity <= 1000),
    calories   numeric(7, 2) not null check (calories >= 0),
    protein_g  numeric(7, 2) not null check (protein_g >= 0),
    carbs_g    numeric(7, 2) not null check (carbs_g >= 0),
    fat_g      numeric(7, 2) not null check (fat_g >= 0),
    created_at timestamptz   not null default now(),
    updated_at timestamptz   not null default now()
);

create index diary_entries_user_date_idx on diary_entries (user_id, entry_date);
```

### `entry_date` is a `date`, not a timestamp

This column is the schema-level expression of the rule stated in
[01-architecture.md](01-architecture.md). It is a Postgres `date` — no time, no zone, no
conversion. The client sends `"2026-08-11"` computed from the device's local clock; the server
stores those characters and nothing else.

`timestamptz` would be actively wrong here. Postgres normalises `timestamptz` to UTC on write,
so a 7pm dinner in UTC−6 would be stored as the following day at 01:00 and would come back on
the wrong diary page. There is no timezone column on `profiles` for the same reason: the server
never needs to know the user's zone, because it never decides what day anything is.

The server must never call `time.Now()` to derive `entry_date`. If a request does not carry a
date, that is a `422`, not a default.

### `meal` is a `CHECK`, not an enum

A native `create type meal_type as enum (...)` would give sqlc a generated Go type for free,
which is genuinely nicer. It is rejected because altering a Postgres enum is awkward — values
cannot be removed, and until recently could not be added inside a transaction, which makes
migrations fussy. A `text` column with a `CHECK` is trivially alterable and costs one
hand-written Go type:

```go
type Meal string

const (
    MealBreakfast Meal = "breakfast"
    // ...
)

func (m Meal) Valid() bool { ... }
```

Validation happens in the handler, so the `CHECK` is a backstop that should never fire. If it
does, that is a bug in validation, and it surfaces as a `500` — correctly, because it means the
service tried to write something it should have rejected.

### The nutrition columns are a snapshot

`calories`, `protein_g`, `carbs_g`, and `fat_g` on this table are **not** copies of the food's
values, and **not** recomputed on read. They are the food's per-serving values multiplied by
`quantity` at the moment of logging, and they are then frozen.

Every read of a diary day sums these columns. The join to `foods` still happens, but only to
fetch `name`, `brand`, and `serving_label` for display. If a query ever reads a nutrient value
out of `foods` while building a diary response, that is the bug this design exists to prevent.

The reason: a user who discovers their homemade granola is 480 kcal, not 380, and corrects the
custom food must not have last month's totals silently rewrite themselves. A diary is a record
of what was believed at the time.

The visible consequence, which is intended: correcting a food does **not** fix already-logged
entries. Re-logging is the fix. `PATCH /api/v1/diary/entries/{id}` re-snapshots from the food's
*current* values, so editing a quantity does pick up a correction — see
[03-api-contract.md](03-api-contract.md), which flags this as a deliberate wrinkle.

### `on delete restrict`

Deleting a food is not in slice 1's scope, and the FK makes that explicit: a food with history
attached cannot be removed. This protects the display join — an entry whose `food_id` dangles
would render with no name.

When food deletion arrives, it will be a soft delete (a `deleted_at` column that search filters
on), not a hard one, precisely so history stays readable.

### The index matches the only read

Every diary read is "this user, this day":

```sql
where user_id = $1 and entry_date = $2
```

`(user_id, entry_date)` serves it directly, and also serves a future date-range query for a
weekly view, since `user_id` is the leading column. No index on `food_id` — nothing looks up
entries by food, and the FK check on the `foods` side uses `foods.id`, the primary key.

## Worked example

A seed banana:

| id | owner_id | name | brand | serving_label | calories | protein_g | carbs_g | fat_g |
|---|---|---|---|---|---|---|---|---|
| `a1…` | `null` | `Banana, raw` | `null` | `medium (118 g)` | `105.00` | `1.29` | `26.95` | `0.39` |

The user logs one and a half of them to breakfast on 11 August 2026. The handler computes each
nutrient as `per-serving × quantity`, rounded half-away-from-zero to two decimals:

```
calories  105.00 × 1.5 = 157.500  → 157.50
protein     1.29 × 1.5 =   1.935  →   1.94
carbs      26.95 × 1.5 =  40.425  →  40.43
fat         0.39 × 1.5 =   0.585  →   0.59
```

The stored row:

| id | user_id | food_id | entry_date | meal | quantity | calories | protein_g | carbs_g | fat_g |
|---|---|---|---|---|---|---|---|---|---|
| `b2…` | `u9…` | `a1…` | `2026-08-11` | `breakfast` | `1.50` | `157.50` | `1.94` | `40.43` |

Two details that the rounding makes concrete:

- **Rounding happens once, at write.** Reads sum already-rounded values. Summing rounded numbers
  can drift from rounding a summed number by a cent or two across a big day, and that is the
  accepted trade: the number shown next to an entry always adds up to the subtotal shown above
  it, which is what a user actually checks.
- **The displayed calorie count is `round(sum)` to a whole number.** Storing two decimals and
  displaying zero is deliberate — the precision is there so a half-serving of something small
  does not vanish, not because anyone wants to see `157.50 kcal`.

The scaling and rounding rules are implemented in `internal/nutrition/scale.go` as pure
functions, and tested exhaustively — see [07-testing.md](07-testing.md).

## Migration order

goose migrations, applied in order at service boot. See [06-deployment.md](06-deployment.md).

| File | Contents |
|---|---|
| `0001_create_profiles.sql` | `profiles` |
| `0002_create_foods.sql` | `foods` + both indexes |
| `0003_create_diary_entries.sql` | `diary_entries` + index |
| `0004_seed_foods.sql` | the curated seed list ([T-14](tasks/T-14-seed-foods.md)) |

`foods` precedes `diary_entries` because of the foreign key. Every migration has a working
`-- +goose Down`, even though rolling back in production is not the plan — writing the down
migration is how you find out that you forgot to drop an index.

`gen_random_uuid()` is built into Postgres 13+ via `pgcrypto` being in core; Supabase runs 15+,
so no `create extension` is needed.

## Related documents

- [01-architecture.md](01-architecture.md) — where these tables sit at runtime
- [03-api-contract.md](03-api-contract.md) — how these columns are shaped into JSON
- [04-auth.md](04-auth.md) — where `user_id` comes from
- [07-testing.md](07-testing.md) — the tests that hold the two cross-cutting rules in place
