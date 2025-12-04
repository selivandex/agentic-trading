import { ForgotPasswordForm } from "@/features/auth/ui";
import { AuthLayout } from "@/widgets/auth-layout/ui";

export default function ForgotPasswordPage() {
  return (
    <AuthLayout 
      title="Reset password" 
      subtitle="Enter your email to receive a reset link"
    >
      <ForgotPasswordForm />
    </AuthLayout>
  );
}

