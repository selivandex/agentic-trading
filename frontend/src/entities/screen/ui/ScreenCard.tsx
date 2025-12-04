/** @format */

"use client";

import Image from "next/image";
import { Badge } from "@/components/base/badges/badges";
import { Button } from "@/components/base/buttons/button";
import type { Screen } from "@/entities/screen/model";

interface ScreenCardProps {
  screen: Screen;
  onClick?: (id: string) => void;
  onEdit?: (id: string) => void;
  onDelete?: (id: string) => void;
  onArchive?: (id: string) => void;
}

/**
 * Check if image URL is valid and accessible
 * Filters out local file paths and invalid URLs
 */
const isValidImageUrl = (url: string | null | undefined): boolean => {
  if (!url) return false;

  // Filter out local file system paths
  if (
    url.startsWith("/var/") ||
    url.startsWith("/tmp/") ||
    url.startsWith("file://")
  ) {
    return false;
  }

  // Check if it's a valid HTTP(S) URL or relative path
  try {
    // Accept relative paths starting with /
    if (url.startsWith("/")) return true;

    // Validate absolute URLs
    const urlObj = new URL(url);
    return urlObj.protocol === "http:" || urlObj.protocol === "https:";
  } catch {
    return false;
  }
};

/**
 * Card component for displaying screen with hover overlay actions
 */
export const ScreenCard = ({
  screen,
  onClick,
  onEdit,
  onArchive,
}: ScreenCardProps) => {
  const handleEdit = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (onEdit) onEdit(screen.uid);
  };

  const handleView = () => {
    if (onClick) onClick(screen.uid);
  };

  const handleArchive = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (onArchive) onArchive(screen.uid);
  };

  // Check if preview image is valid
  const hasValidPreview = isValidImageUrl(screen.previewImage?.url);

  return (
    <div className="group relative flex flex-col overflow-hidden rounded-2xl border border-secondary bg-primary transition-all hover:shadow-xl">
      {/* Header Section - Title, Description, Badge */}
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0 flex-1 p-4">
          <span className="mb-1 font-semibold text-sm text-primary">
            {screen.name}
          </span>
          <p className="text-sm text-secondary">
            Used in {screen.scenariosCount || 0} scenario
            {screen.scenariosCount !== 1 ? "s" : ""}
          </p>
        </div>

        <Badge
          type="pill-color"
          color="gray"
          size="sm"
          className="absolute top-4 right-4"
        >
          {screen.kind}
        </Badge>
      </div>

      {/* Preview Section - Screen mockup with hover overlay */}
      <div
        className="relative overflow-hidden rounded-2xl border border-secondary"
        style={{ margin: "-1px" }}
      >
        {/* Screen Preview - Full iPhone mockup */}
        <div className="relative aspect-[9/19.5] w-full bg-secondary">
          {hasValidPreview ? (
            // Show preview image if available and valid
            <Image
              src={screen.previewImage!.url}
              alt={screen.name}
              fill
              className="object-cover"
              unoptimized={screen.previewImage!.url.startsWith("http")}
            />
          ) : (
            // Fallback placeholder if no preview image or invalid URL
            <div className="absolute inset-0 bg-primary">
              <div className="absolute inset-0 flex items-center justify-center">
                <div className="text-center">
                  <div className="mb-2 text-4xl">📱</div>
                  <div className="text-xs text-gray-400">{screen.kind}</div>
                </div>
              </div>
            </div>
          )}
        </div>

        {/* Hover Overlay with Actions */}
        <div className="absolute inset-0.5 flex items-center rounded-2xl justify-center bg-black/60 opacity-0 backdrop-blur-sm transition-opacity group-hover:opacity-100">
          <div className="flex w-full flex-col gap-3 p-6">
            {onEdit && (
              <Button
                size="lg"
                color="secondary"
                onClick={handleEdit}
                className="w-full justify-center"
              >
                Edit
              </Button>
            )}

            {onClick && (
              <Button
                size="lg"
                color="secondary"
                onClick={handleView}
                className="w-full justify-center"
              >
                View
              </Button>
            )}

            {onArchive && screen.status !== "ARCHIVED" && (
              <Button
                size="lg"
                color="primary-destructive"
                onClick={handleArchive}
                className="w-full justify-center"
              >
                Archive
              </Button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};
