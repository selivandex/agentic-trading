# Entities Layer

Business entities that represent core domain models.

## Structure

Each entity should have its own folder with:
- **ui/** - Entity-specific UI components
- **model/** - Entity types, interfaces, and data structures
- **api/** - Entity-specific API calls
- **lib/** - Entity-specific business logic

## Example Structure

```
entities/
  user/
    ui/
      UserCard.tsx
      UserAvatar.tsx
    model/
      types.ts
      store.ts
    api/
      userApi.ts
  product/
    ui/
      ProductCard.tsx
    model/
      types.ts
```

## Rules

- ✅ Can import from shared layer
- ❌ Cannot import from features, widgets, or pages
- ✅ Should represent independent business entities
