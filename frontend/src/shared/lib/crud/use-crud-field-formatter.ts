/** @format */

"use client";

/**
 * Hook for formatting field values for display
 */
export function useCrudFieldFormatter() {
  const formatFieldValue = (value: unknown, fieldType: string): string => {
    if (value == null) {
      return "—";
    }

    switch (fieldType) {
      case "checkbox":
        return value ? "Yes" : "No";

      case "date":
        if (typeof value === "string" || value instanceof Date) {
          return new Date(value).toLocaleDateString();
        }
        return String(value);

      case "datetime":
        if (typeof value === "string" || value instanceof Date) {
          return new Date(value).toLocaleString();
        }
        return String(value);

      case "number":
        if (typeof value === "number") {
          return value.toLocaleString();
        }
        return String(value);

      default:
        if (typeof value === "object") {
          return JSON.stringify(value, null, 2);
        }
        return String(value);
    }
  };

  return { formatFieldValue };
}

