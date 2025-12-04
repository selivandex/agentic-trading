/** @format */

/**
 * Authenticated fetch wrapper
 * Automatically includes credentials (cookies)
 */

export type FetchOptions = RequestInit;

/**
 * Fetch wrapper that includes credentials
 */
export async function fetchWithAuth(
  url: string,
  options: FetchOptions = {}
): Promise<Response> {
  const { headers, ...restOptions } = options;

  const requestHeaders: Record<string, string> = {
    "Content-Type": "application/json",
    ...(headers as Record<string, string>),
  };

  return fetch(url, {
    ...restOptions,
    headers: requestHeaders,
    credentials: "include", // CRITICAL: Include cookies for session auth
  });
}

