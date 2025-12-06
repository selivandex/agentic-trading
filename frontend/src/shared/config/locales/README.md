<!-- @format -->

# Locales

Translation files for internationalization.

## Quick Start

```tsx
import { useTranslation } from "@/shared/lib";

const { t } = useTranslation();
const message = t("auth.errors.invalidCredentials");
```

## Files

- `ru.ts` - Russian translations (default)
- `en.ts` - English translations
- `index.ts` - Exports and types

## Adding New Translation

1. Add to `ru.ts`:

```typescript
export const ru = {
  // ...
  myFeature: {
    title: "Мой заголовок",
  },
};
```

2. Add same structure to `en.ts`

3. Use in component:

```tsx
const { t } = useTranslation();
<h1>{t("myFeature.title")}</h1>;
```

See [full documentation](../../../docs/I18N_SYSTEM.md) for more details.
