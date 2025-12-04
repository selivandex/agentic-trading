import { SignUpForm } from "@/features/auth/ui";
import { AuthLayout } from "@/widgets/auth-layout/ui";

export default function SignUpPage() {
  return (
    <AuthLayout 
      title="Create account" 
      subtitle="Get started with Flowly"
    >
      <SignUpForm />
    </AuthLayout>
  );
}
