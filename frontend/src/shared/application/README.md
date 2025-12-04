# Application Components

Custom composite components built using [UntitledUI](https://www.untitledui.com/) examples and patterns.

## Overview

These are YOUR custom complex, composed components designed for application structure (navigation, sidebar, etc.). You create these components based on UntitledUI examples/patterns, but they are your own code that you can modify as needed.

## Creating Components

You create these components yourself, using UntitledUI examples as reference. They are not automatically added - you build them based on UntitledUI patterns.

For example, you might create:
- Navigation components based on UntitledUI navigation examples
- Sidebar components based on UntitledUI sidebar patterns
- Layout components using UntitledUI composition patterns

## Importing Components

Use the `@/components/application/*` alias for imports:

```typescript
import { NavItem, NavList } from "@/components/application/app-navigation/base-components/nav-item";
import { NavAccountCard } from "@/components/application/app-navigation/base-components/nav-account-card";
```

## Structure

Components are organized by their purpose:
- `app-navigation/` - Navigation components (NavItem, NavList, NavAccountCard, etc.)

## Difference from `base/` and `ui/`

- **`base/`** - Simple, atomic UI components from UntitledUI library (Button, Input, Tooltip)
  - Ready-made components, don't modify directly
  
- **`application/`** - Your custom complex, composed application components (Navigation, Sidebar)
  - Built by you using UntitledUI examples/patterns
  - You can modify and customize as needed
  
- **`ui/`** - Your custom simple project-specific components
  - Simple reusable components you create yourself

## Notes

- These are YOUR components - you create and modify them
- Use UntitledUI examples/patterns as reference, but adapt to your needs
- All components use the `@/components/application/*` import alias

