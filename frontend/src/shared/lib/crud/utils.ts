/** @format */

/**
 * Get nested object value by path
 * Lightweight alternative to lodash.get
 */
export function get<T = unknown>(
  obj: unknown,
  path: string,
  defaultValue?: T,
): T {
  const keys = path.split(".");
  let result: unknown = obj;

  for (const key of keys) {
    if (result == null || typeof result !== "object") {
      return defaultValue as T;
    }
    result = (result as Record<string, unknown>)[key];
  }

  return (result ?? defaultValue) as T;
}
