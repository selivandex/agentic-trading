/** @format */

"use client";

import type { DateInputProps as AriaDateInputProps } from "react-aria-components";
import {
  DateInput as AriaDateInput,
  DateSegment as AriaDateSegment,
} from "react-aria-components";
import { cx } from "@/utils/cx";

type DateTimeInputProps = Omit<AriaDateInputProps, "children">;

/**
 * Date Time Input Component
 *
 * Same as DateInput but supports time segments (hours and minutes)
 * Renders format: MM / DD / YYYY HH : MM
 */
export const DateTimeInput = (props: DateTimeInputProps) => {
  return (
    <AriaDateInput
      {...props}
      className={cx(
        "flex rounded-lg bg-primary px-2.5 py-2 text-md shadow-xs ring-1 ring-primary ring-inset focus-within:ring-2 focus-within:ring-brand",
        typeof props.className === "string" && props.className
      )}
    >
      {(segment) => (
        <AriaDateSegment
          segment={segment}
          className={cx(
            "rounded px-0.5 text-primary tabular-nums caret-transparent focus:bg-brand-solid focus:font-medium focus:text-white focus:outline-hidden",
            // The placeholder segment.
            segment.isPlaceholder && "text-placeholder uppercase",
            // The separator "/" or ":" segment.
            segment.type === "literal" && "text-fg-quaternary"
          )}
        />
      )}
    </AriaDateInput>
  );
};
