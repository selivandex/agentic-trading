/** @format */

"use client";

import type { ReactNode, Ref } from "react";
import { HelpCircle } from "@untitledui/icons";
import type { LabelProps as AriaLabelProps } from "react-aria-components";
import { Label as AriaLabel } from "react-aria-components";
import { Tooltip, TooltipTrigger } from "@/components/base/tooltip/tooltip";
import { cx } from "@/utils/cx";

export interface LabelProps extends AriaLabelProps {
  children: ReactNode;
  isRequired?: boolean;
  tooltip?: string;
  tooltipDescription?: string;
  /** Action component to display on the right side of the label */
  labelAction?: ReactNode;
  ref?: Ref<HTMLLabelElement>;
}

export const Label = ({
  isRequired,
  tooltip,
  tooltipDescription,
  labelAction,
  className,
  ...props
}: LabelProps) => {
  return (
    <div className="flex w-full items-center justify-between gap-2">
      <AriaLabel
        // Used for conditionally hiding/showing the label element via CSS:
        // <Input label="Visible only on mobile" className="lg:**:data-label:hidden" />
        // or
        // <Input label="Visible only on mobile" className="lg:label:hidden" />
        data-label="true"
        {...props}
        className={cx(
          "flex cursor-default items-center gap-0.5 text-sm font-medium text-secondary",
          className
        )}
      >
        {props.children}

        <span
          className={cx(
            "hidden text-brand-tertiary",
            isRequired && "block",
            typeof isRequired === "undefined" && "group-required:block"
          )}
        >
          *
        </span>

        {tooltip && (
          <Tooltip
            title={tooltip}
            description={tooltipDescription}
            placement="top"
          >
            <TooltipTrigger
              // `TooltipTrigger` inherits the disabled state from the parent form field
              // but we don't that. We want the tooltip be enabled even if the parent
              // field is disabled.
              isDisabled={false}
              className="cursor-pointer text-fg-quaternary transition duration-200 hover:text-fg-quaternary_hover focus:text-fg-quaternary_hover"
            >
              <HelpCircle className="size-4" />
            </TooltipTrigger>
          </Tooltip>
        )}
      </AriaLabel>

      {labelAction && <div className="shrink-0">{labelAction}</div>}
    </div>
  );
};

Label.displayName = "Label";
