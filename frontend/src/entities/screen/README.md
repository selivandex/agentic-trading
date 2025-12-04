# Screen Entity

Business entity representing application screens.

## Structure

- `api/` - GraphQL operations and React hooks
- `model/` - TypeScript types and business logic
- `ui/` - Reusable UI components for displaying screens
- `lib/` - Utility functions

## Usage

```typescript
import { useScreens, useCreateScreen } from "@/entities/screen/api";
import type { Screen, DomElement } from "@/entities/screen/model";
```

## GraphQL Operations

- `GetScreens` - List screens with pagination
- `GetScreenDomElements` - List DOM elements for screen
- `CreateScreen` - Create new screen
- `UpdateScreen` - Update screen
- `DeleteScreen` - Delete screen
- `ArchiveScreen` - Archive screen
- `UnarchiveScreen` - Unarchive screen
- `PublishScreen` - Publish screen




