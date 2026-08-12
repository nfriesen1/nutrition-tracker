import AsyncStorage from '@react-native-async-storage/async-storage';
import { createClient } from '@supabase/supabase-js';
import { Platform } from 'react-native';

/**
 * The Supabase client, constructed once for the life of the app.
 *
 * This is the only thing in the app that talks to Supabase. Login, signup, session
 * persistence, and token refresh all happen here; the Go API is never involved in
 * authentication. See docs/04-auth.md.
 */

// These must be read as full static property accesses. Expo inlines EXPO_PUBLIC_*
// at build time by substituting the literal text, so destructuring or computing the
// key leaves `undefined` in the bundle.
const supabaseUrl = process.env.EXPO_PUBLIC_SUPABASE_URL;
const supabaseAnonKey = process.env.EXPO_PUBLIC_SUPABASE_ANON_KEY;

if (!supabaseUrl || !supabaseAnonKey) {
  // Fail at module load rather than handing createClient an undefined. A client built
  // with undefined credentials fails later, on the first request, with an error that
  // points at the network instead of at the missing config.
  const missing = [
    !supabaseUrl && 'EXPO_PUBLIC_SUPABASE_URL',
    !supabaseAnonKey && 'EXPO_PUBLIC_SUPABASE_ANON_KEY',
  ]
    .filter(Boolean)
    .join(', ');

  throw new Error(
    `Missing Supabase configuration: ${missing}.\n` +
      'Copy app/.env.example to app/.env.local and fill it in from the Supabase ' +
      'dashboard (T-01), then restart the dev server — Expo reads .env.local at ' +
      'startup, so a running server will not pick up changes.',
  );
}

export const supabase = createClient(supabaseUrl, supabaseAnonKey, {
  auth: {
    // The one place the app branches on platform for storage. Web gets the library's
    // default (localStorage); native has no such global and must be handed a store.
    storage: Platform.OS === 'web' ? undefined : AsyncStorage,
    persistSession: true,
    autoRefreshToken: true,
    // No OAuth or magic-link redirect to parse in slice 1. Leaving this on makes the
    // library inspect every URL the web app loads.
    detectSessionInUrl: false,
  },
});
