/** @format */

/**
 * Detect if the current platform is macOS
 */
export const isMacOS = (): boolean => {
  if (typeof window === "undefined") return false;
  return /Mac|iPhone|iPad|iPod/.test(navigator.platform);
};

/**
 * Get platform-specific keyboard shortcut display
 */
export const getShortcutKey = (key: string): string => {
  const isMac = isMacOS();

  const shortcuts: Record<string, { mac: string; windows: string }> = {
    undo: { mac: "⌘Z", windows: "Ctrl+Z" },
    redo: { mac: "⌘⇧Z", windows: "Ctrl+Shift+Z" },
    copy: { mac: "⌘C", windows: "Ctrl+C" },
    cut: { mac: "⌘X", windows: "Ctrl+X" },
    paste: { mac: "⌘V", windows: "Ctrl+V" },
    duplicate: { mac: "⌘D", windows: "Ctrl+D" },
    delete: { mac: "⌫", windows: "Del" },
    selectAll: { mac: "⌘A", windows: "Ctrl+A" },
  };

  const shortcut = shortcuts[key];
  if (!shortcut) return "";

  return isMac ? shortcut.mac : shortcut.windows;
};
