import { z } from 'zod';

/**
 * Client-side validation schemas (Zod).
 * These mirror the backend rules for UX/defense-in-depth only — the server is
 * always the source of truth and re-validates every request.
 */

export const loginSchema = z.object({
  username: z
    .string({ required_error: 'Username is required' })
    .trim()
    .min(1, 'Username is required')
    .max(50, 'Username must be at most 50 characters'),
  password: z
    .string({ required_error: 'Password is required' })
    .min(1, 'Password is required')
    .max(128, 'Password must be at most 128 characters'),
});

export type LoginForm = z.infer<typeof loginSchema>;

export const registerSchema = z
  .object({
    full_name: z
      .string({ required_error: 'Full name is required' })
      .trim()
      .min(1, 'Full name is required')
      .max(200, 'Full name must be at most 200 characters'),
    username: z
      .string({ required_error: 'Username is required' })
      .trim()
      .min(3, 'Username must be at least 3 characters')
      .max(50, 'Username must be at most 50 characters')
      .regex(/^[a-zA-Z0-9_.-]+$/, 'Username may only contain letters, numbers, dots, dashes and underscores'),
    email: z
      .string({ required_error: 'Email is required' })
      .trim()
      .email('Enter a valid email address')
      .max(200, 'Email must be at most 200 characters'),
    password: z
      .string({ required_error: 'Password is required' })
      .min(6, 'Password must be at least 6 characters')
      .max(128, 'Password must be at most 128 characters'),
    confirm_password: z.string({ required_error: 'Please confirm your password' }),
  })
  .refine((data) => data.password === data.confirm_password, {
    message: 'Passwords do not match',
    path: ['confirm_password'],
  });

export type RegisterForm = z.infer<typeof registerSchema>;

/** Cap pagination bounds (page >= 1, per_page 1..100) to mirror the backend. */
export const paginationSchema = z.object({
  page: z.coerce.number().int().min(1).default(1),
  per_page: z.coerce.number().int().min(1).max(100).default(20),
});

/** Generic field-level error helper. Returns a map keyed by field name. */
export function firstError(error: z.ZodError, fields: string[], fallback = 'Please check your input'): string {
  const issue = error.issues.find((i) => fields.includes(String(i.path[0])));
  return issue ? issue.message : fallback;
}
