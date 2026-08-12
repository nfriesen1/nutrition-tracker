import type { Session, User } from '@supabase/supabase-js';
import { createContext, useContext, useEffect, useMemo, useState, type ReactNode } from 'react';

import { supabase } from '@/auth/supabase';

type SessionState = {
  session: Session | null;
  user: User | null;
  /**
   * True until the persisted session has been read from storage.
   *
   * Consumers must not treat `loading` as "signed out". The route gate in
   * app/(app)/_layout.tsx renders a splash while this is true and redirects nothing —
   * redirecting on the first frame bounces every returning user through /login.
   */
  loading: boolean;
};

const SessionContext = createContext<SessionState | undefined>(undefined);

export function SessionProvider({ children }: { children: ReactNode }) {
  const [session, setSession] = useState<Session | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let active = true;

    // Reading the persisted session is asynchronous — from localStorage on web, from
    // AsyncStorage on native. Until it resolves there is no session yet, which is not
    // the same as being signed out.
    supabase.auth.getSession().then(({ data }) => {
      if (!active) return;
      setSession(data.session);
      setLoading(false);
    });

    // Keeps the context current for the rest of the app's life: token refreshes, sign
    // out, and sign out in another tab all arrive here. supabase-js also emits an
    // INITIAL_SESSION event on subscribe, so this doubles as a backstop if getSession
    // rejects.
    const {
      data: { subscription },
    } = supabase.auth.onAuthStateChange((_event, nextSession) => {
      if (!active) return;
      setSession(nextSession);
      setLoading(false);
    });

    return () => {
      active = false;
      subscription.unsubscribe();
    };
  }, []);

  const value = useMemo<SessionState>(
    () => ({ session, user: session?.user ?? null, loading }),
    [session, loading],
  );

  return <SessionContext.Provider value={value}>{children}</SessionContext.Provider>;
}

export function useSession(): SessionState {
  const context = useContext(SessionContext);
  if (context === undefined) {
    throw new Error('useSession must be used inside <SessionProvider>');
  }
  return context;
}
