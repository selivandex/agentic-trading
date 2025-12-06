/** @format */

import { render, screen, fireEvent } from "@testing-library/react";
import { CrudBatchActionsToolbar } from "../CrudBatchActionsToolbar";
import type { CrudConfig, CrudEntity } from "@/shared/lib/crud/types";

interface TestEntity extends CrudEntity {
  name: string;
}

describe("CrudBatchActionsToolbar", () => {
  const mockOnClearSelection = jest.fn();
  const mockOnExecuteAction = jest.fn();

  const mockConfig: Partial<CrudConfig<TestEntity>> = {
    bulkActions: [
      {
        key: "action1",
        label: "Action 1",
        onClick: jest.fn(),
      },
      {
        key: "action2",
        label: "Action 2",
        onClick: jest.fn(),
        destructive: true,
      },
    ],
  };

  const selectedEntities: TestEntity[] = [
    { id: "1", name: "Entity 1" },
    { id: "2", name: "Entity 2" },
  ];

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("should render selected count", () => {
    render(
      <CrudBatchActionsToolbar
        config={mockConfig as CrudConfig<TestEntity>}
        selectedCount={2}
        selectedEntities={selectedEntities}
        onClearSelection={mockOnClearSelection}
        onExecuteAction={mockOnExecuteAction}
      />
    );

    expect(screen.getByText("2 selected")).toBeInTheDocument();
  });

  it("should render clear button", () => {
    render(
      <CrudBatchActionsToolbar
        config={mockConfig as CrudConfig<TestEntity>}
        selectedCount={2}
        selectedEntities={selectedEntities}
        onClearSelection={mockOnClearSelection}
        onExecuteAction={mockOnExecuteAction}
      />
    );

    const clearButton = screen.getByText("Clear");
    expect(clearButton).toBeInTheDocument();

    fireEvent.click(clearButton);
    expect(mockOnClearSelection).toHaveBeenCalled();
  });

  it("should render all bulk actions", () => {
    render(
      <CrudBatchActionsToolbar
        config={mockConfig as CrudConfig<TestEntity>}
        selectedCount={2}
        selectedEntities={selectedEntities}
        onClearSelection={mockOnClearSelection}
        onExecuteAction={mockOnExecuteAction}
      />
    );

    expect(screen.getByText("Action 1")).toBeInTheDocument();
    expect(screen.getByText("Action 2")).toBeInTheDocument();
  });

  it("should execute action on button click", () => {
    render(
      <CrudBatchActionsToolbar
        config={mockConfig as CrudConfig<TestEntity>}
        selectedCount={2}
        selectedEntities={selectedEntities}
        onClearSelection={mockOnClearSelection}
        onExecuteAction={mockOnExecuteAction}
      />
    );

    const actionButton = screen.getByText("Action 1");
    fireEvent.click(actionButton);

    expect(mockOnExecuteAction).toHaveBeenCalledWith("action1");
  });

  it("should not render if no bulk actions configured", () => {
    const configWithoutBulkActions: Partial<CrudConfig<TestEntity>> = {
      bulkActions: undefined,
    };

    const { container } = render(
      <CrudBatchActionsToolbar
        config={configWithoutBulkActions as CrudConfig<TestEntity>}
        selectedCount={2}
        selectedEntities={selectedEntities}
        onClearSelection={mockOnClearSelection}
        onExecuteAction={mockOnExecuteAction}
      />
    );

    expect(container.firstChild).toBeNull();
  });

  it("should disable action if disabled condition met", () => {
    const configWithDisabled: Partial<CrudConfig<TestEntity>> = {
      bulkActions: [
        {
          key: "action1",
          label: "Action 1",
          onClick: jest.fn(),
          disabled: (entity) => entity.name === "Entity 1",
        },
      ],
    };

    render(
      <CrudBatchActionsToolbar
        config={configWithDisabled as CrudConfig<TestEntity>}
        selectedCount={2}
        selectedEntities={selectedEntities}
        onClearSelection={mockOnClearSelection}
        onExecuteAction={mockOnExecuteAction}
      />
    );

    const actionButton = screen.getByText("Action 1").closest("button");
    expect(actionButton).toBeDisabled();
  });
});

