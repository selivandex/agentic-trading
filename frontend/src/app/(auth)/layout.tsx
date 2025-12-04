import { ReactNode } from "react";

// Auth layout group - all auth pages share this layout
export default function AuthLayoutGroup({
  children,
}: {
  children: ReactNode;
}) {
  return <>{children}</>;
}
