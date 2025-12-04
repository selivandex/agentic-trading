/** @format */

import { redirect } from "next/navigation";
import { auth } from "@/shared/lib/auth";

/**
 * Strategies Page
 * 
 * List and manage trading strategies
 */
export default async function StrategiesPage() {
  const session = await auth();
  
  if (!session?.user) {
    redirect("/login");
  }
  
  return (
    <div className="min-h-screen bg-primary p-8">
      <div className="max-w-7xl mx-auto">
        <div className="flex items-center justify-between mb-8">
          <h1 className="text-display-lg font-semibold text-primary">
            Trading Strategies
          </h1>
        </div>
        
        <div className="bg-primary border border-primary rounded-lg p-12 text-center">
          <p className="text-secondary">
            No strategies yet. Create your first strategy to get started.
          </p>
        </div>
      </div>
    </div>
  );
}

