import { z } from 'zod';

/**
 * Minimum password length the signup form enforces.
 *
 * The Supabase project itself allows 6 — verified against the live project, and recorded
 * in T-01. This form is deliberately stricter, which is safe (the server accepts anything
 * this passes) but means 8 is enforced only on the client until the dashboard's
 * "Minimum password length" is raised to match.
 *
 * Raise both together, or drop this to 6. A number that disagrees with the server in the
 * *other* direction — client looser than server — is the broken case: the form would
 * accept a password and the request would then fail with a server error the user cannot
 * act on.
 */
export const PASSWORD_MIN_LENGTH = 8;

export const loginSchema = z.object({
  email: z.email('Enter a valid email address'),
  // No minimum here. Login validates against what the account already has; imposing the
  // signup rule would lock out anyone whose password predates it, and the length of an
  // existing password is not this form's business.
  password: z.string().min(1, 'Enter your password'),
});

export const signupSchema = z
  .object({
    email: z.email('Enter a valid email address'),
    password: z
      .string()
      .min(PASSWORD_MIN_LENGTH, `Use at least ${PASSWORD_MIN_LENGTH} characters`),
    confirmPassword: z.string().min(1, 'Re-enter your password'),
  })
  .refine((values) => values.password === values.confirmPassword, {
    message: 'Passwords do not match',
    path: ['confirmPassword'],
  });

export type LoginValues = z.infer<typeof loginSchema>;
export type SignupValues = z.infer<typeof signupSchema>;
