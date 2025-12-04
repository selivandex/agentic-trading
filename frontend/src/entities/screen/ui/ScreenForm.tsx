/** @format */

"use client";

import { Input, InputBase } from "@/components/base/input/input";
import { InputGroup } from "@/components/base/input/input-group";
import { Skeleton } from "@/shared/ui/skeleton/skeleton";
import type { Screen } from "@/entities/screen/model";

type ScreenFormData = Pick<Screen, "name" | "url">;

interface ScreenFormProps {
  /** Form data */
  formData: ScreenFormData;
  /** Callback when form data changes */
  onChange: (data: ScreenFormData) => void;
}

/**
 * Remove protocol (http://, https://, etc.) from URL
 */
const removeProtocol = (url: string): string => {
  return url.replace(/^(https?:\/\/)/, "");
};

/**
 * Add https:// protocol to URL if not present
 */
const addProtocol = (url: string): string => {
  // Convert to string in case it's not
  const urlString = String(url || "");
  if (!urlString) return "";
  // If already has protocol, return as is
  if (urlString.match(/^https?:\/\//)) return urlString;
  // Otherwise add https://
  return `https://${urlString}`;
};

/**
 * Screen form component for creating/editing screens
 * Contains Name and URL fields with validation
 */
export function ScreenForm({ formData, onChange }: ScreenFormProps) {
  // Display URL without protocol (since we have https:// prefix)
  const displayUrl = removeProtocol(formData.url);

  const handleUrlChange = (value: unknown) => {
    // InputBase might pass event or string, handle both cases
    let valueWithoutProtocol: string;

    if (typeof value === "string") {
      valueWithoutProtocol = value;
    } else if (value && typeof value === "object" && "target" in value) {
      // It's an event
      valueWithoutProtocol = (value as React.ChangeEvent<HTMLInputElement>)
        .target.value;
    } else {
      valueWithoutProtocol = String(value || "");
    }

    // Store URL with protocol in state
    const urlWithProtocol = addProtocol(valueWithoutProtocol);
    onChange({ ...formData, url: urlWithProtocol });
  };

  return (
    <div className="flex flex-col">
      {/* Name field */}
      <div className="flex items-start border-b border-secondary py-5">
        <label
          htmlFor="screen-name"
          className="w-1/3 pt-2.5 text-sm font-semibold text-secondary"
        >
          Name
          <span className="ml-0.5 text-error">*</span>
        </label>
        <div className="flex-1">
          <Input
            id="screen-name"
            name="name"
            value={formData.name}
            onChange={(value) => onChange({ ...formData, name: value })}
            placeholder="Enter screen name"
            size="md"
            isRequired
          />
        </div>
      </div>

      {/* URL field */}
      <div className="flex items-start border-b border-secondary py-5">
        <div className="w-1/3 pt-2.5">
          <label className="text-sm font-semibold text-secondary">
            Figma Sites URL
            <span className="ml-0.5 text-error">*</span>
          </label>
          <p className="mt-1 text-sm text-tertiary">
            Paste screen webpage URL from Figma Sites.
          </p>
        </div>
        <div className="flex-1">
          <InputGroup
            size="md"
            isRequired
            leadingAddon={<InputGroup.Prefix>https://</InputGroup.Prefix>}
          >
            <InputBase
              id="screen-url"
              name="url"
              type="url"
              value={displayUrl}
              onChange={handleUrlChange}
              placeholder="www.example.com/screen"
            />
          </InputGroup>
        </div>
      </div>
    </div>
  );
}

/**
 * ScreenFormSkeleton - Loading state for ScreenForm component
 *
 * Displays skeleton placeholders that match the ScreenForm layout
 */
export function ScreenFormSkeleton() {
  return (
    <div className="flex flex-col">
      {/* Name field skeleton */}
      <div className="flex items-start border-b border-secondary py-5">
        <div className="w-1/3 pt-2.5">
          <Skeleton className="h-5 w-16 rounded" />
        </div>
        <div className="flex-1">
          <Skeleton className="h-10 w-full rounded-lg" />
        </div>
      </div>

      {/* URL field skeleton */}
      <div className="flex items-start border-b border-secondary py-5">
        <div className="w-1/3 pt-2.5">
          <Skeleton className="h-5 w-32 rounded mb-2" />
          <Skeleton className="h-4 w-48 rounded" />
        </div>
        <div className="flex-1">
          <Skeleton className="h-10 w-full rounded-lg" />
        </div>
      </div>
    </div>
  );
}
