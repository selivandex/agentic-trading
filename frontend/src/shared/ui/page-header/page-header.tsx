/** @format */

"use client";

import type { ComponentType, FC, HTMLAttributes, ReactNode } from "react";
import { useRouter } from "next/navigation";
import { ArrowLeft } from "@untitledui/icons";
import { Button } from "@/shared/base/buttons/button";
import { Avatar } from "@/shared/base/avatar/avatar";
import { Input } from "@/shared/base/input/input";
import { Tabs, TabList, Tab, TabsSkeleton } from "@/shared/ui/tabs/tabs";
import { Breadcrumbs } from "@/shared/ui/breadcrumbs/breadcrumbs";
import { Skeleton } from "@/shared/ui/skeleton/skeleton";
import { cx } from "@/utils/cx";

export interface BreadcrumbItemData {
  /** URL to navigate to when the breadcrumb is clicked */
  href?: string;
  /** Icon component to display */
  icon?: FC<{ className?: string }> | ReactNode;
  /** Label text */
  label: string;
}

export interface TabItemData {
  /** Unique key for the tab */
  id: string;
  /** Label text */
  label: string;
  /** Badge to display next to the label */
  badge?: number | string;
  /** @deprecated Use onTabChange callback for client-side routing instead of href */
  href?: string;
}

export interface PageHeaderProps {
  /** Background variant */
  background?: "transparent" | "primary";
  /** Whether to show breadcrumbs */
  showBreadcrumbs?: boolean;
  /** Breadcrumb items */
  breadcrumbs?: BreadcrumbItemData[];
  /** Back button href for mobile */
  backHref?: string;
  /** Back button click handler */
  onBackClick?: () => void;
  /** Title text */
  title?: string;
  /** Description text */
  description?: string;
  /** Avatar props (if provided, shows avatar layout) */
  avatar?: {
    src?: string | null;
    alt?: string;
    initials?: string;
  };
  /** @deprecated Use PageHeader.Actions compound component instead */
  actions?: ReactNode;
  /** Search input props */
  search?: {
    placeholder?: string;
    value?: string;
    onChange?: (value: string) => void;
    icon?: ComponentType<HTMLAttributes<HTMLOrSVGElement>>;
    shortcut?: string | boolean;
  };
  /** Tab items */
  tabs?: TabItemData[];
  /** Selected tab key */
  selectedTab?: string;
  /** Tab selection handler */
  onTabChange?: (key: string) => void;
  /** Tab style variant */
  tabStyle?:
    | "underline"
    | "button-brand"
    | "button-gray"
    | "button-border"
    | "button-minimal";
  /** Additional className */
  className?: string;
  /** Children content */
  children?: ReactNode;
}

