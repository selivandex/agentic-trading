/** @format */

"use client";

import type { FC, HTMLAttributes } from "react";
import { useCallback, useEffect, useRef } from "react";
import { useRouter } from "next/navigation";
import type { Placement } from "@react-types/overlays";
import {
  BookOpen01,
  ChevronSelectorVertical,
  LogOut01,
  Settings01,
  User01,
} from "@untitledui/icons";
import { useFocusManager } from "react-aria";
import type { DialogProps as AriaDialogProps } from "react-aria-components";
import {
  Button as AriaButton,
  Dialog as AriaDialog,
  DialogTrigger as AriaDialogTrigger,
  Popover as AriaPopover,
  OverlayTriggerStateContext,
} from "react-aria-components";
import { AvatarLabelGroup } from "@/components/base/avatar/avatar-label-group";
import { RadioButtonBase } from "@/components/base/radio-buttons/radio-buttons";
import { useBreakpoint } from "@/hooks/use-breakpoint";
import { cx } from "@/utils/cx";

type OrganizationType = {
  /** Unique identifier for the organization */
  id: string;
  /** Organization name */
  name: string;
  /** Organization slug (used in URLs) */
  slug: string;
  /** Organization status */
  status?: string;
  __typename?: string;
};

type MembershipType = {
  /** Unique identifier for the membership */
  id: string;
  /** Whether membership is active */
  active: boolean;
  /** Organization for this membership */
  organization: OrganizationType;
  /** Role information */
  role: {
    id: string;
    name: string;
    key?: string;
    __typename?: string;
  };
  __typename?: string;
};

type MembershipEdge = {
  node?: MembershipType | null;
  __typename?: string;
};

type MembershipConnection = {
  edges?: Array<MembershipEdge | null> | null;
  __typename?: string;
};

export const NavAccountMenu = ({
  className,
  memberships,
  currentOrganizationId,
  currentProjectId,
  onSignOut,
  onOrganizationSwitch,
  ...dialogProps
}: AriaDialogProps & {
  className?: string;
  memberships?: MembershipConnection | null;
  currentOrganizationId: string;
  currentProjectId: string;
  onSignOut?: () => Promise<void> | void;
  onOrganizationSwitch?: (organizationId: string, projectId: string) => void;
}) => {
  const focusManager = useFocusManager();
  const dialogRef = useRef<HTMLDivElement>(null);

  const onKeyDown = useCallback(
    (e: KeyboardEvent) => {
      switch (e.key) {
        case "ArrowDown":
          focusManager?.focusNext({ tabbable: true, wrap: true });
          break;
        case "ArrowUp":
          focusManager?.focusPrevious({ tabbable: true, wrap: true });
          break;
      }
    },
    [focusManager]
  );

  useEffect(() => {
    const element = dialogRef.current;
    if (element) {
      element.addEventListener("keydown", onKeyDown);
    }

    return () => {
      if (element) {
        element.removeEventListener("keydown", onKeyDown);
      }
    };
  }, [onKeyDown]);

  // Extract nodes from Relay connection and filter active memberships
  const activeMemberships =
    memberships?.edges
      ?.map((edge) => edge?.node)
      .filter((node): node is MembershipType => node != null && node.active) ??
    [];

  return (
    <AriaDialog
      {...dialogProps}
      ref={dialogRef}
      className={cx(
        "w-66 rounded-xl bg-secondary_alt shadow-lg ring ring-secondary_alt outline-hidden",
        className
      )}
    >
      <div className="rounded-xl bg-primary ring-1 ring-secondary">
        <div className="flex flex-col gap-0.5 py-1.5">
          <NavAccountCardMenuItem
            label="View profile"
            icon={User01}
            shortcut="⌘K->P"
          />
          <NavAccountCardMenuItem
            label="Account settings"
            icon={Settings01}
            shortcut="⌘S"
          />
          <NavAccountCardMenuItem label="Documentation" icon={BookOpen01} />
        </div>

        {activeMemberships.length > 0 && (
          <div className="flex flex-col gap-0.5 border-t border-secondary py-1.5">
            <div className="px-3 pt-1.5 pb-1 text-xs font-semibold text-tertiary">
              Switch organization
            </div>

            <div className="flex flex-col gap-0.5 px-1.5">
              {activeMemberships.map((membership) => {
                const isSelected =
                  membership.organization.slug === currentOrganizationId;

                return (
                  <button
                    key={membership.id}
                    onClick={() => {
                      // When switching organization, navigate to the same project
                      onOrganizationSwitch?.(
                        membership.organization.slug,
                        currentProjectId
                      );
                    }}
                    className={cx(
                      "relative w-full cursor-pointer rounded-md px-2 py-1.5 text-left outline-focus-ring hover:bg-primary_hover focus:z-10 focus-visible:outline-2 focus-visible:outline-offset-2",
                      isSelected && "bg-primary_hover"
                    )}
                  >
                    <AvatarLabelGroup
                      status="online"
                      size="md"
                      src="" // Organizations don't have avatars yet
                      title={membership.organization.name}
                      subtitle={membership.role.name}
                    />

                    <RadioButtonBase
                      isSelected={isSelected}
                      className="absolute top-2 right-2"
                    />
                  </button>
                );
              })}
            </div>
          </div>
        )}
      </div>

      <OverlayTriggerStateContext.Consumer>
        {(state) => (
          <div className="pt-1 pb-1.5">
            <NavAccountCardMenuItem
              label="Sign out"
              icon={LogOut01}
              shortcut="⌥⇧Q"
              onClick={async (e) => {
                e.preventDefault();
                e.stopPropagation();

                // Close popover first
                state?.close();

                // Small delay to let popover close animation finish
                await new Promise((resolve) => setTimeout(resolve, 100));

                // Then sign out
                await onSignOut?.();
              }}
            />
          </div>
        )}
      </OverlayTriggerStateContext.Consumer>
    </AriaDialog>
  );
};

