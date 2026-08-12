import { useQueryClient } from '@tanstack/react-query';
import { Link } from 'expo-router';
import { Pressable, StyleSheet, Text, View } from 'react-native';

import { useSession } from '@/auth/SessionProvider';
import { supabase } from '@/auth/supabase';

/**
 * The real diary screen arrives in T-15. For now this is the placeholder plus a temporary
 * sign-out, which T-20 moves to settings.
 */
export default function Diary() {
  const { user } = useSession();
  const queryClient = useQueryClient();

  const signOut = async () => {
    await supabase.auth.signOut();
    // Not optional. Without it the next user to sign in on this device sees the previous
    // user's cached diary for a frame — a privacy problem on a shared device, not a
    // cosmetic one. See docs/05-app-ui.md.
    queryClient.clear();
    // No navigation: the (app) gate sees the session go away and redirects to /login.
  };

  return (
    <View style={styles.container}>
      <Text style={styles.name}>diary</Text>
      <Text style={styles.note}>placeholder — real screen in T-15</Text>

      <Text style={styles.email}>{user?.email ?? 'no session'}</Text>

      <Link href="/add-food" style={styles.link}>
        /add-food
      </Link>
      <Link href="/settings" style={styles.link}>
        /settings
      </Link>

      <Pressable onPress={signOut} accessibilityRole="button" style={styles.signOut}>
        <Text style={styles.signOutLabel}>Sign out</Text>
      </Pressable>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, alignItems: 'center', justifyContent: 'center', gap: 8, padding: 24 },
  name: { fontSize: 24, fontWeight: '600' },
  note: { fontSize: 13, opacity: 0.5 },
  email: { fontSize: 15, marginVertical: 12 },
  link: { fontSize: 16, color: '#2563eb', paddingVertical: 6 },
  signOut: {
    marginTop: 24,
    borderWidth: 1,
    borderColor: '#d4d4d8',
    borderRadius: 8,
    paddingVertical: 12,
    paddingHorizontal: 24,
  },
  signOutLabel: { fontSize: 16, fontWeight: '600' },
});
