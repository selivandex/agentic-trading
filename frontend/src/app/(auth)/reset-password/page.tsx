import { ResetPasswordForm } from "@/features/auth/ui";
import { AuthLayout } from "@/widgets/auth-layout/ui";

export default function ResetPasswordPage() {
  return (
    <AuthLayout 
      title="Set new password" 
      subtitle="Enter your new password below"
    >
      <ResetPasswordForm />
    </AuthLayout>
  );
}
