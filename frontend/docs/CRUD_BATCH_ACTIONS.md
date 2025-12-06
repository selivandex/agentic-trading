<!-- @format -->

# CRUD Batch Actions API

## Overview

Batch actions allow you to perform operations on multiple selected entities at once.

## How to Enable

### 1. Enable Selection in Config

```tsx
const config: CrudConfig<MyEntity> = {
  // ... other config
  enableSelection: true, // Required for batch actions
  bulkActions: [], // Define batch actions here
};
```

### 2. Define Batch Actions

Batch actions are defined in the `bulkActions` array. Each action will be executed for **each selected entity**.

```tsx
interface CrudAction<TEntity> {
  key: string; // Unique key
  label: string; // Button label
  icon?: React.FC; // Optional icon
  onClick: (entity: TEntity) => void | Promise<void>; // Action handler
  disabled?: (entity: TEntity) => boolean; // Conditional disable
  hidden?: (entity: TEntity) => boolean; // Conditional hide
  destructive?: boolean; // Red/destructive styling
}
```

## Examples

### Basic Delete All

```tsx
const config: CrudConfig<Strategy> = {
  // ... other config
  enableSelection: true,
  bulkActions: [
    {
      key: "delete-all",
      label: "Delete Selected",
      icon: Trash01,
      onClick: async (strategy) => {
        await deleteStrategy({ variables: { id: strategy.id } });
      },
      destructive: true,
    },
  ],
};
```

### Multiple Batch Actions

```tsx
const config: CrudConfig<Strategy> = {
  enableSelection: true,
  bulkActions: [
    // Pause selected
    {
      key: "pause-all",
      label: "Pause Selected",
      onClick: async (strategy) => {
        await pauseStrategy({ variables: { id: strategy.id } });
      },
    },
    // Resume selected
    {
      key: "resume-all",
      label: "Resume Selected",
      icon: PlayCircle,
      onClick: async (strategy) => {
        await resumeStrategy({ variables: { id: strategy.id } });
      },
    },
    // Delete selected
    {
      key: "delete-all",
      label: "Delete Selected",
      icon: Trash01,
      onClick: async (strategy) => {
        await deleteStrategy({ variables: { id: strategy.id } });
      },
      destructive: true,
    },
  ],
};
```

### With Conditional Logic

```tsx
{
  key: "approve-all",
  label: "Approve Selected",
  onClick: async (item) => {
    await approveItem({ variables: { id: item.id } });
  },
  // Disable if any selected item is already approved
  disabled: (item) => item.status === "approved",
  // Hide for admin users
  hidden: (item) => currentUser.role === "admin",
}
```

### With Error Handling

```tsx
{
  key: "process-all",
  label: "Process Selected",
  onClick: async (item) => {
    try {
      await processItem({ variables: { id: item.id } });
      toast.success(`Processed ${item.name}`);
    } catch (error) {
      const message = error instanceof Error ? error.message : "Failed";
      toast.error(`Failed to process ${item.name}: ${message}`);
      throw error; // Re-throw to stop batch processing
    }
  },
}
```

## Full Example (StrategyManager)

```tsx
export function StrategyManager() {
  const [pauseStrategy, { loading: pauseLoading }] = usePauseStrategy();
  const [resumeStrategy, { loading: resumeLoading }] = useResumeStrategy();
  const [closeStrategy, { loading: closeLoading }] = useCloseStrategy();

  const enhancedConfig = useMemo(
    () => ({
      ...strategyCrudConfig,

      // Batch actions for multiple selection
      bulkActions: [
        {
          key: "pause-all",
          label: "Pause Selected",
          onClick: async (strategy: Strategy) => {
            await pauseStrategy({ variables: { id: strategy.id } });
          },
          disabled: () => pauseLoading,
        },
        {
          key: "resume-all",
          label: "Resume Selected",
          icon: PlayCircle,
          onClick: async (strategy: Strategy) => {
            await resumeStrategy({ variables: { id: strategy.id } });
          },
          disabled: () => resumeLoading,
        },
        {
          key: "close-all",
          label: "Close Selected",
          icon: XClose,
          onClick: async (strategy: Strategy) => {
            await closeStrategy({ variables: { id: strategy.id } });
          },
          disabled: () => closeLoading,
          destructive: true,
        },
      ],
    }),
    [pauseStrategy, pauseLoading, resumeStrategy, resumeLoading, closeStrategy, closeLoading]
  );

  return <Crud config={enhancedConfig} />;
}
```

## UI Behavior

1. **Selection Required**: Batch actions toolbar only appears when entities are selected
2. **Select All**: Click checkbox in table header to select all visible entities
3. **Toolbar**: Shows at the top of the table with:
   - Selected count
   - Clear button
   - Batch action buttons
4. **Execution**: Runs `onClick` for each selected entity sequentially
5. **Auto-clear**: Selection cleared and data refetched after batch action completes

## Best Practices

1. **Confirmation**: Add confirmation for destructive actions inside `onClick`
2. **Loading States**: Use `disabled: () => loading` to prevent double-clicks
3. **Error Handling**: Throw errors to stop batch processing on first failure
4. **Toasts**: Show feedback for each processed entity
5. **Performance**: Be mindful of executing many mutations at once

## Styling

- **Normal actions**: `color: "secondary"` (default)
- **Destructive actions**: Set `destructive: true` for red styling
