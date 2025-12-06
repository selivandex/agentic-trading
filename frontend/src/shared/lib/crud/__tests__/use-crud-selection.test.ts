/** @format */

import { renderHook, act } from "@testing-library/react";
import { useCrudSelection } from "../use-crud-selection";
import type { CrudEntity, CrudActions } from "../types";

interface TestEntity extends CrudEntity {
  name: string;
}

describe("useCrudSelection", () => {
  const mockEntities: TestEntity[] = [
    { id: "1", name: "Entity 1" },
    { id: "2", name: "Entity 2" },
    { id: "3", name: "Entity 3" },
  ];

  const mockActions: CrudActions<TestEntity> = {
    setSelectedEntities: jest.fn(),
  } as unknown as CrudActions<TestEntity>;

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("should handle selection change with specific keys", () => {
    const { result } = renderHook(() =>
      useCrudSelection(mockEntities, mockActions)
    );

    const selectedKeys = new Set(["1", "2"]);

    act(() => {
      result.current.handleSelectionChange(selectedKeys);
    });

    expect(mockActions.setSelectedEntities).toHaveBeenCalledWith([
      { id: "1", name: "Entity 1" },
      { id: "2", name: "Entity 2" },
    ]);
  });

  it("should handle select all", () => {
    const { result } = renderHook(() =>
      useCrudSelection(mockEntities, mockActions)
    );

    act(() => {
      result.current.handleSelectionChange("all");
    });

    expect(mockActions.setSelectedEntities).toHaveBeenCalledWith(mockEntities);
  });

  it("should clear selection", () => {
    const { result } = renderHook(() =>
      useCrudSelection(mockEntities, mockActions)
    );

    act(() => {
      result.current.clearSelection();
    });

    expect(mockActions.setSelectedEntities).toHaveBeenCalledWith([]);
  });

  it("should handle empty selection", () => {
    const { result } = renderHook(() =>
      useCrudSelection(mockEntities, mockActions)
    );

    const emptyKeys = new Set<string>();

    act(() => {
      result.current.handleSelectionChange(emptyKeys);
    });

    expect(mockActions.setSelectedEntities).toHaveBeenCalledWith([]);
  });
});

