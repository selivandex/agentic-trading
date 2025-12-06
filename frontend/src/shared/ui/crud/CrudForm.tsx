/** @format */

"use client";

import { useCallback, useEffect } from "react";
import { Controller, useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { ArrowLeft } from "@untitledui/icons";
import { Button } from "@/shared/base/buttons/button";
import { Input } from "@/shared/base/input/input";
import { TextArea } from "@/shared/base/textarea/textarea";
import { Select } from "@/shared/base/select/select";
import { Checkbox } from "@/shared/base/checkbox/checkbox";
import { useCrudContext } from "@/shared/lib/crud/context";
import { useCrudShowQuery } from "@/shared/lib/crud/use-crud-query";
import { useCrudMutations } from "@/shared/lib/crud/use-crud-mutations";
import { Skeleton } from "@/shared/ui/skeleton/skeleton";
import { logger } from "@/shared/lib";
import type { CrudEntity, CrudFormField } from "@/shared/lib/crud/types";

/**
 * CRUD Form Component
 * Handles both create and edit modes
 */
export function CrudForm<TEntity extends CrudEntity = CrudEntity>({
  entityId,
  mode,
}: {
  entityId?: string;
  mode: "new" | "edit";
}) {
  const { config, actions } = useCrudContext<TEntity>();

  // Fetch entity for edit mode
  const { entity, loading: entityLoading } = useCrudShowQuery<TEntity>(
    config,
    entityId ?? "",
    { skip: mode === "new" || !entityId },
  );

  // Mutations
  const { create, update, isLoading: mutationLoading } =
    useCrudMutations<TEntity>(config);

  // Build validation schema from form fields
  const validationSchema = config.formValidationSchema ?? buildValidationSchema(config.formFields);

  // Initialize form
  const {
    control,
    handleSubmit,
    formState: { errors },
    reset,
  } = useForm({
    resolver: zodResolver(validationSchema),
    defaultValues: buildDefaultValues(config.formFields),
  });

  // Load entity data into form (edit mode)
  useEffect(() => {
    if (mode === "edit" && entity) {
      const formData = config.transformBeforeEdit
        ? config.transformBeforeEdit(entity)
        : entity;

      reset(formData);
    }
  }, [mode, entity, config, reset]);

  // Handle form submission
  const onSubmit = useCallback(
    async (data: Record<string, unknown>) => {
      try {
        if (mode === "new") {
          await create(data);
          actions.goToIndex();
        } else if (mode === "edit" && entityId) {
          await update(entityId, data);
          actions.goToIndex();
        }
      } catch (error) {
        // Error is handled by mutation hook
        logger.error("Form submission error:", error);
      }
    },
    [mode, entityId, create, update, actions],
  );

  // Render loading state
  if (mode === "edit" && entityLoading) {
    return (
      <div className="mx-auto max-w-3xl">
        <div className="mb-8 flex items-center gap-4">
          <Skeleton className="h-10 w-24" />
          <Skeleton className="h-8 w-48" />
        </div>

        <div className="rounded-xl border border-gray-200 bg-white p-8 shadow-sm">
          <div className="space-y-6">
            {Array.from({ length: 6 }).map((_, i) => (
              <div key={i} className="space-y-2">
                <Skeleton className="h-4 w-32" />
                <Skeleton className="h-10 w-full" />
              </div>
            ))}
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-3xl">
      {/* Header */}
      <div className="mb-8 flex items-center gap-4">
        <Button
          color="secondary"
          size="md"
          iconLeading={ArrowLeft}
          onClick={() => actions.goToIndex()}
        >
          Back
        </Button>
        <h1 className="text-2xl font-semibold text-gray-900">
          {mode === "new" ? "New" : "Edit"} {config.resourceName}
        </h1>
      </div>

      {/* Form */}
      <form
        onSubmit={handleSubmit(onSubmit)}
        className="rounded-xl border border-gray-200 bg-white p-8 shadow-sm"
      >
        <div className="space-y-6">
          {config.formFields
            .filter((field) => !field.hidden)
            .map((field) => (
              <div key={field.name}>
                {renderFormField(field, control, errors)}
              </div>
            ))}
        </div>

        {/* Form actions */}
        <div className="mt-8 flex items-center justify-end gap-3 border-t border-gray-200 pt-6">
          <Button
            type="button"
            color="secondary"
            size="lg"
            onClick={() => actions.goToIndex()}
            isDisabled={mutationLoading}
          >
            Cancel
          </Button>
          <Button type="submit" size="lg" isDisabled={mutationLoading}>
            {mutationLoading ? "Saving..." : mode === "new" ? "Create" : "Update"}
          </Button>
        </div>
      </form>
    </div>
  );
}

/**
 * Build Zod validation schema from form fields
 */
function buildValidationSchema(
  fields: CrudFormField[],
): z.ZodObject<Record<string, z.ZodTypeAny>> {
  const shape: Record<string, z.ZodTypeAny> = {};

  for (const field of fields) {
    if (field.validation) {
      shape[field.name] = field.validation;
    } else {
      // Default validations based on type
      switch (field.type) {
        case "email":
          shape[field.name] = z.string().email();
          break;
        case "number":
          shape[field.name] = z.number();
          break;
        case "checkbox":
          shape[field.name] = z.boolean();
          break;
        default:
          shape[field.name] = z.string();
      }
    }
  }

  return z.object(shape);
}

/**
 * Build default form values
 */
function buildDefaultValues(
  fields: CrudFormField[],
): Record<string, unknown> {
  const defaults: Record<string, unknown> = {};

  for (const field of fields) {
    if (field.defaultValue !== undefined) {
      defaults[field.name] = field.defaultValue;
    } else {
      // Default values based on type
      switch (field.type) {
        case "checkbox":
          defaults[field.name] = false;
          break;
        case "number":
          defaults[field.name] = 0;
          break;
        default:
          defaults[field.name] = "";
      }
    }
  }

  return defaults;
}

/**
 * Render individual form field
 */
function renderFormField(
  field: CrudFormField,
  control: ReturnType<typeof useForm>["control"],
  errors: ReturnType<typeof useForm>["formState"]["errors"],
) {
  const error = errors[field.name]?.message as string | undefined;

  return (
    <Controller
      name={field.name}
      control={control}
      render={({ field: { onChange, onBlur, value, ref } }) => {
        // Custom render function
        if (field.render) {
          return field.render({
            field,
            value,
            onChange,
            error,
            disabled: field.disabled,
          });
        }

        // Standard field types
        switch (field.type) {
          case "textarea":
            return (
              <TextArea
                label={field.label}
                placeholder={field.placeholder}
                hint={field.helperText}
                isInvalid={!!error}
                isDisabled={field.disabled}
                value={value as string}
                onChange={onChange}
                onBlur={onBlur}
                textAreaRef={ref}
              />
            );

          case "select":
            return (
              <Select
                label={field.label}
                placeholder={field.placeholder}
                hint={field.helperText}
                isInvalid={!!error}
                selectedKey={value as string}
                onSelectionChange={onChange}
              >
                {field.options?.map((option) => (
                  <Select.Item key={option.value} id={String(option.value)}>
                    {option.label}
                  </Select.Item>
                ))}
              </Select>
            );

          case "checkbox":
            return (
              <div className="flex items-start gap-3 rounded-lg border border-gray-200 p-4">
                <Checkbox
                  isSelected={value as boolean}
                  onChange={onChange}
                  isDisabled={field.disabled}
                />
                <div className="flex-1">
                  <label className="text-sm font-medium text-gray-900">
                    {field.label}
                  </label>
                  {field.helperText && (
                    <p className="mt-1 text-sm text-gray-600">{field.helperText}</p>
                  )}
                  {error && <p className="mt-1 text-sm text-red-600">{error}</p>}
                </div>
              </div>
            );

          case "date":
          case "datetime":
            return (
              <Input
                type={field.type === "datetime" ? "datetime-local" : "date"}
                label={field.label}
                placeholder={field.placeholder}
                hint={field.helperText}
                isInvalid={!!error}
                isDisabled={field.disabled}
                value={value as string}
                onChange={onChange}
                onBlur={onBlur}
                ref={ref}
              />
            );

          case "number":
            return (
              <Input
                type="number"
                label={field.label}
                placeholder={field.placeholder}
                hint={field.helperText}
                isInvalid={!!error}
                isDisabled={field.disabled}
                value={value as string}
                onChange={(val) => onChange(parseFloat(val as string))}
                onBlur={onBlur}
                ref={ref}
              />
            );

          default:
            return (
              <Input
                type={field.type as string}
                label={field.label}
                placeholder={field.placeholder}
                hint={field.helperText}
                isInvalid={!!error}
                isDisabled={field.disabled}
                value={value as string}
                onChange={onChange}
                onBlur={onBlur}
                ref={ref}
              />
            );
        }
      }}
    />
  );
}
