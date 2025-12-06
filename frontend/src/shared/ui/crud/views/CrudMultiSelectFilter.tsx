/** @format */

"use client";

import { useEffect } from "react";
import { useListData } from "react-stately";
import { MultiSelect } from "@/shared/base/select/multi-select";
import type { FilterMetadata } from "@/shared/lib/crud/types";

interface CrudMultiSelectFilterProps {
  filter: FilterMetadata;
  value: unknown;
  onChange: (filterId: string, value: unknown) => void;
}

/**
 * MultiSelect Filter Component
 * Separated because it needs useListData hook
 */
export function CrudMultiSelectFilter({
  filter,
  value,
  onChange,
}: CrudMultiSelectFilterProps) {
  const selectedValues = (value as string[]) || [];

  // Convert FilterOption[] to SelectItemType[]
  const selectItems =
    filter.options?.map((opt) => ({
      id: opt.value,
      label: opt.label,
    })) || [];

  // Create ListData for selected items
  const selectedListData = useListData({
    initialItems: selectItems.filter((item) =>
      selectedValues.includes(item.id)
    ),
  });

  // Sync selectedListData when value changes externally
  useEffect(() => {
    const currentIds = selectedListData.items.map((item) => item.id);
    const newIds = selectedValues;

    // Check if we need to update
    const isEqual =
      currentIds.length === newIds.length &&
      currentIds.every((id) => newIds.includes(id));

    if (!isEqual) {
      // Clear and repopulate
      selectedListData.setSelectedKeys(new Set());
      selectedListData.remove(...currentIds);
      const itemsToAdd = selectItems.filter((item) =>
        newIds.includes(item.id)
      );
      itemsToAdd.forEach((item) => selectedListData.append(item));
    }
  }, [selectedValues]); // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <MultiSelect
      label={filter.name}
      placeholder={filter.placeholder || `Select ${filter.name}`}
      items={selectItems}
      selectedItems={selectedListData}
      onItemInserted={(key) => {
        const newValues = [...selectedValues, String(key)];
        onChange(filter.id, newValues);
      }}
      onItemCleared={(key) => {
        const newValues = selectedValues.filter((v) => v !== String(key));
        onChange(filter.id, newValues);
      }}
    >
      {(item) => <MultiSelect.Item id={item.id}>{item.label}</MultiSelect.Item>}
    </MultiSelect>
  );
}
