/** @format */

import { redirect } from "next/navigation";
import { auth } from "@/shared/lib/auth";

/**
 * Dashboard Page
 * 
 * Main trading dashboard
 */
export default async function DashboardPage() {
  const session = await auth();
  
  if (!session?.user) {
    redirect("/login");
  }
  
  return (
    <div className="min-h-screen bg-primary p-8">
      <div className="max-w-7xl mx-auto">
        <h1 className="text-display-lg font-semibold text-primary mb-8">
          Trading Dashboard
        </h1>
        
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          <div className="p-6 bg-primary border border-primary rounded-lg">
            <h2 className="text-lg font-semibold text-primary mb-2">
              Welcome, {session.user.firstName || session.user.email}!
            </h2>
            <p className="text-secondary">
              Your AI-powered trading platform is ready.
            </p>
          </div>
          
          <div className="p-6 bg-primary border border-primary rounded-lg">
            <h3 className="text-sm font-medium text-tertiary mb-1">
              Active Strategies
            </h3>
            <p className="text-display-md font-semibold text-primary">
              0
            </p>
          </div>
          
          <div className="p-6 bg-primary border border-primary rounded-lg">
            <h3 className="text-sm font-medium text-tertiary mb-1">
              Total P&L
            </h3>
            <p className="text-display-md font-semibold text-success">
              $0.00
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}

