/** @format */

"use client";

import { createContext, useContext, useState, ReactNode } from "react";

export interface FilterSidebarContent {
  component: ReactNode;
  isVisible: boolean;
}

interface FilterSidebarContextValue {
  content: FilterSidebarContent | null;
  setContent: (content: FilterSidebarContent | null) => void;
  show: () => void;
  hide: () => void;
}

const FilterSidebarContext = createContext<FilterSidebarContextValue | null>(
  null
);

export function FilterSidebarProvider({ children }: { children: ReactNode }) {
  const [content, setContent] = useState<FilterSidebarContent | null>(null);

  const show = () => {
    if (content) {
      setContent({ ...content, isVisible: true });
    }
  };

  const hide = () => {
    if (content) {
      setContent({ ...content, isVisible: false });
    }
  };

  return (
    <FilterSidebarContext.Provider
      value={{ content, setContent, show, hide }}
    >
      {children}
    </FilterSidebarContext.Provider>
  );
}

export function useFilterSidebar() {
  const context = useContext(FilterSidebarContext);
  if (!context) {
    throw new Error(
      "useFilterSidebar must be used within FilterSidebarProvider"
    );
  }
  return context;
}
