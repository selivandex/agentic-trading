/**
 * Locales export
 */

import { ru } from "./ru";
import { en } from "./en";

export const locales = {
  ru,
  en,
} as const;

export type Locale = keyof typeof locales;
export type { Translations } from "./ru";

// Default locale
export const DEFAULT_LOCALE: Locale = "ru";
