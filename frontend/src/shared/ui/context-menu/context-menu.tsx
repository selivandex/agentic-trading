/** @format */

"use client";

import type { FC, ReactNode, MouseEvent } from "react";
import { useState, useRef, useId, createContext, useContext } from "react";
import { useContextMenuManager } from "./context-menu-manager";
import type {
  MenuItemProps as AriaMenuItemProps,
  MenuProps as AriaMenuProps,
  PopoverProps as AriaPopoverProps,
  SeparatorProps as AriaSeparatorProps,
} from "react-aria-components";
import {
  Header as AriaHeader,
  Menu as AriaMenu,
  MenuItem as AriaMenuItem,
  MenuSection as AriaMenuSection,
  MenuTrigger as AriaMenuTrigger,
  Popover as AriaPopover,
  Separator as AriaSeparator,
  Button as AriaButton,
} from "react-aria-components";
import { cx } from "@/utils/cx";

// Internal context for closing menu from items
const CloseMenuContext = createContext<(() => void) | null>(null);

// Root component for context menu
interface ContextMenuRootProps {
  children: ReactNode;
  /** Whether the menu is open (controlled) */
  isOpen?: boolean;
  /** Called when the open state changes */
  onOpenChange?: (isOpen: boolean) => void;
}

const ContextMenuRoot = ({
  children,
  isOpen: controlledIsOpen,
  onOpenChange,
}: ContextMenuRootProps) => {
  const [uncontrolledIsOpen, setUncontrolledIsOpen] = useState(false);
  const isControlled = controlledIsOpen !== undefined;
  const isOpen = isControlled ? controlledIsOpen : uncontrolledIsOpen;

  const handleOpenChange = (open: boolean) => {
    if (!isControlled) {
      setUncontrolledIsOpen(open);
    }
    onOpenChange?.(open);
  };

  return (
    <AriaMenuTrigger isOpen={isOpen} onOpenChange={handleOpenChange}>
      {children}
    </AriaMenuTrigger>
  );
};

// Trigger component that opens menu on right click
interface ContextMenuTriggerProps {
  children?: ReactNode;
  /** Additional CSS classes */
  className?: string;
  /** Disable context menu */
  disabled?: boolean;
  /** Accessible label for the trigger area */
  "aria-label"?: string;
}

const ContextMenuTrigger = ({
  children,
  className,
  disabled,
  "aria-label": ariaLabel,
}: ContextMenuTriggerProps) => {
  const buttonRef = useRef<HTMLButtonElement>(null);

  const handleContextMenu = (e: MouseEvent<HTMLDivElement>) => {
    if (disabled) return;

    e.preventDefault();
    e.stopPropagation();

    // Programmatically click the hidden button to open the menu
    buttonRef.current?.click();
  };

  return (
    <>
      {/* Hidden button that MenuTrigger expects */}
      <AriaButton
        ref={buttonRef}
        aria-label={ariaLabel || "Open context menu"}
        className="sr-only"
      />
      {/* Actual trigger area */}
      <div
        onContextMenu={handleContextMenu}
        className={cx("inline-block", className)}
      >
        {children}
      </div>
    </>
  );
};

// Content/Popover component
interface ContextMenuContentProps extends Omit<AriaPopoverProps, "style"> {
  /** Position for fixed positioning (e.g., at cursor) */
  position?: { x: number; y: number };
  /** Additional styles */
  style?: React.CSSProperties;
}

const ContextMenuContent = ({
  position,
  style,
  ...props
}: ContextMenuContentProps) => {
  // When position is provided, use fixed positioning at cursor
  // Otherwise, use React Aria's automatic positioning
  if (position) {
    return (
      <AriaPopover
        {...props}
        // Override React Aria's positioning with inline styles
        style={
          {
            ...style,
            position: "fixed",
            left: `${position.x}px`,
            top: `${position.y}px`,
            margin: 0,
            inset: "auto",
            transform: "none",
            "--trigger-anchor-point": "0 0",
          } as React.CSSProperties
        }
        // Disable React Aria positioning
        shouldFlip={false}
        placement="bottom left"
        offset={0}
        className={(state) =>
          cx(
            "min-w-[12rem] overflow-auto rounded-lg bg-primary shadow-lg ring-1 ring-secondary_alt",
            state.isEntering &&
              "duration-150 ease-out animate-in fade-in zoom-in-95",
            state.isExiting &&
              "duration-100 ease-in animate-out fade-out zoom-out-95",
            typeof props.className === "function"
              ? props.className(state)
              : props.className
          )
        }
      >
        {props.children}
      </AriaPopover>
    );
  }

  // Default behavior with automatic positioning
  return (
    <AriaPopover
      placement="bottom left"
      {...props}
      style={style}
      className={(state) =>
        cx(
          "min-w-[12rem] origin-(--trigger-anchor-point) overflow-auto rounded-lg bg-primary shadow-lg ring-1 ring-secondary_alt will-change-transform",
          state.isEntering &&
            "duration-150 ease-out animate-in fade-in zoom-in-95 placement-right:slide-in-from-left-0.5 placement-top:slide-in-from-bottom-0.5 placement-bottom:slide-in-from-top-0.5 placement-left:slide-in-from-right-0.5",
          state.isExiting &&
            "duration-100 ease-in animate-out fade-out zoom-out-95 placement-right:slide-out-to-left-0.5 placement-top:slide-out-to-bottom-0.5 placement-bottom:slide-out-to-top-0.5 placement-left:slide-out-to-right-0.5",
          typeof props.className === "function"
            ? props.className(state)
            : props.className
        )
      }
    >
      {props.children}
    </AriaPopover>
  );
};

