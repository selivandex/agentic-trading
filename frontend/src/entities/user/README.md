# User Entity

User entity represents the core user data and settings in the trading platform.

## Structure

```
user/
├── api/              # GraphQL queries and hooks
├── lib/              # Utility functions and formatters
├── model/            # TypeScript types
├── ui/               # React components
└── index.ts          # Public API
```

## Usage

```tsx
import { useMe, UserCard, formatUserDisplayName } from "@/entities/user";

function UserProfile() {
  const { data, loading } = useMe();

  if (loading) return <div>Loading...</div>;
  if (!data?.me) return <div>Not authenticated</div>;

  return <UserCard user={data.me} showEmail showStatus />;
}
```

## API

### Hooks

- `useMe()` - Get current authenticated user
- `useUser(id)` - Get user by ID
- `useUserByTelegramID(telegramID)` - Get user by Telegram ID
- `useUpdateUserSettings()` - Update user settings
- `useSetUserActive()` - Set user active/inactive status

### Components

- `UserAvatar` - Display user avatar with initials
- `UserCard` - Display user information card

### Formatters

- `formatUserFullName(user)` - Format full name
- `formatUserDisplayName(user)` - Format display name with fallbacks
- `getUserInitials(user)` - Get user initials for avatar
- `formatRiskLevel(riskLevel)` - Format risk level display name
