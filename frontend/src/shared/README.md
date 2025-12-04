# Shared Layer

Reusable infrastructure code that doesn't depend on business logic.

## Structure

- **base/** - Base UI components from UntitledUI library
  - Simple, atomic UI components (buttons, inputs, tooltips, etc.)
  - Components are added here automatically via `npx untitledui@latest add <component>`
  - Use alias `@/components/base/*` for imports
  - Example: `import { Button } from "@/components/base/buttons/button";`
  - ⚠️ Do not modify these components directly - they are from UntitledUI

- **foundations/** - Foundation components (icons, payment icons, etc.)
  - Components are added here automatically via `npx untitledui@latest add <component>`
  - Use alias `@/components/foundations/*` for imports
  - Example: `import { VisaIcon } from "@/components/foundations/payment-icons";`

- **application/** - Application structure components (built using UntitledUI examples)
  - Components that define application structure: Navigation, Sidebar, Header, Layout
  - These are YOUR custom components, built based on UntitledUI examples/patterns
  - Use alias `@/components/application/*` for imports
  - Example: `import { NavItem } from "@/components/application/app-navigation/base-components/nav-item";`

- **ui/** - Content UI components (from UntitledUI or custom)
  - Components used in content areas: Modals, Cards, Forms, Lists, Tables, Tabs, etc.
  - Can be from UntitledUI (added via `npx untitledui@latest add`) or your custom components
  - If `untitledui` adds a component to `application/` but it's a content component, move it here
  - Use alias `@/shared/ui/*` for imports
  - Example: `import { Tabs } from "@/shared/ui/tabs/tabs";` or `import { CustomModal } from "@/shared/ui/CustomModal";`

- **lib/** - Third-party library wrappers and configurations
- **api/** - API client configuration, interceptors, base requests
- **config/** - Application configuration (constants, env variables)
- **types/** - Common TypeScript types and interfaces
- **utils/** - Pure utility functions (formatters, validators, helpers)
  - Files are added here via `npx untitledui@latest add <component>` (e.g., `cx.ts`)
  - Use alias `@/utils/*` for imports
- **assets/** - Static assets (images, icons, fonts)

## When to use `base/` vs `application/` vs `ui/`?

- **`base/`** - Components from UntitledUI library (ready-made, don't modify)
  - Simple, atomic components: Button, Input, Tooltip
  
- **`application/`** - Application structure components (built using UntitledUI examples)
  - Components that define app structure: Navigation, Sidebar, Header, Layout
  - These are structural components that frame your application
  
- **`ui/`** - Content UI components (from UntitledUI or custom)
  - Components used in content areas: Modals, Cards, Forms, Lists, Tables, Tabs, etc.
  - Can be from UntitledUI or your custom components
  - If `untitledui` adds a component to `application/` but it's a content component (like Tabs), move it to `ui/`

**Key difference:**
- `application/` = Structure of the app (navigation, layout, sidebar) - frames the app
- `ui/` = Content components (modals, cards, forms, tabs) - used inside content areas

If you need to customize a component from `base/`, create a wrapper in `application/` or `ui/` that uses the base component.

## Rules

- ✅ Can import from other shared modules
- ❌ Cannot import from entities, features, widgets, or pages
- ✅ Should be framework-agnostic when possible
