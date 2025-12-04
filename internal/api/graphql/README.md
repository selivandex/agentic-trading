# GraphQL API

Schema-first GraphQL API implementation using [gqlgen](https://gqlgen.com/).

## Architecture

Following Clean Architecture principles:

```
GraphQL Layer (Presentation)
    ↓
Services Layer (Application)
    ↓
Domain Layer
    ↓
Repository Layer (Infrastructure)
```

**Key principles:**
- Resolvers use **Services**, not Repositories directly
- JWT authentication via HTTP-only cookies (1 year TTL)
- All custom scalars properly marshaled (UUID, Time, Decimal, JSONObject)

## Structure

```
internal/api/graphql/
├── schema/              # GraphQL schemas (*.graphql)
│   ├── scalars.graphql  # Custom scalar definitions
│   ├── auth.graphql     # Authentication (register, login, logout)
│   ├── user.graphql     # User queries & mutations
│   ├── strategy.graphql # Strategy queries & mutations
│   └── fund_watchlist.graphql # Fund watchlist queries & mutations
├── generated/           # Auto-generated code (by gqlgen)
│   ├── generated.go     # GraphQL server implementation
│   ├── models.go        # GraphQL models
│   └── scalars.go       # Custom scalar marshalers (DO NOT EDIT - hand-written)
├── resolvers/           # Resolver implementations
│   ├── resolver.go      # Root resolver with DI
│   ├── auth.resolvers.go
│   ├── user.resolvers.go
│   ├── strategy.resolvers.go
│   └── fund_watchlist.resolvers.go
├── middleware/          # HTTP middleware
│   ├── auth.go          # JWT authentication from cookies
│   └── auth_test.go
└── handler.go           # GraphQL HTTP handler setup
```

## Development

### Generate GraphQL code

After modifying schemas:

```bash
make g-gen
```

This will:
1. Read `*.graphql` schemas
2. Generate Go code in `generated/`
3. Scaffold missing resolvers in `resolvers/`

**Important:** `generated/scalars.go` is hand-written and contains custom scalar marshalers. It will NOT be overwritten.

### Endpoints

| Endpoint | Description |
|----------|-------------|
| `/graphql` | GraphQL API endpoint |
| `/playground` | GraphQL Playground (dev only) |

### Authentication

All mutations and protected queries use JWT from HTTP-only cookie:

1. Client calls `register` or `login` mutation
2. Server returns JWT token in response AND sets HTTP-only cookie
3. Client stores token in Next.js session
4. All subsequent requests include cookie automatically
5. Middleware extracts token from cookie and validates

**Cookie details:**
- Name: `auth_token`
- HttpOnly: `true`
- Secure: `true` (production only)
- SameSite: `Strict`
- MaxAge: 1 year (31,536,000 seconds)

## Schema Overview

### Authentication

```graphql
mutation {
  register(input: {
    email: "user@example.com"
    password: "secure123"
    firstName: "John"
    lastName: "Doe"
  }) {
    token
    user { id email }
  }
}

mutation {
  login(input: {
    email: "user@example.com"
    password: "secure123"
  }) {
    token
    user { id email }
  }
}

query {
  me {
    id
    email
    firstName
    settings { riskLevel maxPositions }
  }
}
```

### User Management

```graphql
query {
  user(id: "uuid") {
    id
    email
    telegramID
    settings {
      maxPositionSizeUsd
      circuitBreakerOn
    }
  }
}

mutation {
  updateUserSettings(userID: "uuid", input: {
    riskLevel: "aggressive"
    maxPositions: 5
  }) {
    id
    settings { riskLevel }
  }
}
```

### Strategy

```graphql
query {
  userStrategies(userID: "uuid", status: ACTIVE) {
    id
    name
    allocatedCapital
    currentEquity
    totalPnL
    totalPnLPercent
  }
}

mutation {
  createStrategy(userID: "uuid", input: {
    name: "Crypto Portfolio"
    description: "Long-term crypto holdings"
    allocatedCapital: "10000.00"
    marketType: SPOT
    riskTolerance: MODERATE
    rebalanceFrequency: WEEKLY
  }) {
    id
    status
  }
}
```

### Fund Watchlist

```graphql
query {
  monitoredSymbols(marketType: "spot") {
    id
    symbol
    category
    tier
    lastAnalyzedAt
  }
}

mutation {
  createFundWatchlist(input: {
    symbol: "BTC/USDT"
    marketType: "spot"
    category: "major"
    tier: 1
  }) {
    id
    symbol
  }
}
```

## Testing

Run all GraphQL-related tests:

```bash
# JWT service tests
go test ./pkg/auth/... -v

# Auth service tests
go test ./internal/services/auth/... -v

# Middleware tests
go test ./internal/api/graphql/middleware/... -v

# Fund watchlist service tests
go test ./internal/services/fundwatchlist/... -v
```

## Configuration

Add to `.env`:

```bash
# JWT Configuration
JWT_SECRET=your-secret-key-min-32-characters-long-please
JWT_ISSUER=prometheus
JWT_TTL=8760h  # 1 year
```

Generate secret:

```bash
make gen-encryption-key
```

## Database Migration

Migration `019` adds email/password auth fields to users table:

```bash
make migrate-up
```

This adds:
- `email` VARCHAR(255) UNIQUE
- `password_hash` VARCHAR(255)
- Makes `telegram_id` nullable
- Adds constraint: user must have EITHER telegram_id OR email+password

## Next.js Integration

### Server-side in Next.js

```typescript
// In Next.js server action or API route
const response = await fetch('http://backend:8080/graphql', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Cookie': cookies().toString() // Forward cookies from client
  },
  body: JSON.stringify({
    query: `mutation Login($input: LoginInput!) {
      login(input: $input) {
        token
        user { id email }
      }
    }`,
    variables: { input: { email, password } }
  })
});

// Extract Set-Cookie from response and forward to client
const setCookie = response.headers.get('set-cookie');
```

### Client-side (Apollo Client)

```typescript
import { ApolloClient, InMemoryCache, HttpLink } from '@apollo/client';

const client = new ApolloClient({
  link: new HttpLink({
    uri: '/api/graphql', // Proxy через Next.js API route
    credentials: 'include' // Include cookies
  }),
  cache: new InMemoryCache()
});
```

## Security Notes

✅ **What's implemented:**
- Bcrypt password hashing (cost: 10)
- HTTP-only cookies (XSS protection)
- SameSite=Strict (CSRF protection)
- Secure flag in production
- 1-year token expiration
- Token validation on every request
- User deactivation check

⚠️ **TODO for production:**
- Rate limiting on auth endpoints
- CORS configuration
- HTTPS enforcement
- Token rotation mechanism (optional)
- 2FA support (optional)
