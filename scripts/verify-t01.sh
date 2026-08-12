#!/usr/bin/env bash
#
# Verifies the checkable parts of T-01 (docs/tasks/T-01-supabase-project.md).
#
#   ./scripts/verify-t01.sh
#
# Reads app/.env.local and api/.env.local. Prints no secret values — only whether
# each one is set and whether it works. Safe to run with the output pasted into a PR.
#
# Requires: curl. psql is optional; the database check is skipped without it.

set -uo pipefail

cd "$(dirname "$0")/.." || exit 1

pass=0
fail=0
skip=0

ok()   { printf '  \033[32mPASS\033[0m  %s\n' "$1"; pass=$((pass + 1)); }
bad()  { printf '  \033[31mFAIL\033[0m  %s\n' "$1"; fail=$((fail + 1)); }
note() { printf '  \033[33mSKIP\033[0m  %s\n' "$1"; skip=$((skip + 1)); }

# True when a value is still template text rather than a real one.
#
# Covers both conventions in play: this repo's .env.example files use <angle>
# brackets, and Supabase's dashboard hands out connection strings containing a
# literal [YOUR-PASSWORD]. Missing the second is how an unsubstituted password
# reaches a deploy and fails as an authentication error rather than a config one.
is_placeholder() {
  case "$1" in
    ""|*"<"*|*"["*"]"*|*YOUR-PASSWORD*|*your-password*) return 0 ;;
    *) return 1 ;;
  esac
}

echo
echo "T-01 — Supabase project setup"
echo "─────────────────────────────────────────────────────────────"

# ── env files ────────────────────────────────────────────────────────────────
echo
echo "Environment files"

for f in app/.env.local api/.env.local; do
  if [ -f "$f" ]; then ok "$f exists"; else bad "$f is missing — cp ${f%.local}.example $f"; fi
done

# The .env.local files must never be committable. .gitignore already covers them;
# this confirms it, because a leaked DATABASE_URL is the worst outcome of this task.
if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  for f in app/.env.local api/.env.local; do
    [ -f "$f" ] || continue
    if git check-ignore -q "$f"; then ok "$f is gitignored"; else bad "$f is NOT gitignored — do not commit"; fi
  done
else
  note "not a git repository — cannot check gitignore"
fi

[ -f app/.env.local ] && set -a && . ./app/.env.local && set +a
[ -f api/.env.local ] && set -a && . ./api/.env.local && set +a

# ── required variables ───────────────────────────────────────────────────────
echo
echo "Required variables (values not printed)"

for var in EXPO_PUBLIC_SUPABASE_URL EXPO_PUBLIC_SUPABASE_ANON_KEY DATABASE_URL; do
  val="${!var:-}"
  if ! is_placeholder "$val"; then ok "$var is set"; else bad "$var is empty or still a placeholder"; fi
done

# The service role key must never reach the client bundle. Every EXPO_PUBLIC_* value
# is compiled into the shipped JavaScript — see docs/04-auth.md.
if grep -rqi 'service.role' app/.env.local 2>/dev/null; then
  bad "app/.env.local mentions a service role key — REMOVE IT, it bypasses all authorization"
else
  ok "no service role key in app/.env.local"
fi

# ── JWT signing mode ─────────────────────────────────────────────────────────
echo
echo "JWT signing mode (docs/04-auth.md)"

if [ -n "${SUPABASE_JWT_SECRET:-}" ]; then
  note "SUPABASE_JWT_SECRET is set — legacy shared-secret mode. T-08 implements the HS256 path."
  if ! is_placeholder "${SUPABASE_JWKS_URL:-}"; then
    bad "both SUPABASE_JWT_SECRET and SUPABASE_JWKS_URL are set — pick ONE path"
  fi
elif ! is_placeholder "${SUPABASE_JWKS_URL:-}"; then
  body=$(curl -fsS --max-time 10 "$SUPABASE_JWKS_URL" 2>/dev/null)
  if [ -z "$body" ]; then
    bad "JWKS endpoint unreachable: $SUPABASE_JWKS_URL"
  elif echo "$body" | grep -q '"kty"'; then
    ok "JWKS endpoint returns a key set (asymmetric mode — the preferred path)"
  elif echo "$body" | grep -q '"keys"[[:space:]]*:[[:space:]]*\[[[:space:]]*\]'; then
    bad "JWKS endpoint returned an EMPTY key set — the project is probably still on the legacy shared secret. Migrate it in the dashboard, or set SUPABASE_JWT_SECRET and implement the HS256 path in T-08."
  else
    bad "JWKS endpoint returned something unexpected"
  fi