export const PageHeader = ({
  background = "primary",
  showBreadcrumbs = true,
  breadcrumbs,
  backHref,
  onBackClick,
  title,
  description,
  avatar,
  actions,
  search,
  tabs,
  selectedTab,
  onTabChange,
  tabStyle = "underline",
  className,
  children,
}: PageHeaderProps) => {
  const paddingClasses = "px-4 lg:px-8";
  const hasTabs = tabs && tabs.length > 0;
  const hasSearch = search !== undefined;
  const hasAvatar = avatar !== undefined;
  const hasTitle = title !== undefined;
  const hasDescription = description !== undefined;

  // Extract PageHeader.Actions from children
  const childrenArray = Array.isArray(children) ? children : [children];
  const actionsFromChildren = childrenArray.find(
    (child) => child?.type === PageHeaderActions
  );
  const hasActions = actions !== undefined || actionsFromChildren !== undefined;
  const otherChildren = childrenArray.filter(
    (child) => child?.type !== PageHeaderActions
  );

  const hasHeaderContent =
    hasTitle || hasDescription || hasAvatar || hasActions || hasSearch;

  return (
    <div
      className={cx(
        "relative w-full flex flex-col gap-4 pt-4 mb-5 border-b border-gray-200",
        !hasTabs ? "pb-4" : "", // Add padding-bottom when no tabs
        background === "primary" ? "bg-primary" : "bg-transparent",
        className
      )}
    >
      {/* Breadcrumbs - Desktop */}
      {showBreadcrumbs && breadcrumbs && breadcrumbs.length > 0 && (
        <div className={`max-lg:hidden ${paddingClasses}`}>
          <Breadcrumbs type="button" maxVisibleItems={3}>
            {breadcrumbs.map((item, index) => (
              <Breadcrumbs.Item key={index} href={item.href} icon={item.icon}>
                {item.label}
              </Breadcrumbs.Item>
            ))}
          </Breadcrumbs>
        </div>
      )}

      {/* Back Button - Mobile */}
      {(backHref || onBackClick) && (
        <div className="flex lg:hidden">
          <Button
            href={backHref}
            onClick={onBackClick}
            color="link-gray"
            size="md"
            iconLeading={ArrowLeft}
          >
            Back
          </Button>
        </div>
      )}

      {/* Header Content */}
      {hasHeaderContent && (
        <div
          className={`${cx(
            "flex flex-col gap-4",
            //hasTabs ? "border-b border-secondary pb-4" : "",
            "lg:flex-row lg:items-center lg:justify-between"
          )} ${paddingClasses}`}
        >
          {/* Left side: Avatar + Title/Description or Title/Description only */}
          {(hasAvatar || hasTitle || hasDescription) && (
            <div className="flex flex-1 items-center gap-3 lg:gap-4 min-w-0">
              {hasAvatar && (
                <Avatar
                  size="xl"
                  src={avatar.src}
                  alt={avatar.alt}
                  initials={avatar.initials}
                />
              )}
              {(hasTitle || hasDescription) && (
                <div className="min-w-0 flex-1">
                  {hasTitle && (
                    <h1 className="font-semibold text-primary text-lg">
                      {title}
                    </h1>
                  )}
                  {hasDescription && (
                    <p className="text-md text-balance text-tertiary">
                      {description}
                    </p>
                  )}
                </div>
              )}
            </div>
          )}

          {/* Right side: Actions + Search */}
          {(hasActions || hasSearch) && (
            <div className="flex items-center gap-3 flex-shrink-0">
              {/* Legacy actions prop (deprecated) */}
              {actions && (
                <div className="flex items-center gap-3">{actions}</div>
              )}
              {/* Compound component actions */}
              {actionsFromChildren}
              {hasSearch && (
                <Input
                  shortcut={search.shortcut}
                  className="w-full md:max-w-80"
                  size="sm"
                  aria-label="Search"
                  placeholder={search.placeholder || "Search"}
                  icon={search.icon}
                  value={search.value}
                  onChange={(value) => search.onChange?.(value)}
                />
              )}
            </div>
          )}
        </div>
      )}

      {/* Custom children content (excluding PageHeader.Actions) */}
      {otherChildren}

      {/* Tabs */}
      {hasTabs && (
        <Tabs
          selectedKey={selectedTab}
          onSelectionChange={(key) => onTabChange?.(String(key))}
          orientation="horizontal"
        >
          <TabList
            size="sm"
            type={tabStyle}
            className={paddingClasses}
            items={tabs.map((tab) => ({
              id: tab.id,
              label: tab.label,
              badge: tab.badge,
            }))}
          >
            {(item) => (
              <Tab key={item.id} id={item.id} badge={item.badge}>
                {item.label}
              </Tab>
            )}
          </TabList>
        </Tabs>
      )}
    </div>
  );
};

PageHeader.displayName = "PageHeader";

// Compound Components for Actions
/**
 * PageHeader.Actions - Action buttons section
 *
 * @example
 * ```tsx
 * <PageHeader title="Users">
 *   <PageHeader.Actions>
 *     <Button color="secondary">Export</Button>
 *     <Button color="primary">Add User</Button>
 *   </PageHeader.Actions>
 * </PageHeader>
 * ```
 */
interface PageHeaderActionsProps {
  children: ReactNode;
}

const PageHeaderActions = ({ children }: PageHeaderActionsProps) => {
  return <div className="flex items-center gap-3">{children}</div>;
};

PageHeaderActions.displayName = "PageHeader.Actions";

PageHeader.Actions = PageHeaderActions;

// Compound Components for Tabs
interface PageHeaderTabsProps {
  selectedTab?: string;
  tabStyle?:
    | "underline"
    | "button-brand"
    | "button-gray"
    | "button-border"
    | "button-minimal";
  children: ReactNode;
}

interface PageHeaderTabProps {
  id: string;
  href?: string;
  badge?: number | string;
  children: ReactNode;
}

const PageHeaderTabs = ({
  selectedTab,
  tabStyle = "underline",
  children,
}: PageHeaderTabsProps) => {
  const paddingClasses = "px-4 lg:px-8";
  const router = useRouter();

  // Extract tab data from children
  const tabChildren = Array.isArray(children) ? children : [children];
  const tabs = tabChildren
    .filter((child) => child?.type === PageHeaderTab)
    .map((child) => ({
      id: child.props.id,
      label: child.props.children,
      badge: child.props.badge,
      href: child.props.href,
    }));

  const handleSelectionChange = (key: string | number) => {
    const selectedTabData = tabs.find((tab) => tab.id === key);
    if (selectedTabData?.href) {
      router.push(selectedTabData.href);
    }
  };

  return (
    <Tabs
      selectedKey={selectedTab}
      onSelectionChange={handleSelectionChange}
      orientation="horizontal"
    >
      <TabList
        size="sm"
        type={tabStyle}
        className={paddingClasses}
        items={tabs}
      >
        {(item) => (
          <Tab key={item.id} id={item.id} badge={item.badge}>
            {item.label}
          </Tab>
        )}
      </TabList>
    </Tabs>
  );
};

