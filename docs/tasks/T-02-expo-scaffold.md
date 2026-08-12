# T-02 — Expo app scaffold

**Goal.** `app/` holds an Expo Router project in TypeScript that runs on web and on a device,
with the route groups from [01-architecture.md](../01-architecture.md) in place.

**Depends on.** Nothing.

## Steps

1. Scaffold into `app/` with the TypeScript + Expo Router template
   (`npx create-expo-app@latest app --template`). The directory already exists and is empty.
2. Install the dependencies the rest of the app needs:
   - `@supabase/supabase-js`, `@react-native-async-storage/async-storage`
   - `@tanstack/react-query`
   - `react-hook-form`, `zod`, `@hookform/resolvers`
3. Create the route tree as empty placeholder screens:
   ```
   app/_layout.tsx
   app/(auth)/_layout.tsx   login.tsx   signup.tsx
   app/(app)/_layout.tsx    diary.tsx   add-food.tsx   create-food.tsx   settings.tsx
   ```
   Each screen renders its own name and nothing else for now.
4. Create the empty `src/` directories from
   [01-architecture.md](../01-architecture.md): `api/`, `auth/`, `schemas/`, `components/`,
   `lib/`.
5. Set `"strict": true` in `tsconfig.json` if the template did not, and add the `@/*` path alias
   to `src/*`.
6. Confirm `app.json` has a scheme set — deep links need it later, and adding it after a native
   build exists is more disruptive.
7. Add `npm run typecheck` → `tsc --noEmit` to `package.json`.

## Done when

- [ ] `npx expo start --web` opens the app in a browser.
- [ ] `npx expo start` and the Expo Go app show the same screen on a phone.
- [ ] Navigating to `/diary` in the browser address bar renders the diary placeholder — this
      confirms Expo Router is producing real URLs.
- [ ] `npm run typecheck` passes.
- [ ] `npx expo export --platform web` writes to `app/dist/` — the same command
      `netlify.toml` runs.

## Notes

Run the export once here even though nothing is deployed until
[T-16](T-16-first-deploy.md). It is much easier to fix a build problem now than in the middle of
a first deploy.

No UI library. [05-app-ui.md](../05-app-ui.md) explains why: six screens is not enough to earn
one.

## Spec references

- [01-architecture.md](../01-architecture.md) — the app directory layout
- [05-app-ui.md](../05-app-ui.md) — the routes and what each becomes
- [06-deployment.md](../06-deployment.md) — the Netlify build command and Node version
