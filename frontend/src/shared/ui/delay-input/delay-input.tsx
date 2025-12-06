/** @format */

"use client";

import { Input } from "@/components/base/input/input";
import { Select } from "@/components/base/select/select";

type ScenarioDelayUnit = "MINUTES" | "HOURS" | "DAYS";

interface DelayInputProps {
  label: string;
  value: number;
  unit: ScenarioDelayUnit;
  onValueChange: (value: number) => void;
  onUnitChange: (unit: ScenarioDelayUnit) => void;
  tooltip?: string;
  isRequired?: boolean;
  isDisabled?: boolean;
  className?: string;
}

const DELAY_UNIT_OPTIONS: Array<{ id: ScenarioDelayUnit; label: string }> = [
  { id: "MINUTES" as ScenarioDelayUnit, label: "Minutes" },
  { id: "HOURS" as ScenarioDelayUnit, label: "Hours" },
  { id: "DAYS" as ScenarioDelayUnit, label: "Days" },
];

/**
 * Delay Input Component
 *
 * Input with unit selector for delay configuration
 * Allows entering integer > 0 with unit selection (minutes/hours/days)
 */
export const DelayInput = ({
  label,
  value,
  unit,
  onValueChange,
  onUnitChange,
  tooltip,
  isRequired = false,
  isDisabled = false,
  className,
}: DelayInputProps) => {
  const handleValueChange = (newValue: string | number) => {
    const numValue = parseInt(newValue as string, 10);
    // Ensure value is > 0
    if (!isNaN(numValue) && numValue > 0) {
      onValueChange(numValue);
    } else if (newValue === "" || newValue === "0") {
      onValueChange(1); // Default to 1 if empty or 0
    }
  };

  return (
    <div className={className}>
      <div className="space-y-1.5">
        {/* Label with tooltip */}
        <div className="flex items-center gap-1">
          <label className="text-sm font-medium text-primary">
            {label}
            {isRequired && <span className="text-error"> *</span>}
          </label>
        </div>

        {/* Input + Select in one row */}
        <div className="flex gap-2 max-w-md">
          <Input
            type="number"
            value={value.toString()}
            onChange={handleValueChange}
            isRequired={isRequired}
            isDisabled={isDisabled}
            className="flex-1"
            aria-label={`${label} value`}
          />
          <Select
            items={DELAY_UNIT_OPTIONS}
            selectedKey={unit}
            onSelectionChange={(key) => onUnitChange(key as ScenarioDelayUnit)}
            isDisabled={isDisabled}
            aria-label="Time unit"
            className="w-32"
          >
            {(item) => <Select.Item id={item.id}>{item.label}</Select.Item>}
          </Select>
        </div>

        {/* Tooltip hint */}
        {tooltip && <p className="text-sm text-secondary">{tooltip}</p>}
      </div>
    </div>
  );
};
