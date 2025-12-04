<!-- @format -->

# Flowly Frontend

Visual workflow builder for creating automation scenarios in Flowly app.

## Tech Stack

- **React 19** + **TypeScript 5.8**
- **Next.js 16** (App Router)
- **Apollo Client 3.14** (GraphQL)
- **React Flow 11** (visual flow canvas)
- **Tailwind CSS 4** (styling)

## Getting Started

### Prerequisites

- Node.js 20+
- npm

### Installation

```bash
npm install
```

### Environment Variables

Create `.env.local` file:

```bash
# --------------------------
# Backend API
# --------------------------
# Backend GraphQL API URL (for server-side requests)
# Client-side requests use /api/graphql proxy
BACKEND_GRAPHQL_URL=http://api.lvh.me:3000/graphql

# --------------------------
# Authentication
# --------------------------
# Next-Auth Secret (generate with: openssl rand -base64 32)
NEXTAUTH_SECRET=your-secret-key-here
NEXTAUTH_URL=http://localhost:3001

# --------------------------
# Error Tracking (optional)
# --------------------------
# Sentry DSN for error tracking
# Get this from: https://sentry.io/settings/[org]/projects/[project]/keys/
NEXT_PUBLIC_SENTRY_DSN=https://your-key@sentry.apphud.com/project-id

# Sentry source maps upload (optional, only needed for production builds)
# Get auth token from: https://sentry.io/settings/account/api/auth-tokens/
SENTRY_AUTH_TOKEN=your-sentry-auth-token
SENTRY_ORG=apphud
SENTRY_PROJECT=flowly-frontend

# --------------------------
# Feature Flags (optional)
# --------------------------
# Enable Apollo Client debug logging
NEXT_PUBLIC_APOLLO_DEBUG=false
```

Generate NextAuth secret:

```bash
openssl rand -base64 32
```

**Note:** All variables prefixed with `NEXT_PUBLIC_` are exposed to the browser. Keep sensitive keys without this prefix.

### Development

```bash
npm run dev          # Start dev server on http://localhost:3001
npm run lint         # Run linter
npm run codegen      # Generate GraphQL types and hooks
npm run codegen:watch # Watch mode for codegen
```

### GraphQL Code Generation

The project uses GraphQL Code Generator to automatically generate TypeScript types and React hooks from the GraphQL schema.

**Configuration:** `codegen.yml`

**Schema location:** `../app/graphql/schema.graphql` (Rails backend)

**Generated files:** `src/shared/api/generated/graphql.ts`

See [GRAPHQL_CODEGEN_SETUP.md](./docs/GRAPHQL_CODEGEN_SETUP.md) for detailed documentation.

## Architecture

The project follows **Feature-Sliced Design (FSD)** methodology.

**Layers** (from bottom to top):

1. `src/shared/` - Reusable infrastructure (UI components, API clients, utilities)
2. `src/entities/` - Business entities (scenario, organization, project)
3. `src/features/` - User interactions (auth, scenario-builder)
4. `src/widgets/` - Large blocks (scenario-editor, project-layout)
5. `src/app/` - Next.js App Router (pages, layouts)

See [FSD_ARCHITECTURE.md](./docs/FSD_ARCHITECTURE.md) for details.

## GraphQL Usage

### Write Operation in api/ Layer

```typescript
// src/features/auth/api/auth.graphql.ts
import { gql } from "@apollo/client";

export const SIGN_IN_MUTATION = gql`
  mutation SignIn($input: SignInInput!) {
    signIn(input: $input) {
      accessToken
      user {
        id
        email
      }
    }
  }
`;
```

### Generate Types

```bash
yarn codegen
```

### Use Generated Hook

```typescript
import { useSignInMutation } from "@/shared/api/generated/graphql";

const [signIn, { loading, error }] = useSignInMutation();

await signIn({
  variables: {
    input: { email: "user@example.com", password: "password" },
  },
});
```

## Documentation

- [FSD Architecture](./docs/FSD_ARCHITECTURE.md)
- [GraphQL Codegen Setup](./docs/GRAPHQL_CODEGEN_SETUP.md)
- [GraphQL Setup & Usage](./docs/GRAPHQL_SETUP.md)
- [GraphQL Schema Sync](./docs/GRAPHQL_SCHEMA_SYNC.md)
- [Scenario Editor Usage](./docs/SCENARIO_EDITOR_USAGE.md)

## Project Structure

```
src/
  app/              # Next.js App Router
  entities/         # Business entities
  features/         # User interactions
  widgets/          # Large composite blocks
  shared/           # Shared infrastructure
    api/            # API clients
      generated/    # Generated GraphQL types
    base/           # Base UI components (UntitledUI)
    ui/             # Content UI components
    utils/          # Utilities
```

## License

Proprietary
