/** @format */

import { render, screen } from "@testing-library/react";
import { CrudShowContent } from "../CrudShowContent";
import type { CrudConfig, CrudEntity } from "@/shared/lib/crud/types";

interface TestEntity extends CrudEntity {
  name: string;
  email: string;
  age: number;
}

describe("CrudShowContent", () => {
  const mockFormatFieldValue = jest.fn((value) => String(value));

  const mockConfig: Partial<CrudConfig<TestEntity>> = {
    formFields: [
      {
        name: "name",
        label: "Name",
        type: "text",
      },
      {
        name: "email",
        label: "Email",
        type: "email",
      },
      {
        name: "age",
        label: "Age",
        type: "number",
      },
    ],
  };

  const mockEntity: TestEntity = {
    id: "1",
    name: "John Doe",
    email: "john@example.com",
    age: 30,
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("should render all form fields", () => {
    render(
      <CrudShowContent
        config={mockConfig as CrudConfig<TestEntity>}
        entity={mockEntity}
        formatFieldValue={mockFormatFieldValue}
      />
    );

    expect(screen.getByText("Name")).toBeInTheDocument();
    expect(screen.getByText("Email")).toBeInTheDocument();
    expect(screen.getByText("Age")).toBeInTheDocument();
  });

  it("should display formatted field values", () => {
    render(
      <CrudShowContent
        config={mockConfig as CrudConfig<TestEntity>}
        entity={mockEntity}
        formatFieldValue={mockFormatFieldValue}
      />
    );

    expect(screen.getByText("John Doe")).toBeInTheDocument();
    expect(screen.getByText("john@example.com")).toBeInTheDocument();
    expect(screen.getByText("30")).toBeInTheDocument();
  });

  it("should call formatFieldValue for each field", () => {
    render(
      <CrudShowContent
        config={mockConfig as CrudConfig<TestEntity>}
        entity={mockEntity}
        formatFieldValue={mockFormatFieldValue}
      />
    );

    expect(mockFormatFieldValue).toHaveBeenCalledWith("John Doe", "text");
    expect(mockFormatFieldValue).toHaveBeenCalledWith(
      "john@example.com",
      "email"
    );
    expect(mockFormatFieldValue).toHaveBeenCalledWith(30, "number");
  });

  it("should handle undefined field values", () => {
    const entityWithUndefined: TestEntity = {
      ...mockEntity,
      email: undefined as unknown as string,
    };

    render(
      <CrudShowContent
        config={mockConfig as CrudConfig<TestEntity>}
        entity={entityWithUndefined}
        formatFieldValue={mockFormatFieldValue}
      />
    );

    expect(mockFormatFieldValue).toHaveBeenCalledWith(undefined, "email");
  });
});

