/**
 * TypeScript types for authentication
 */

/**
 * User data returned from authentication queries/mutations
 */
export interface User {
  id: string;
  email: string;
  firstName?: string | null;
  lastName?: string | null;
  confirmedAt?: string | null;
  createdAt: string;
}

/**
 * Authentication response from signIn/signUp mutations
 */
export interface AuthResponse {
  accessToken: string;
  expiresIn: number;
  user: User;
}

/**
 * Sign in input
 */
export interface SignInInput {
  email: string;
  password: string;
}

/**
 * Sign up input
 */
export interface SignUpInput {
  email: string;
  password: string;
  passwordConfirmation?: string;
  firstName?: string;
  lastName?: string;
}

/**
 * Forgot password input
 */
export interface ForgotPasswordInput {
  email: string;
}

/**
 * Reset password input
 */
export interface ResetPasswordInput {
  token: string;
  password: string;
  passwordConfirmation: string;
}

/**
 * Auth context value
 */
export interface AuthContextValue {
  user: User | null;
  loading: boolean;
  error: Error | null;
  signIn: (input: SignInInput) => Promise<AuthResponse>;
  signUp: (input: SignUpInput) => Promise<AuthResponse>;
  logout: () => Promise<void>;
  refetch: () => Promise<void>;
}





