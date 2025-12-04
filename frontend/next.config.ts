import type { NextConfig } from "next";

// Load environment variables in development mode
// Next.js loads .env files automatically, but we can use dotenv for additional control
if (process.env.NODE_ENV === "development") {
  try {
    // eslint-disable-next-line @typescript-eslint/no-require-imports
    const dotenv = require("dotenv");
    dotenv.config({ path: ".env.local" });
  } catch {
    // dotenv is optional, Next.js will handle .env loading
  }
}

const nextConfig: NextConfig = {
  /* config options here */

  // Sentry source maps upload (optional)
  // Enabled if SENTRY_AUTH_TOKEN is set
  ...(process.env.SENTRY_AUTH_TOKEN && {
    productionBrowserSourceMaps: true,
  }),
};

export default nextConfig;
