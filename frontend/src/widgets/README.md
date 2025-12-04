# Widgets Layer

Large independent blocks that compose multiple features and entities.

## Structure

Each widget should have its own folder with:
- **ui/** - Widget UI components
- **model/** - Widget state management (if needed)

## Example Structure

```
widgets/
  auth-layout/
    ui/
      AuthLayout.tsx
  project-layout/
    ui/
      ProjectLayout.tsx
      Breadcrumbs.tsx
  scenario-editor/
    ui/
      ScenarioEditor.tsx
      Canvas.tsx
  macros-manager/
    ui/
      MacrosManager.tsx
    README.md
```

## Rules

- ✅ Can import from shared, entities, and features layers
- ❌ Cannot import from pages or other widgets
- ✅ Should be independent, reusable page sections
