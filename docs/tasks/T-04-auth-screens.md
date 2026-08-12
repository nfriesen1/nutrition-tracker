# T-04 — Login / signup / sign-out screens

**Goal.** A new user can create an account and sign in from inside the app.

**Depends on.** [T-03](T-03-supabase-client-session.md).

## Steps

1. `app/(auth)/login.tsx` — email and password, react-hook-form with a zod resolver, calling
   `supabase.auth.signInWithPassword`. A link to `/signup`.
2. `app/(auth)/signup.tsx` — email, password, confirmation, calling `supabase.auth.signUp`. A
   link back to `/login`. Password minimum must match what
   [T-01](T-01-supabase-project.md) recorded from the dashboard.
3. **No imperative navigation on success.** The session gate does it. Navigating here as well
   produces a double navigation and a visible flicker.
4. Error handling per [05-app-ui.md](../05-app-ui.md): Supabase's vague "Invalid login
   credentials" passed through unchanged on login, "User already registered" passed through on
   signup. Shown above the form, not attached to a field.
5. Submitting state: button disabled with a "Signing in…" label, inputs left enabled.
6. A temporary sign-out button on `diary.tsx` calling `supabase.auth.signOut()` and
   `queryClient.clear()`. [T-20](T-20-settings-screen.md) moves it to settings.

## Done when

- [ ] A brand-new email address can sign up and lands on `/diary` with no email confirmation
      step.
- [ ] Signing out returns to `/login`, and the browser back button does not get back into
      `/diary`.
- [ ] A wrong password shows the error and leaves the typed email in place.
- [ ] Submitting an invalid email never sends a request.
- [ ] The whole flow works on a device as well as in a browser.

## Notes

Clearing the query cache on sign-out is not cosmetic — without it the next user to sign in on
that device sees the previous user's cached diary for a frame.

**Milestone:** with [T-05](T-05-go-service-scaffold.md) done, both processes run locally. That
is the state `README.md` describes.

## Spec references

- [05-app-ui.md](../05-app-ui.md) — both screens, states, and the error-message reasoning
- [04-auth.md](../04-auth.md) — why the signup error is an acceptable existence oracle and the
  login error is not
