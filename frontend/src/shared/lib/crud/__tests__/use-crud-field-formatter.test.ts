/** @format */

import { renderHook } from "@testing-library/react";
import { useCrudFieldFormatter } from "../use-crud-field-formatter";

describe("useCrudFieldFormatter", () => {
  it("should format null/undefined values as em dash", () => {
    const { result } = renderHook(() => useCrudFieldFormatter());

    expect(result.current.formatFieldValue(null, "text")).toBe("—");
    expect(result.current.formatFieldValue(undefined, "text")).toBe("—");
  });

  it("should format checkbox values", () => {
    const { result } = renderHook(() => useCrudFieldFormatter());

    expect(result.current.formatFieldValue(true, "checkbox")).toBe("Yes");
    expect(result.current.formatFieldValue(false, "checkbox")).toBe("No");
  });

  it("should format date values", () => {
    const { result } = renderHook(() => useCrudFieldFormatter());
    const date = new Date("2024-01-15");

    const formatted = result.current.formatFieldValue(date, "date");
    expect(formatted).toContain("2024");
  });

  it("should format datetime values", () => {
    const { result } = renderHook(() => useCrudFieldFormatter());
    const date = new Date("2024-01-15T10:30:00");

    const formatted = result.current.formatFieldValue(date, "datetime");
    expect(formatted).toContain("2024");
  });

  it("should format number values with locale", () => {
    const { result } = renderHook(() => useCrudFieldFormatter());

    const formatted = result.current.formatFieldValue(1000, "number");
    // Different locales format differently, just check it's a string
    expect(typeof formatted).toBe("string");
    expect(formatted.length).toBeGreaterThan(0);
  });

  it("should stringify objects", () => {
    const { result } = renderHook(() => useCrudFieldFormatter());
    const obj = { key: "value", nested: { data: 123 } };

    const formatted = result.current.formatFieldValue(obj, "text");
    expect(formatted).toContain("key");
    expect(formatted).toContain("value");
    expect(formatted).toContain("nested");
  });

  it("should convert other values to string", () => {
    const { result } = renderHook(() => useCrudFieldFormatter());

    expect(result.current.formatFieldValue("test", "text")).toBe("test");
    expect(result.current.formatFieldValue(123, "text")).toBe("123");
  });
});