// Menu component
type ContextMenuProps<T extends object> = AriaMenuProps<T>;

const ContextMenuComponent = <T extends object>(props: ContextMenuProps<T>) => {
  return (
    <AriaMenu
      selectionMode="single"
      disallowEmptySelection
      {...props}
      className={(state) =>
        cx(
          "h-min max-h-[calc(100vh-2rem)] overflow-y-auto py-1 outline-hidden select-none",
          typeof props.className === "function"
            ? props.className(state)
            : props.className
        )
      }
    />
  );
};

// Item component
export interface ContextMenuItemProps extends AriaMenuItemProps {
  /** The label of the item to be displayed */
  label?: string;
  /** An addon to be displayed on the right side of the item */
  addon?: string;
  /** An icon to be displayed on the left side of the item */
  icon?: FC<{ className?: string }>;
  /** If true, the item will have destructive styling */
  destructive?: boolean;
  /** If true, the item will not have any styles */
  unstyled?: boolean;
}

const ContextMenuItem = ({
  label,
  children,
  addon,
  icon: Icon,
  destructive,
  unstyled,
  onAction,
  ...props
}: ContextMenuItemProps) => {
  // Get close function from context (if available)
  const closeMenu = useContext(CloseMenuContext);

  // Wrap onAction to close menu automatically
  const handleAction = () => {
    onAction?.();
    // Close menu if closeMenu function is available (from PositionWrapper)
    closeMenu?.();
  };

  if (unstyled) {
    return (
      <AriaMenuItem
        id={label}
        textValue={label}
        {...props}
        onAction={handleAction}
      >
        {children}
      </AriaMenuItem>
    );
  }

  return (
    <AriaMenuItem
      {...props}
      id={label}
      textValue={label}
      onAction={handleAction}
      className={(state) =>
        cx(
          "group block cursor-pointer px-1.5 py-px outline-hidden",
          state.isDisabled && "cursor-not-allowed",
          typeof props.className === "function"
            ? props.className(state)
            : props.className
        )
      }
    >
      {(state) => (
        <div
          className={cx(
            "relative flex items-center rounded-md px-2.5 py-2 outline-focus-ring transition duration-100 ease-linear",
            !state.isDisabled && !destructive && "group-hover:bg-primary_hover",
            !state.isDisabled &&
              destructive &&
              "group-hover:bg-destructive/10 dark:group-hover:bg-destructive/20",
            state.isFocused && !destructive && "bg-primary_hover",
            state.isFocused &&
              destructive &&
              "bg-destructive/10 dark:bg-destructive/20",
            state.isFocusVisible && "outline-2 -outline-offset-2"
          )}
        >
          {Icon && (
            <Icon
              aria-hidden="true"
              className={cx(
                "mr-2 size-4 shrink-0 stroke-[2.25px]",
                state.isDisabled && "text-fg-disabled",
                !state.isDisabled && !destructive && "text-fg-quaternary",
                !state.isDisabled && destructive && "text-fg-error-primary"
              )}
            />
          )}

          <span
            className={cx(
              "grow truncate text-sm font-semibold",
              state.isDisabled && "text-disabled",
              !state.isDisabled && !destructive && "text-secondary",
              !state.isDisabled &&
                !destructive &&
                state.isFocused &&
                "text-secondary_hover",
              !state.isDisabled && destructive && "text-error-primary"
            )}
          >
            {label ||
              (typeof children === "function" ? children(state) : children)}
          </span>

          {addon && (
            <span
              className={cx(
                "ml-3 shrink-0 rounded px-1 py-px text-xs font-medium ring-1 ring-secondary ring-inset",
                state.isDisabled && "text-disabled",
                !state.isDisabled && "text-quaternary"
              )}
            >
              {addon}
            </span>
          )}
        </div>
      )}
    </AriaMenuItem>
  );
};

// Separator component
const ContextMenuSeparator = (props: AriaSeparatorProps) => {
  return (
    <AriaSeparator
      {...props}
      className={cx("my-1 h-px w-full bg-border-secondary", props.className)}
    />
  );
};

