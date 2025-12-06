/**
 * English translations
 *
 * All user-facing text in English language
 */

export const en = {
  auth: {
    errors: {
      invalidCredentials: "Invalid email or password. Please check your credentials.",
      serverError: "Server error. Please try again later.",
      authenticationFailed: "Authentication failed. Please try again.",
      genericError: "Failed to sign in. Please check your credentials.",
      emailRequired: "Email is required.",
      passwordRequired: "Password is required.",
      sessionExpired: "Session expired. Please sign in again.",
    },
    login: {
      title: "Sign in",
      subtitle: "Enter your credentials to continue",
      emailLabel: "Email",
      emailPlaceholder: "Enter your email",
      passwordLabel: "Password",
      passwordPlaceholder: "Enter your password",
      submitButton: "Sign in",
      submittingButton: "Signing in...",
      forgotPassword: "Forgot password?",
    },
  },
  common: {
    loading: "Loading...",
    error: "Error",
    success: "Success",
    cancel: "Cancel",
    save: "Save",
    delete: "Delete",
    edit: "Edit",
    create: "Create",
    close: "Close",
    yes: "Yes",
    no: "No",
    ok: "OK",
  },
  validation: {
    required: "Required field",
    email: "Invalid email",
    minLength: "Minimum length: {min} characters",
    maxLength: "Maximum length: {max} characters",
  },
} as const;

export type Translations = typeof en;
