/** @format */

"use client";

import {
  createContext,
  useContext,
  useState,
  useCallback,
  useEffect,
  useRef,
  type ReactNode,
} from "react";

interface ContextMenuState {
  id: string | null;
  position: { x: number; y: number };
  content: ReactNode | null;
}

interface ContextMenuContextValue {
  state: ContextMenuState;
  openMenu: (
    id: string,
    position: { x: number; y: number },
    content: ReactNode
  ) => void;
  closeMenu: () => void;
  isMenuOpen: (id: string) => boolean;
}

const ContextMenuContext = createContext<ContextMenuContextValue | null>(null);

/**
 * Context Menu Manager Provider
 * Manages global state for context menus - only one can be open at a time
 */
export const ContextMenuProvider = ({ children }: { children: ReactNode }) => {
  const [state, setState] = useState<ContextMenuState>({
    id: null,
    position: { x: 0, y: 0 },
    content: null,
  });

  const openMenu = useCallback(
    (id: string, position: { x: number; y: number }, content: ReactNode) => {
      setState({ id, position, content });
    },
    []
  );

  const closeMenu = useCallback(() => {
    setState({ id: null, position: { x: 0, y: 0 }, content: null });
  }, []);

  const isMenuOpen = useCallback(
    (id: string) => {
      return state.id === id;
    },
    [state.id]
  );

  return (
    <ContextMenuContext.Provider
      value={{ state, openMenu, closeMenu, isMenuOpen }}
    >
      {children}
    </ContextMenuContext.Provider>
  );
};

/**
 * Hook to access context menu manager
 */
export const useContextMenuManager = () => {
  const context = useContext(ContextMenuContext);
  if (!context) {
    throw new Error(
      "useContextMenuManager must be used within ContextMenuProvider"
    );
  }
  return context;
};

/**
 * Global Context Menu Renderer
 * Renders the currently active context menu in a portal
 */
export const ContextMenuRenderer = () => {
  const { state, closeMenu } = useContextMenuManager();
  const menuRef = useRef<HTMLDivElement>(null);

  // Close menu on click outside & block native context menu
  useEffect(() => {
    if (!state.id) return;

    const handleClickOutside = (e: MouseEvent) => {
      // Close menu on left click outside
      if (
        e.button === 0 &&
        menuRef.current &&
        !menuRef.current.contains(e.target as Node)
      ) {
        closeMenu();
      }
    };

    const handleGlobalContextMenu = (e: MouseEvent) => {
      // Prevent native context menu globally when our menu is open
      // But don't stop propagation - let our wrapper handlers work
      e.preventDefault();
    };

    // Use capture phase to ensure we catch events before other handlers
    document.addEventListener("mousedown", handleClickOutside, true);
    document.addEventListener("contextmenu", handleGlobalContextMenu, true);

    return () => {
      document.removeEventListener("mousedown", handleClickOutside, true);
      document.removeEventListener(
        "contextmenu",
        handleGlobalContextMenu,
        true
      );
    };
  }, [state.id, closeMenu]);

  if (!state.id || !state.content) {
    return null;
  }

  const handleMenuContextMenu = (e: React.MouseEvent) => {
    // Prevent native context menu on menu itself
    e.preventDefault();
    e.stopPropagation();
  };

  return (
    <div
      ref={menuRef}
      key={`${state.position.x}-${state.position.y}`}
      style={{
        position: "fixed",
        left: `${state.position.x}px`,
        top: `${state.position.y}px`,
        zIndex: 9999,
      }}
      onContextMenu={handleMenuContextMenu}
      className="min-w-[12rem] overflow-auto rounded-lg bg-primary shadow-lg ring-1 ring-secondary_alt animate-in fade-in zoom-in-95 duration-150"
    >
      {state.content}
    </div>
  );
};
