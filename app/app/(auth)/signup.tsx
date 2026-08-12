import { zodResolver } from '@hookform/resolvers/zod';
import { Link } from 'expo-router';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, Text, View } from 'react-native';

import { PASSWORD_MIN_LENGTH, signupSchema, type SignupValues } from '@/auth/schemas';
import { supabase } from '@/auth/supabase';
import { ErrorNotice } from '@/components/error-notice';
import { SubmitButton } from '@/components/submit-button';
import { TextField } from '@/components/text-field';

export default function Signup() {
  const [formError, setFormError] = useState<string | null>(null);

  const { control, handleSubmit, formState } = useForm<SignupValues>({
    resolver: zodResolver(signupSchema),
    defaultValues: { email: '', password: '', confirmPassword: '' },
  });

  const onSubmit = async (values: SignupValues) => {
    setFormError(null);

    const { data, error } = await supabase.auth.signUp({
      email: values.email.trim(),
      password: values.password,
    });

    if (error) {
      // "User already registered" is passed through. This *is* an account-existence
      // oracle, and it is the trade every consumer signup makes: the alternative is a
      // form that appears to succeed and sends nothing, which users read as a broken app.
      setFormError(error.message);
      return;
    }

    if (!data.session) {
      // Email confirmation is off in T-01, so signup returns a live session and the gate
      // navigates. If confirmation is ever turned back on this branch is what fires, and
      // silently doing nothing would look like a hung button.
      setFormError('Check your email to confirm your account, then sign in.');
      return;
    }

    // No navigation on success — same reason as login.
  };

  return (
    <KeyboardAvoidingView
      style={styles.flex}
      behavior={Platform.OS === 'ios' ? 'padding' : undefined}
    >
      <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
        <Text style={styles.heading}>Create an account</Text>

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
          placeholder={`At least ${PASSWORD_MIN_LENGTH} characters`}
          secureTextEntry
          autoCapitalize="none"
          autoComplete="new-password"
          textContentType="newPassword"
        />

        <TextField
          control={control}
          name="confirmPassword"
          label="Confirm password"
          secureTextEntry
          autoCapitalize="none"
          autoComplete="new-password"
          textContentType="newPassword"
          onSubmitEditing={handleSubmit(onSubmit)}
          returnKeyType="go"
        />

        <SubmitButton
          label="Create account"
          busyLabel="Creating account…"
          busy={formState.isSubmitting}
          onPress={handleSubmit(onSubmit)}
        />

        <View style={styles.footer}>
          <Text style={styles.footerText}>Already have an account? </Text>
          <Link href="/login" style={styles.link}>
            Sign in
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
