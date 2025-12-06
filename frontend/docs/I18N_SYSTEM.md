<!-- @format -->

# Internationalization (i18n) System

Simple and lightweight i18n system for managing translations without external dependencies.

## 📁 Structure

```
src/shared/
├── config/
│   └── locales/
│       ├── ru.ts         # Russian translations
│       ├── en.ts         # English translations
│       └── index.ts      # Exports
└── lib/
    └── use-translation.ts # Translation hook
```

## 🚀 Usage

### Basic Usage in Components

```tsx
import { useTranslation } from "@/shared/lib";

export const MyComponent = () => {
  const { t } = useTranslation();

  return (
    <div>
      <h1>{t("auth.login.title")}</h1>
      <p>{t("auth.errors.invalidCredentials")}</p>
    </div>
  );
};
```

### With Parameters

```tsx
const { t } = useTranslation();

// Translation with parameters
const message = t("validation.minLength", { min: 8 });
// Result: "Минимальная длина: 8 символов"
```

### Different Locale

```tsx
const { t } = useTranslation("en");

const message = t("auth.login.title");
// Result: "Sign in" (English)
```

### Outside React Components

```typescript
import { getTranslation } from "@/shared/lib";
import { ru } from "@/shared/config/locales/ru";

const message = getTranslation(ru, "auth.errors.serverError");
// Result: "Ошибка сервера. Попробуйте позже."
```

## 📝 Adding New Translations

### 1. Add to Russian (`ru.ts`)

```typescript
export const ru = {
  // ... existing translations
  myFeature: {
    title: "Мой заголовок",
    description: "Моё описание",
    buttons: {
      save: "Сохранить",
      cancel: "Отмена",
    },
  },
} as const;
```

### 2. Add to English (`en.ts`)

```typescript
export const en = {
  // ... existing translations
  myFeature: {
    title: "My Title",
    description: "My Description",
    buttons: {
      save: "Save",
      cancel: "Cancel",
    },
  },
} as const;
```

### 3. Use in Component

```tsx
const { t } = useTranslation();

return (
  <div>
    <h1>{t("myFeature.title")}</h1>
    <p>{t("myFeature.description")}</p>
    <Button>{t("myFeature.buttons.save")}</Button>
  </div>
);
```

## 🎯 Best Practices

### 1. Organize by Feature

Group translations by feature/module:

```typescript
{
  auth: { /* auth-related texts */ },
  dashboard: { /* dashboard-related texts */ },
  settings: { /* settings-related texts */ },
}
```

### 2. Use Nested Structure

Use nested objects for better organization:

```typescript
{
  auth: {
    login: { /* login form texts */ },
    register: { /* register form texts */ },
    errors: { /* auth error messages */ },
  }
}
```

### 3. Consistent Naming

- Use camelCase for keys
- Use descriptive names
- Group related items together

```typescript
// ✅ Good
{
  auth: {
    errors: {
      invalidCredentials: "...",
      serverError: "...",
    }
  }
}

// ❌ Bad
{
  authInvalidCreds: "...",
  authServerErr: "...",
}
```

### 4. Reuse Common Texts

Use `common` section for frequently used texts:

```typescript
{
  common: {
    loading: "Загрузка...",
    error: "Ошибка",
    save: "Сохранить",
    cancel: "Отмена",
  }
}
```

## 🔄 Migration from Hardcoded Texts

### Before

```tsx
export const MyComponent = () => {
  return (
    <div>
      <h1>Вход в систему</h1>
      <p>Неверный email или пароль</p>
    </div>
  );
};
```

### After

```tsx
export const MyComponent = () => {
  const { t } = useTranslation();

  return (
    <div>
      <h1>{t("auth.login.title")}</h1>
      <p>{t("auth.errors.invalidCredentials")}</p>
    </div>
  );
};
```

## 🌐 Supported Locales

- `ru` - Russian (default)
- `en` - English

To add more locales, create a new file in `src/shared/config/locales/` and update the `locales` object in `index.ts`.

## 🔍 API Reference

### `useTranslation(locale?)`

Hook for translations in React components.

**Parameters:**

- `locale` (optional): Locale to use. Defaults to `DEFAULT_LOCALE` ("ru").

**Returns:**

```typescript
{
  t: (key: string, params?: Record<string, string | number>) => string,
  locale: Locale,
  translations: Translations
}
```

### `getTranslation(translations, key, params?)`

Function for translations outside React components.

**Parameters:**

- `translations`: Translation object (e.g., `ru` or `en`)
- `key`: Dot-notation key path
- `params` (optional): Parameters for string interpolation

**Returns:** Translated string

## 📦 Example: Complete Feature

```typescript
// 1. Add translations to ru.ts
export const ru = {
  // ...
  profile: {
    title: "Профиль",
    editButton: "Редактировать",
    saveButton: "Сохранить",
    errors: {
      updateFailed: "Не удалось обновить профиль",
    },
    success: {
      updated: "Профиль успешно обновлен",
    },
  },
};

// 2. Use in component
import { useTranslation } from "@/shared/lib";

export const ProfilePage = () => {
  const { t } = useTranslation();

  return (
    <div>
      <h1>{t("profile.title")}</h1>
      <Button>{t("profile.editButton")}</Button>
    </div>
  );
};
```

## 🚨 Error Handling

If a key is not found, the hook returns the key itself as a fallback:

```tsx
const { t } = useTranslation();

console.log(t("non.existent.key"));
// Output: "non.existent.key"
```

This makes it easy to spot missing translations during development.

---

**Note:** This is a lightweight custom solution. For complex multi-language applications with locale detection, consider using `next-intl` or `react-i18next`.
