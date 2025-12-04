/** @format */

"use client";

import type { HTMLAttributes, PropsWithChildren } from "react";
import { Fragment, useContext, useState } from "react";
import { getLocalTimeZone, now, toCalendarDate } from "@internationalized/date";
import type { ZonedDateTime } from "@internationalized/date";
import { ChevronLeft, ChevronRight } from "@untitledui/icons";
import type {
  CalendarProps as AriaCalendarProps,
  DateValue,
} from "react-aria-components";
import {
  Calendar as AriaCalendar,
  CalendarContext as AriaCalendarContext,
  CalendarGrid as AriaCalendarGrid,
  CalendarGridBody as AriaCalendarGridBody,
  CalendarGridHeader as AriaCalendarGridHeader,
  CalendarHeaderCell as AriaCalendarHeaderCell,
  CalendarStateContext as AriaCalendarStateContext,
  DatePickerStateContext,
  Heading as AriaHeading,
  useSlottedContext,
} from "react-aria-components";
import { Button } from "@/components/base/buttons/button";
import { cx } from "@/utils/cx";
import { CalendarCell } from "./cell";
import { DateTimeInput } from "./date-time-input";

export const DateTimeCalendarContextProvider = ({
  children,
}: PropsWithChildren) => {
  const [value, onChange] = useState<DateValue | null>(null);
  const [focusedValue, onFocusChange] = useState<DateValue | undefined>();

  return (
    <AriaCalendarContext.Provider
      value={{ value, onChange, focusedValue, onFocusChange }}
    >
      {children}
    </AriaCalendarContext.Provider>
  );
};

const NowPresetButton = ({
  value,
  children,
  ...props
}: HTMLAttributes<HTMLButtonElement> & { value: ZonedDateTime }) => {
  // Use DatePickerStateContext instead of CalendarStateContext
  // to properly set both date and time
  const datePickerContext = useContext(DatePickerStateContext);
  const calendarContext = useContext(AriaCalendarStateContext);

  const handleClick = (e: React.MouseEvent<HTMLButtonElement>) => {
    e.preventDefault();
    e.stopPropagation();

    // If we're inside a DatePicker, use its context (includes time)
    if (datePickerContext) {
      datePickerContext.setValue(value as never);
    } else if (calendarContext) {
      // Fallback to calendar context if not in DatePicker
      calendarContext.setValue(value as never);
      calendarContext.setFocusedDate(toCalendarDate(value));
    }
  };

  return (
    <Button
      {...props}
      // It's important to give `null` explicitly to the `slot` prop
      // otherwise the button will throw an error due to not using one of
      // the required slots inside the Calendar component.
      // Passing `null` will tell the button to not use a slot context.
      slot={null}
      size="md"
      color="secondary"
      onClick={handleClick}
    >
      {children}
    </Button>
  );
};

interface DateTimeCalendarProps extends AriaCalendarProps<DateValue> {
  /** The dates to highlight. */
  highlightedDates?: DateValue[];
}

export const DateTimeCalendar = ({
  highlightedDates,
  className,
  ...props
}: DateTimeCalendarProps) => {
  const context = useSlottedContext(AriaCalendarContext)!;
  const timeZone = getLocalTimeZone();

  const ContextWrapper = context ? Fragment : DateTimeCalendarContextProvider;

  return (
    <ContextWrapper>
      <AriaCalendar
        {...props}
        className={(state) =>
          cx(
            "flex flex-col gap-3",
            typeof className === "function" ? className(state) : className
          )
        }
      >
        <header className="flex items-center justify-between">
          <Button
            slot="previous"
            iconLeading={ChevronLeft}
            size="sm"
            color="tertiary"
            className="size-8"
          />
          <AriaHeading className="text-sm font-semibold text-fg-secondary" />
          <Button
            slot="next"
            iconLeading={ChevronRight}
            size="sm"
            color="tertiary"
            className="size-8"
          />
        </header>

        <div className="flex gap-3">
          <DateTimeInput className="flex-1" />
          <NowPresetButton value={now(timeZone)}>Now</NowPresetButton>
        </div>

        <AriaCalendarGrid weekdayStyle="short" className="w-full">
          <AriaCalendarGridHeader className="border-b-4 border-transparent">
            {(day) => (
              <AriaCalendarHeaderCell className="p-0">
                <div className="flex size-10 items-center justify-center text-sm font-medium text-secondary">
                  {day.slice(0, 2)}
                </div>
              </AriaCalendarHeaderCell>
            )}
          </AriaCalendarGridHeader>
          <AriaCalendarGridBody className="[&_td]:p-0 [&_tr]:border-b-4 [&_tr]:border-transparent [&_tr:last-of-type]:border-none">
            {(date) => (
              <CalendarCell
                date={date}
                isHighlighted={highlightedDates?.some(
                  (highlightedDate) => date.compare(highlightedDate) === 0
                )}
              />
            )}
          </AriaCalendarGridBody>
        </AriaCalendarGrid>
      </AriaCalendar>
    </ContextWrapper>
  );
};
