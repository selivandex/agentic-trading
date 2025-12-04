# Features Layer

User interactions and business actions that represent specific functionality.

## Structure

Each feature should have its own folder with:
- **ui/** - Feature-specific UI components
- **model/** - Feature state management, types
- **api/** - Feature-specific API calls
- **lib/** - Feature-specific business logic

## Example Structure

```
features/
  auth/
    ui/
      LoginForm.tsx
      SignUpForm.tsx
    model/
      authStore.ts
    api/
      authApi.ts
  cart/
    ui/
      AddToCartButton.tsx
      CartItem.tsx
    model/
      cartStore.ts
```

## Rules

- ✅ Can import from shared and entities layers
- ❌ Cannot import from widgets or pages
- ✅ Should represent user actions and interactions
