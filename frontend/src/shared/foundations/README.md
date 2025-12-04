# Foundation Components

Foundation components from [UntitledUI](https://www.untitledui.com/) library, such as icons and other foundational elements.

## Adding Components

Components are added here automatically via:

```bash
npx untitledui@latest add <component-name>
```

For example:
```bash
npx untitledui@latest add payment-icons
```

## Importing Components

Use the `@/components/foundations/*` alias for imports:

```typescript
import { VisaIcon, MastercardIcon, AmexIcon } from "@/components/foundations/payment-icons";
```

## Structure

- `payment-icons/` - Payment method icons (Visa, Mastercard, Amex, etc.)

## Notes

- These components are from UntitledUI and should not be modified directly
- If customization is needed, create wrapper components in `@/shared/ui/`
- All components use the `@/components/foundations/*` import alias

