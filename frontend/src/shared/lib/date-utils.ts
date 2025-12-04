/** @format */

import type {
  CalendarDate,
  CalendarDateTime,
  ZonedDateTime,
} from "@internationalized/date";
import { toZoned } from "@internationalized/date";

/**
 * Date utility functions
 *
 * Helpers for working with dates and ISO 8601 format
 */

/**
 * Converts various date/time representations to UTC ISO 8601 string
 *
 * @param value - Date value in various formats:
 *   - CalendarDate from @internationalized/date (date only, no time)
 *   - CalendarDateTime from @internationalized/date (date + time, no timezone)
 *   - ZonedDateTime from @internationalized/date (date + time + timezone)
 *   - ISO 8601 string
 *   - null/undefined
 * @returns ISO 8601 UTC string (ending with Z) or null if input is null/undefined
 *
 * @example
 * ```ts
 * // From CalendarDateTime
 * const dt = parseDateTime("2025-11-27T11:48:46");
 * toUTCISO(dt) // Returns: "2025-11-27T11:48:46.000Z"
 *
 * // From ZonedDateTime (auto-converts to UTC)
 * const zdt = now(getLocalTimeZone()); // 2025-11-27T18:48:46+07:00[Asia/Bangkok]
 * toUTCISO(zdt) // Returns: "2025-11-27T11:48:46.000Z"
 *
 * // From string
 * toUTCISO("2025-11-27T11:48:46+07:00") // Returns: "2025-11-27T04:48:46.000Z"
 * ```
 */
export const toUTCISO = (
  value:
    | CalendarDate
    | CalendarDateTime
    | ZonedDateTime
    | string
    | null
    | undefined
): string | null => {
  if (!value) return null;

  // If it's a string, parse with native Date API
  if (typeof value === "string") {
    const date = new Date(value);
    return isNaN(date.getTime()) ? null : date.toISOString();
  }

  // If it's ZonedDateTime, convert to UTC zone first
  if ("timeZone" in value) {
    const utcZoned = toZoned(value, "UTC");
    return utcZoned.toDate().toISOString();
  }

  // If it's CalendarDateTime, treat as UTC and convert
  const utcZoned = toZoned(value, "UTC");
  return utcZoned.toDate().toISOString();
};

/**
 * Gets current date/time as UTC ISO 8601 string
 *
 * @returns Current date/time in ISO 8601 UTC format
 *
 * @example
 * ```ts
 * nowUTCISO()
 * // Returns: "2025-11-27T04:48:46.455Z"
 * ```
 */
export const nowUTCISO = (): string => {
  return new Date().toISOString();
};
