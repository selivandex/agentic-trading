/**
 * API route to clear last context cookies
 * Called during logout
 */

import { NextResponse } from "next/server";
import { cookies } from "next/headers";

const LAST_ORG_COOKIE = "flowly_last_org_id";
const LAST_PROJECT_COOKIE = "flowly_last_project_id";

export async function POST() {
  try {
    const cookieStore = await cookies();
    
    // Delete last context cookies
    cookieStore.delete(LAST_ORG_COOKIE);
    cookieStore.delete(LAST_PROJECT_COOKIE);
    
    return NextResponse.json({ success: true });
  } catch (error) {
    console.error("Failed to clear context cookies:", error);
    return NextResponse.json(
      { success: false, error: "Failed to clear context" },
      { status: 500 }
    );
  }
}