const PageHeaderTab = ({ children }: PageHeaderTabProps) => {
  // This component is just a placeholder for type checking
  // Actual rendering happens in PageHeaderTabs
  return <>{children}</>;
};

PageHeader.Tabs = PageHeaderTabs;
PageHeader.Tab = PageHeaderTab;

/**
 * PageHeaderSkeleton - Loading state for PageHeader component
 *
 * Accepts the same props as PageHeader and automatically displays
 * skeleton placeholders for the provided layout configuration.
 */
export type PageHeaderSkeletonProps = Omit<
  PageHeaderProps,
  | "breadcrumbs"
  | "backHref"
  | "onBackClick"
  | "title"
  | "description"
  | "avatar"
  | "actions"
  | "search"
  | "tabs"
  | "selectedTab"
  | "onTabChange"
  | "children"
> & {
  /** Number of breadcrumb items to show (if showBreadcrumbs is true) */
  breadcrumbCount?: number;
  /** Whether to show avatar skeleton */
  showAvatar?: boolean;
  /** Whether to show title skeleton */
  showTitle?: boolean;
  /** Whether to show description skeleton */
  showDescription?: boolean;
  /** Number of action buttons to show */
  actionCount?: number;
  /** Whether to show search skeleton */
  showSearch?: boolean;
  /** Number of tabs to show */
  tabCount?: number;
};

export const PageHeaderSkeleton = ({
  background = "primary",
  showBreadcrumbs = true,
  breadcrumbCount = 3,
  showAvatar = false,
  showTitle = true,
  showDescription = false,
  actionCount = 1,
  showSearch = false,
  tabCount = 0,
  tabStyle = "underline",
  className,
}: PageHeaderSkeletonProps) => {
  const paddingClasses = "px-4 lg:px-8";
  const hasTabs = tabCount > 0;
  const hasActions = actionCount > 0;
  const hasHeaderContent =
    showTitle || showDescription || showAvatar || hasActions || showSearch;

  return (
    <div
      className={cx(
        "relative w-full flex flex-col gap-4 pt-4",
        background === "primary" ? "bg-primary" : "bg-transparent",
        className
      )}
    >
      {/* Breadcrumbs Skeleton - Desktop */}
      {showBreadcrumbs && breadcrumbCount > 0 && (
        <div className={`max-lg:hidden ${paddingClasses}`}>
          <div className="flex items-center gap-2">
            {Array.from({ length: breadcrumbCount }).map((_, index) => (
              <div key={index} className="flex items-center gap-2">
                {index > 0 && (
                  <Skeleton className="h-4 w-4 rounded-full" /> // Separator
                )}
                <Skeleton className="h-4 w-20 rounded" />
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Header Content Skeleton */}
      {hasHeaderContent && (
        <div
          className={`${cx(
            "flex flex-col gap-4",
            "lg:flex-row"
          )} ${paddingClasses}`}
        >
          {/* Left side: Avatar + Title/Description */}
          {(showAvatar || showTitle || showDescription) && (
            <div className="flex flex-1 items-center gap-3 lg:gap-4">
              {showAvatar && (
                <Skeleton className="h-16 w-16 rounded-full shrink-0" />
              )}
              {(showTitle || showDescription) && (
                <div className="flex flex-col gap-2 flex-1">
                  {showTitle && <Skeleton className="h-6 w-48 rounded" />}
                  {showDescription && <Skeleton className="h-4 w-64 rounded" />}
                </div>
              )}
            </div>
          )}

          {/* Right side: Actions + Search */}
          {(hasActions || showSearch) && (
            <div className="flex items-start gap-3">
              {hasActions &&
                Array.from({ length: actionCount }).map((_, index) => (
                  <Skeleton key={index} className="h-10 w-24 rounded-lg" />
                ))}
              {showSearch && (
                <Skeleton className="h-10 w-full md:max-w-80 rounded-lg" />
              )}
            </div>
          )}
        </div>
      )}

      {/* Tabs Skeleton */}
      {hasTabs && (
        <div className={`${paddingClasses}`}>
          <TabsSkeleton
            count={tabCount}
            type={
              tabStyle as
                | "underline"
                | "button-brand"
                | "button-gray"
                | "button-border"
                | "button-minimal"
            }
            size="sm"
          />
        </div>
      )}
    </div>
  );
};

PageHeaderSkeleton.displayName = "PageHeaderSkeleton";
