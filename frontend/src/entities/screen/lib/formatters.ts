/** @format */

/**
 * Format screen name for display
 */
export const formatScreenName = (name: string): string => {
  return name.trim();
};

/**
 * Generate screen slug from name
 */
export const generateScreenSlug = (name: string): string => {
  return name
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
};




