/** @format */

"use client";

import type { SVGProps } from "react";
import { cx } from "@/utils/cx";

/**
 * 403 Forbidden Illustration
 * Shows a lock icon with decorative elements
 */
export const ForbiddenIllustration = (props: SVGProps<SVGSVGElement>) => (
  <svg
    width="240"
    height="240"
    viewBox="0 0 240 240"
    fill="none"
    {...props}
    className={cx("text-fg-quaternary", props.className)}
  >
    {/* Background circles */}
    <circle cx="120" cy="120" r="118" stroke="currentColor" strokeWidth="2" />
    <circle cx="120" cy="120" r="90" stroke="currentColor" strokeWidth="2" />

    {/* Lock body */}
    <rect
      x="80"
      y="110"
      width="80"
      height="60"
      rx="8"
      stroke="currentColor"
      strokeWidth="3"
      fill="none"
    />

    {/* Lock shackle */}
    <path
      d="M90 110V90C90 73.4315 103.431 60 120 60C136.569 60 150 73.4315 150 90V110"
      stroke="currentColor"
      strokeWidth="3"
      strokeLinecap="round"
    />

    {/* Keyhole */}
    <circle cx="120" cy="135" r="6" fill="currentColor" />
    <rect
      x="117"
      y="135"
      width="6"
      height="20"
      rx="3"
      fill="currentColor"
    />

    {/* Decorative corner elements */}
    <path
      d="M40 40L60 40L60 60"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    />
    <path
      d="M200 40L180 40L180 60"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    />
    <path
      d="M40 200L60 200L60 180"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    />
    <path
      d="M200 200L180 200L180 180"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    />
  </svg>
);

/**
 * Small version of 403 Forbidden Illustration
 */
export const ForbiddenIllustrationSM = (props: SVGProps<SVGSVGElement>) => (
  <svg
    width="120"
    height="120"
    viewBox="0 0 120 120"
    fill="none"
    {...props}
    className={cx("text-fg-quaternary", props.className)}
  >
    {/* Background circle */}
    <circle cx="60" cy="60" r="58" stroke="currentColor" strokeWidth="2" />

    {/* Lock body */}
    <rect
      x="40"
      y="55"
      width="40"
      height="30"
      rx="4"
      stroke="currentColor"
      strokeWidth="2"
      fill="none"
    />

    {/* Lock shackle */}
    <path
      d="M45 55V45C45 38.3726 50.3726 33 57 33C63.6274 33 69 38.3726 69 45V55"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
    />

    {/* Keyhole */}
    <circle cx="60" cy="67" r="3" fill="currentColor" />
    <rect x="58.5" y="67" width="3" height="10" rx="1.5" fill="currentColor" />
  </svg>
);

