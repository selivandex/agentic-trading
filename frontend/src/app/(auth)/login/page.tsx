import { LoginForm } from "@/features/auth/ui";
import { AuthLayout } from "@/widgets/auth-layout/ui";

export default function LoginPage() {
  return (
    <AuthLayout title="Sign in" subtitle="Enter your credentials to continue">
      <LoginForm />
    </AuthLayout>
  );
}
