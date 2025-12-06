/** @format */

import { renderHook, act } from "@testing-library/react";
import { useCrudBatchActions } from "../use-crud-batch-actions";
import type { CrudConfig, CrudEntity, CrudActions } from "../types";

interface TestEntity extends CrudEntity {
  name: string;
}

describe("useCrudBatchActions", () => {
  const mockRefetch = jest.fn();
  const mockActionOnClick = jest.fn();

  const mockConfig: Partial<CrudConfig<TestEntity>> = {
    bulkActions: [
      {
        key: "test-action",
        label: "Test Action",
        onClick: mockActionOnClick,
      },
    ],
  };

  const selectedEntities: TestEntity[] = [
    { id: "1", name: "Entity 1" },
    { id: "2", name: "Entity 2" },
  ];

  const mockActions: CrudActions<TestEntity> = {
    setSelectedEntities: jest.fn(),
  } as unknown as CrudActions<TestEntity>;

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("should execute batch action for all selected entities", async () => {
    const { result } = renderHook(() =>
      useCrudBatchActions(
        mockConfig as CrudConfig<TestEntity>,
        selectedEntities,
        mockActions,
        mockRefetch
      )
    );

    await act(async () => {
      await result.current.executeBatchAction("test-action");
    });

    expect(mockActionOnClick).toHaveBeenCalledTimes(2);
    expect(mockActionOnClick).toHaveBeenCalledWith(selectedEntities[0]);
    expect(mockActionOnClick).toHaveBeenCalledWith(selectedEntities[1]);
  });

  it("should clear selection after batch action", async () => {
    const { result } = renderHook(() =>
      useCrudBatchActions(
        mockConfig as CrudConfig<TestEntity>,
        selectedEntities,
        mockActions,
        mockRefetch
      )
    );

    await act(async () => {
      await result.current.executeBatchAction("test-action");
    });

    expect(mockActions.setSelectedEntities).toHaveBeenCalledWith([]);
  });

  it("should refetch after batch action", async () => {
    const { result } = renderHook(() =>
      useCrudBatchActions(
        mockConfig as CrudConfig<TestEntity>,
        selectedEntities,
        mockActions,
        mockRefetch
      )
    );

    await act(async () => {
      await result.current.executeBatchAction("test-action");
    });

    expect(mockRefetch).toHaveBeenCalled();
  });

  it("should not execute if action not found", async () => {
    const { result } = renderHook(() =>
      useCrudBatchActions(
        mockConfig as CrudConfig<TestEntity>,
        selectedEntities,
        mockActions,
        mockRefetch
      )
    );

    await act(async () => {
      await result.current.executeBatchAction("non-existent-action");
    });

    expect(mockActionOnClick).not.toHaveBeenCalled();
    expect(mockRefetch).not.toHaveBeenCalled();
  });
});