// Checkbox item component
export interface ContextMenuCheckboxItemProps extends AriaMenuItemProps {
  /** The label of the checkbox item */
  label?: string;
  /** Whether the checkbox is checked */
  checked?: boolean;
  /** Icon to show when checked */
  icon?: FC<{ className?: string }>;
  /** Callback when item is selected */
  onAction?: () => void;
}

const ContextMenuCheckboxItem = ({
  label,
  children,
  checked,
  icon: Icon,
  onAction,
  ...props
}: ContextMenuCheckboxItemProps) => {
  // Get close function from context (if available)
  const closeMenu = useContext(CloseMenuContext);

  // Wrap onAction to close menu automatically
  const handleAction = () => {
    onAction?.();
    // Close menu if closeMenu function is available (from PositionWrapper)
    closeMenu?.();
  };

  return (
    <AriaMenuItem
      {...props}
      id={label}
      textValue={label}
      onAction={handleAction}
      className={(state) =>
        cx(
          "group block cursor-pointer px-1.5 py-px outline-hidden",
          state.isDisabled && "cursor-not-allowed",
          typeof props.className === "function"
            ? props.className(state)
            : props.className
        )
      }
    >
      {(state) => (
        <div
          className={cx(
            "relative flex items-center rounded-md px-2.5 py-2 pl-8 outline-focus-ring transition duration-100 ease-linear",
            !state.isDisabled && "group-hover:bg-primary_hover",
            state.isFocused && "bg-primary_hover",
            state.isFocusVisible && "outline-2 -outline-offset-2"
          )}
        >
          <span className="pointer-events-none absolute left-3.5 flex size-4 items-center justify-center">
            {checked && Icon && (
              <Icon
                aria-hidden="true"
                className={cx(
                  "size-4 stroke-[2.25px]",
                  state.isDisabled
                    ? "text-fg-disabled"
                    : "text-fg-brand-primary"
                )}
              />
            )}
          </span>

          <span
            className={cx(
              "grow truncate text-sm font-semibold",
              state.isDisabled ? "text-disabled" : "text-secondary",
              state.isFocused && "text-secondary_hover"
            )}
          >
            {label ||
              (typeof children === "function" ? children(state) : children)}
          </span>
        </div>
      )}
    </AriaMenuItem>
  );
};

// Label component for sections
export interface ContextMenuLabelProps {
  children: ReactNode;
  className?: string;
}

const ContextMenuLabel = ({ children, className }: ContextMenuLabelProps) => {
  return (
    <AriaHeader
      className={cx(
        "px-4 py-1.5 text-xs font-semibold text-quaternary",
        className
      )}
    >
      {children}
    </AriaHeader>
  );
};

// Position-based Context Menu Wrapper (for right-click menus with coordinate positioning)
interface PositionContextMenuWrapperProps {
  children: ReactNode;
  /** Menu content to render */
  menu: ReactNode;
  /** Whether the context menu is disabled */
  disabled?: boolean;
  /** Wrapper className for trigger */
  wrapperClassName?: string;
  /** Custom validator before opening menu - return false to prevent opening */
  onBeforeOpen?: (e: MouseEvent) => boolean;
}

const PositionContextMenuWrapper = ({
  children,
  menu,
  disabled = false,
  wrapperClassName = "inline-flex",
  onBeforeOpen,
}: PositionContextMenuWrapperProps) => {
  const menuId = useId();
  const { openMenu, closeMenu } = useContextMenuManager();

  if (disabled) {
    return <>{children}</>;
  }

  const handleContextMenu = (e: MouseEvent) => {
    // Custom validation before opening
    if (onBeforeOpen && !onBeforeOpen(e)) {
      return; // Don't open menu if validation fails
    }

    e.preventDefault();
    e.stopPropagation();

    // Get position and open menu (or reposition if already open)
    const position = { x: e.clientX, y: e.clientY };

    // Wrap menu with CloseMenuContext provider
    const menuWithContext = (
      <CloseMenuContext.Provider value={closeMenu}>
        {menu}
      </CloseMenuContext.Provider>
    );

    // Always call openMenu - it will update position if same menu, or switch if different
    openMenu(menuId, position, menuWithContext);
  };

  return (
    <div
      onContextMenu={handleContextMenu}
      className={wrapperClassName}
      role="presentation"
    >
      {children}
    </div>
  );
};

// Export as compound component
export const ContextMenu = Object.assign(ContextMenuRoot, {
  Trigger: ContextMenuTrigger,
  Content: ContextMenuContent,
  Menu: ContextMenuComponent,
  Item: ContextMenuItem,
  CheckboxItem: ContextMenuCheckboxItem,
  Separator: ContextMenuSeparator,
  Label: ContextMenuLabel,
  Section: AriaMenuSection,
  PositionWrapper: PositionContextMenuWrapper,
});
