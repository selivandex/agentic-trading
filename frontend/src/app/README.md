# App Layer (Next.js App Router)

Next.js application pages, layouts, and routing configuration.

## Structure

This layer follows Next.js App Router conventions:
- **layout.tsx** - Root and nested layouts
- **page.tsx** - Page components
- **loading.tsx** - Loading UI
- **error.tsx** - Error UI
- **not-found.tsx** - 404 UI
- **route.ts** - API routes

## Rules

- ✅ Can import from all layers (shared, entities, features, widgets)
- ✅ Should compose widgets and features into pages
- ✅ Handles routing and Next.js-specific logic
