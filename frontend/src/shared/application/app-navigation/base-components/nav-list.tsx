/** @format */

"use client";

import { useState } from "react";
import { cx } from "@/utils/cx";
import type { NavItemDividerType, NavItemType } from "../config";
import { NavItemBase } from "./nav-item";

interface NavListProps {
  /** URL of the currently active item. */
  activeUrl?: string;
  /** Additional CSS classes to apply to the list. */
  className?: string;
  /** List of items to display. */
  items: (NavItemType | NavItemDividerType)[];
}

export const NavList = ({ activeUrl, items, className }: NavListProps) => {
  const [open, setOpen] = useState(false);

  // Check if current URL matches item or any of its children
  const isItemActive = (item: NavItemType) => {
    if (!activeUrl) return false;

    // Check if current URL matches the item href
    if (item.href === activeUrl) return true;

    // Check if current URL matches any child item
    if (item.items?.some((subItem) => subItem.href === activeUrl)) return true;

    // Check if current URL is a nested route under this item or its children
    if (activeUrl.startsWith(item.href + "/")) return true;

    if (
      item.items?.some((subItem) => activeUrl.startsWith(subItem.href + "/"))
    ) {
      return true;
    }

    return false;
  };

  const activeItem = items
    .filter((item): item is NavItemType => !item.divider)
    .find((item) => isItemActive(item));
  const [currentItem, setCurrentItem] = useState(activeItem);

  return (
    <ul className={cx("mt-4 flex flex-col px-2 lg:px-4", className)}>
      {items.map((item, index) => {
        if (item.divider) {
          return (
            <li key={index} className="w-full px-0.5 py-2">
              <hr className="h-px w-full border-none bg-border-secondary" />
            </li>
          );
        }

        if (item.items?.length) {
          return (
            <details
              key={item.label}
              open={isItemActive(item)}
              className="appearance-none py-0.5"
              onToggle={(e) => {
                setOpen(e.currentTarget.open);
                setCurrentItem(item);
              }}
            >
              <NavItemBase
                href={item.href}
                badge={item.badge}
                icon={item.icon}
                type="collapsible"
              >
                {item.label}
              </NavItemBase>

              <dd>
                <ul className="py-0.5">
                  {item.items.map((childItem) => (
                    <li key={childItem.label} className="py-0.5">
                      <NavItemBase
                        href={childItem.href}
                        badge={childItem.badge}
                        type="collapsible-child"
                        current={
                          activeUrl === childItem.href ||
                          (!!activeUrl &&
                            activeUrl.startsWith(childItem.href + "/"))
                        }
                      >
                        {childItem.label}
                      </NavItemBase>
                    </li>
                  ))}
                </ul>
              </dd>
            </details>
          );
        }

        return (
          <li key={item.label} className="py-0.5">
            <NavItemBase
              type="link"
              badge={item.badge}
              icon={item.icon}
              href={item.href}
              current={currentItem?.href === item.href}
              open={open && currentItem?.href === item.href}
              onClick={item.onClick}
            >
              {item.label}
            </NavItemBase>
          </li>
        );
      })}
    </ul>
  );
};
