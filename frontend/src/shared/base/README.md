# Base Components

Base UI components from [UntitledUI](https://www.untitledui.com/) library.

## Adding Components

Components are added here automatically via:

```bash
npx untitledui@latest add <component-name>
```

For example:
```bash
npx untitledui@latest add input
npx untitledui@latest add button
npx untitledui@latest add tooltip
```

## Importing Components

Use the `@/components/base/*` alias for imports:

```typescript
import { Input, InputGroup } from "@/components/base/input/input";
import { Tooltip } from "@/components/base/tooltip/tooltip";
```

## Structure

Components are organized by their type:
- `input/` - Input components (Input, InputGroup, InputPayment, etc.)
- `tooltip/` - Tooltip components
- `button/` - Button components (when added)
- etc.

## Notes

- These components are from UntitledUI and should not be modified directly
- If customization is needed, create wrapper components in `@/shared/ui/`
- All components use the `@/components/base/*` import alias