const NavAccountCardMenuItem = ({
  icon: Icon,
  label,
  shortcut,
  ...buttonProps
}: {
  icon?: FC<{ className?: string }>;
  label: string;
  shortcut?: string;
} & HTMLAttributes<HTMLButtonElement>) => {
  return (
    <button
      {...buttonProps}
      className={cx(
        "group/item w-full cursor-pointer px-1.5 focus:outline-hidden",
        buttonProps.className
      )}
    >
      <div
        className={cx(
          "flex w-full items-center justify-between gap-3 rounded-md p-2 group-hover/item:bg-primary_hover",
          // Focus styles.
          "outline-focus-ring group-focus-visible/item:outline-2 group-focus-visible/item:outline-offset-2"
        )}
      >
        <div className="flex gap-2 text-sm font-semibold text-secondary group-hover/item:text-secondary_hover">
          {Icon && <Icon className="size-5 text-fg-quaternary" />} {label}
        </div>

        {shortcut && (
          <kbd className="flex rounded px-1 py-px font-body text-xs font-medium text-tertiary ring-1 ring-secondary ring-inset">
            {shortcut}
          </kbd>
        )}
      </div>
    </button>
  );
};

interface NavAccountCardProps {
  popoverPlacement?: Placement;
  /** Current user data to display in the card */
  user?: {
    id: string;
    email: string;
    firstName?: string | null;
    lastName?: string | null;
    memberships?: MembershipConnection | null;
  } | null;
  /** Current organization ID (slug from URL) */
  currentOrganizationId: string;
  /** Current project ID (slug from URL) */
  currentProjectId: string;
  /** Sign out handler (should be a server action) */
  onSignOut?: () => Promise<void> | void;
}

export const NavAccountCard = ({
  popoverPlacement,
  user,
  currentOrganizationId,
  currentProjectId,
  onSignOut,
}: NavAccountCardProps) => {
  const triggerRef = useRef<HTMLDivElement>(null);
  const isDesktop = useBreakpoint("lg");
  const router = useRouter();

  const handleSignOut = async (e?: React.MouseEvent) => {
    // Prevent default link behavior and stop propagation
    e?.preventDefault();
    e?.stopPropagation();

    // Call sign out handler from props
    await onSignOut?.();
  };

  const handleOrganizationSwitch = (
    organizationId: string,
    projectId: string
  ) => {
    // Navigate to the selected organization/project
    router.push(`/${organizationId}/${projectId}/home`);
  };

  if (!user) {
    return null;
  }

  // Display user name or email
  const displayName =
    user.firstName && user.lastName
      ? `${user.firstName} ${user.lastName}`
      : user.email;

  // Find current organization membership from Relay connection
  const currentMembership = user.memberships?.edges?.find(
    (edge) => edge?.node?.organization.slug === currentOrganizationId
  )?.node;

  return (
    <div
      ref={triggerRef}
      className="relative flex items-center gap-3 rounded-xl p-3 ring-1 ring-secondary ring-inset"
    >
      <AvatarLabelGroup
        size="md"
        src="" // TODO: Add avatar support
        title={displayName}
        subtitle={currentMembership?.organization.name || ""}
        status="online"
      />

      <div className="absolute top-1.5 right-1.5">
        <AriaDialogTrigger>
          <AriaButton className="flex cursor-pointer items-center justify-center rounded-md p-1.5 text-fg-quaternary outline-focus-ring transition duration-100 ease-linear hover:bg-primary_hover hover:text-fg-quaternary_hover focus-visible:outline-2 focus-visible:outline-offset-2 pressed:bg-primary_hover pressed:text-fg-quaternary_hover">
            <ChevronSelectorVertical className="size-4 shrink-0" />
          </AriaButton>
          <AriaPopover
            placement={
              popoverPlacement ?? (isDesktop ? "right bottom" : "top right")
            }
            triggerRef={triggerRef}
            offset={8}
            className={({ isEntering, isExiting }) =>
              cx(
                "origin-(--trigger-anchor-point) will-change-transform",
                isEntering &&
                  "duration-150 ease-out animate-in fade-in placement-right:slide-in-from-left-0.5 placement-top:slide-in-from-bottom-0.5 placement-bottom:slide-in-from-top-0.5",
                isExiting &&
                  "duration-100 ease-in animate-out fade-out placement-right:slide-out-to-left-0.5 placement-top:slide-out-to-bottom-0.5 placement-bottom:slide-out-to-top-0.5"
              )
            }
          >
            <NavAccountMenu
              memberships={user.memberships}
              currentOrganizationId={currentOrganizationId}
              currentProjectId={currentProjectId}
              onSignOut={handleSignOut}
              onOrganizationSwitch={handleOrganizationSwitch}
            />
          </AriaPopover>
        </AriaDialogTrigger>
      </div>
    </div>
  );
};
