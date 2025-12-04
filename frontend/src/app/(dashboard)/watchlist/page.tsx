/** @format */

import { redirect } from "next/navigation";
import { auth } from "@/shared/lib/auth";

/**
 * Watchlist Page
 * 
 * Manage fund watchlist
 */
export default async function WatchlistPage() {
  const session = await auth();
  
  if (!session?.user) {
    redirect("/login");
  }
  
  return (
    <div className="min-h-screen bg-primary p-8">
      <div className="max-w-7xl mx-auto">
        <div className="flex items-center justify-between mb-8">
          <h1 className="text-display-lg font-semibold text-primary">
            Fund Watchlist
          </h1>
        </div>
        
        <div className="bg-primary border border-primary rounded-lg p-12 text-center">
          <p className="text-secondary">
            No symbols in watchlist. Add symbols to start monitoring.
          </p>
        </div>
      </div>
    </div>
  );
}

