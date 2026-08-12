import { zodResolver } from '@hookform/resolvers/zod';
import { Link } from 'expo-router';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, Text, View } from 'react-native';

import { loginSchema, type LoginValues } from '@/auth/schemas';
import { supabase } from '@/auth/supabase';
import { ErrorNotice } from '@/components/error-notice';
import { SubmitButton } from '@/components/submit-button';
import { TextField } from '@/components/text-field';

export default function Login() {
  const [formError, setFormError] = useState<string | null>(null);

  const { control, handleSubmit, formState } = useForm<LoginValues>({
    resolver: zodResolver(loginSchema),
    defaultValues: { email: '', password: '' },
  });

  const onSubmit = async (values: LoginValues) => {
    setFormError(null);

    const { error } = await supabase.auth.signInWithPassword({
      email: values.email.trim(),
      password: values.password,
    });

    if (error) {
      // Passed through unchanged on purpose. Supabase answers "Invalid login credentials"
      // for both a wrong password and an unknown email, and that vagueness is what stops
      // this form being an account-existence oracle. Do not improve on it.
      setFormError(error.message);
      return;
    }

    // No navigation here. onAuthStateChange fires, the session context updates, the
    // (auth) gate stops matching, and the router lands on /diary. Navigating imperatively
    // as well produces a double navigation and a visible flicker.
  };

  return (
    <KeyboardAvoidingView
      style={styles.flex}
      behavior={Platform.OS === 'ios' ? 'padding' : undefined}
    >
      <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
        <Text style={styles.heading}>Sign in</Text>

        {formError ? <ErrorNotice message={formError} /> : null}

        <TextField
          control={control}
          name="email"
          label="Email"
          placeholder="you@example.com"
          keyboardType="email-address"
          inputMode="email"
          autoCapitalize="none"
          autoCorrect={false}
          autoComplete="email"
          textContentType="emailAddress"
        />

        <TextField
          control={control}
          name="password"
          label="Password"
          secureTextEntry
          autoCapitalize="none"
          autoComplete="current-password"
          textContentType="password"
          onSubmitEditing={handleSubmit(onSubmit)}
          returnKeyType="go"
        />

        <SubmitButton
          label="Sign in"
          busyLabel="Signing in…"
          busy={formState.isSubmitting}
          onPress={handleSubmit(onSubmit)}
        />

        <View style={styles.footer}>
          <Text style={styles.footerText}>No account? </Text>
          <Link href="/signup" style={styles.link}>
            Create one
          </Link>
        </View>
      </ScrollView>
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  content: { flexGrow: 1, justifyContent: 'center', gap: 16, padding: 24, maxWidth: 420, width: '100%', alignSelf: 'center' },
  heading: { fontSize: 28, fontWeight: '700', marginBottom: 8 },
  footer: { flexDirection: 'row', justifyContent: 'center', marginTop: 8 },
  footerText: { fontSize: 15, opacity: 0.7 },
  link: { fontSize: 15, color: '#2563eb', fontWeight: '600' },
});
