/** @format */

"use client";

// import { Slider } from "@/shared/base/slider";
import { Input } from "@/components/base/input/input";
import type { FilterComponentProps } from "./types";

/**
 * Number range filter component
 */
export function NumberRangeFilter({
  filter,
  value,
  onChange,
}: FilterComponentProps) {
  // Parse min/max from value or use defaults from filter metadata
  const rangeValue = value as [number, number] | null | undefined;
  const minValue = filter.min ?? 0;
  const maxValue = filter.max ?? 100;

  const handleMinChange = (newMin: string) => {
    const min = newMin ? Number(newMin) : null;
    const max = rangeValue?.[1] ?? null;
    // Only update if at least one value is set
    if (min !== null || max !== null) {
      onChange(filter.id, [min ?? minValue, max ?? maxValue]);
    } else {
      onChange(filter.id, null);
    }
  };

  const handleMaxChange = (newMax: string) => {
    const max = newMax ? Number(newMax) : null;
    const min = rangeValue?.[0] ?? null;
    // Only update if at least one value is set
    if (min !== null || max !== null) {
      onChange(filter.id, [min ?? minValue, max ?? maxValue]);
    } else {
      onChange(filter.id, null);
    }
  };

  return (
    <div className="space-y-2">
      <label className="block text-sm font-medium text-secondary">
        {filter.name}
      </label>
      <div className="flex gap-2 items-center">
        <Input
          type="number"
          placeholder="От"
          value={rangeValue?.[0] !== undefined ? String(rangeValue[0]) : ""}
          onChange={handleMinChange}
          min={minValue}
          max={maxValue}
        />
        <span className="text-sm text-secondary">—</span>
        <Input
          type="number"
          placeholder="До"
          value={rangeValue?.[1] !== undefined ? String(rangeValue[1]) : ""}
          onChange={handleMaxChange}
          min={minValue}
          max={maxValue}
        />
      </div>
      {/* Slider implementation - commented out as per request */}
      {/* <div className="pt-2 pb-6">
        <Slider
          value={currentValues}
          onChange={(newValues) => onChange(filter.id, newValues)}
          minValue={minValue}
          maxValue={maxValue}
          labelPosition="bottom"
          labelFormatter={(val) => String(val)}
          formatOptions={{ style: "decimal" }}
        />
      </div> */}
    </div>
  );
}
