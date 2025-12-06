/** @format */

"use client";

import { useMemo } from "react";
import { locales, DEFAULT_LOCALE, type Locale, type Translations } from "@/shared/config/locales";

/**
 * Simple i18n hook for translations
 *
 * Returns translation function and current locale.
 * Supports nested key access with dot notation.
 *
 * @param locale - Current locale (defaults to DEFAULT_LOCALE)
 *
 * @example
 * ```tsx
 * const { t } = useTranslation();
 * return <div>{t("auth.errors.invalidCredentials")}</div>;
 * ```
 *
 * @example With parameters
 * ```tsx
 * const { t } = useTranslation();
 * return <div>{t("validation.minLength", { min: 8 })}</div>;
 * ```
 */
export function useTranslation(locale: Locale = DEFAULT_LOCALE) {
  const translations = locales[locale];

  const t = useMemo(() => {
    return (key: string, params?: Record<string, string | number>): string => {
      // Navigate through nested object using dot notation
      const keys = key.split(".");
      let value: unknown = translations;

      for (const k of keys) {
        if (value && typeof value === "object" && k in value) {
          value = (value as Record<string, unknown>)[k];
        } else {
          // Key not found, return the key itself as fallback
          return key;
        }
      }

      // If value is not a string, return key as fallback
      if (typeof value !== "string") {
        return key;
      }

      // Replace parameters in the string
      if (params) {
        return Object.entries(params).reduce((acc, [paramKey, paramValue]) => {
          return acc.replace(new RegExp(`\\{${paramKey}\\}`, "g"), String(paramValue));
        }, value);
      }

      return value;
    };
  }, [translations]);

  return {
    t,
    locale,
    translations,
  };
}

/**
 * Get nested value from translations object by key path
 *
 * Type-safe helper for accessing translations outside of React components
 *
 * @param translations - Translations object
 * @param key - Dot-notation key path
 * @returns Translation string or key if not found
 *
 * @example
 * ```ts
 * import { ru } from "@/shared/config/locales/ru";
 * const message = getTranslation(ru, "auth.errors.invalidCredentials");
 * ```
 */
export function getTranslation(
  translations: Translations,
  key: string,
  params?: Record<string, string | number>
): string {
  const keys = key.split(".");
  let value: unknown = translations;

  for (const k of keys) {
    if (value && typeof value === "object" && k in value) {
      value = (value as Record<string, unknown>)[k];
    } else {
      return key;
    }
  }

  if (typeof value !== "string") {
    return key;
  }

  if (params) {
    return Object.entries(params).reduce((acc, [paramKey, paramValue]) => {
      return acc.replace(new RegExp(`\\{${paramKey}\\}`, "g"), String(paramValue));
    }, value);
  }

  return value;
}