else
  bad "neither SUPABASE_JWKS_URL nor SUPABASE_JWT_SECRET is set — T-01 step 4 is unanswered"
fi

for var in SUPABASE_JWT_ISS SUPABASE_JWT_AUD; do
  val="${!var:-}"
  if ! is_placeholder "$val"; then ok "$var is set"; else bad "$var is empty or still a placeholder"; fi
done

# ── database ─────────────────────────────────────────────────────────────────
echo
echo "Database"

if is_placeholder "${DATABASE_URL:-}"; then
  bad "DATABASE_URL is empty or still contains a placeholder password"
else
  # Shape first, and unconditionally — this is the check that catches the most common
  # mistake, and it needs no Postgres client. docs/06-deployment.md: the pooler on
  # 6543, not the direct connection on 5432. A pgx pool against the direct port will
  # exhaust Supabase's connection cap once Fly runs more than one machine.
  case "$DATABASE_URL" in
    *:6543/*) ok "using the transaction pooler (port 6543)" ;;
    *:5432/*) bad "using port 5432 — that is the DIRECT connection (or the session pooler). Switch to the transaction pooler on 6543." ;;
    *)        note "could not determine the port from DATABASE_URL" ;;
  esac

  case "$DATABASE_URL" in
    *@*.pooler.supabase.com:*) ok "host is the Supabase pooler" ;;
    *@db.*.supabase.co:*)      bad "host is db.<ref>.supabase.co — that is the direct connection, not the pooler" ;;
    *)                         note "unrecognised database host" ;;
  esac

  # postgres.<ref> rather than bare postgres is what the pooler expects.
  case "$DATABASE_URL" in
    postgresql://postgres.*) ok "user is postgres.<ref>, as the pooler requires" ;;
    postgresql://postgres:*) bad "user is bare 'postgres' — the pooler needs postgres.<project-ref>" ;;
    *)                       note "could not determine the user from DATABASE_URL" ;;
  esac

  if command -v psql >/dev/null 2>&1; then
    if psql "$DATABASE_URL" -c 'select 1' >/dev/null 2>&1; then
      ok "database reachable and credentials accepted"
    else
      bad "cannot connect with DATABASE_URL"
    fi
  else
    note "psql not installed — cannot test the password. Shape checks above still ran; the real connection is proven in T-06."
  fi
fi

# ── auth config ──────────────────────────────────────────────────────────────
echo
echo "Auth configuration"

if ! is_placeholder "${EXPO_PUBLIC_SUPABASE_URL:-}"; then
  settings=$(curl -fsS --max-time 10 "${EXPO_PUBLIC_SUPABASE_URL}/auth/v1/settings" \
    -H "apikey: ${EXPO_PUBLIC_SUPABASE_ANON_KEY:-}" 2>/dev/null)
  if [ -z "$settings" ]; then
    note "could not read /auth/v1/settings — check email signup by hand in the dashboard"
  else
    if echo "$settings" | grep -q '"email"[[:space:]]*:[[:space:]]*true'; then
      ok "email provider is enabled"
    else
      bad "email provider does not look enabled — T-01 step 2"
    fi
    # Step 2 disables confirmation so signup returns a live session in one step.
    if echo "$settings" | grep -q '"mailer_autoconfirm"[[:space:]]*:[[:space:]]*true'; then
      ok "email confirmation is disabled (autoconfirm on)"
    else
      bad "email confirmation still appears to be ON — signup will not return a session (T-01 step 2)"
    fi
  fi
else
  note "EXPO_PUBLIC_SUPABASE_URL not set — skipping auth settings check"
fi

# ── manual ───────────────────────────────────────────────────────────────────
echo
echo "Still manual — this script cannot check these"
echo "  ·  Record section in docs/tasks/T-01-supabase-project.md is filled in"
echo "  ·  credentials stored in a password manager, not in this repo"
echo "  ·  the project's region is noted (T-16 matches it in fly.toml)"
echo "  ·  the password minimum length is noted (T-04's zod schema matches it)"

echo
echo "─────────────────────────────────────────────────────────────"
printf '  %d passed, %d failed, %d skipped\n\n' "$pass" "$fail" "$skip"

[ "$fail" -eq 0 ]
