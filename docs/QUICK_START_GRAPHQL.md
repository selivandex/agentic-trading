# Quick Start: GraphQL API

## 1. Setup

### Add to `.env`:

```bash
# Generate secret
JWT_SECRET=$(make gen-encryption-key)
JWT_ISSUER=prometheus
JWT_TTL=8760h
```

### Run migration:

```bash
make migrate-up
```

## 2. Start Server

```bash
make run
```

## 3. Test GraphQL API

Open GraphQL Playground:
```
http://localhost:8080/playground
```

### Register User

```graphql
mutation {
  register(input: {
    email: "test@example.com"
    password: "password123"
    firstName: "Test"
    lastName: "User"
  }) {
    token
    user {
      id
      email
      settings { riskLevel }
    }
  }
}
```

### Login

```graphql
mutation {
  login(input: {
    email: "test@example.com"
    password: "password123"
  }) {
    token
    user { id email }
  }
}
```

### Get Current User

```graphql
query {
  me {
    id
    email
    firstName
    strategies {
      id
      name
      totalPnL
    }
  }
}
```

## 4. Frontend Integration

### Apollo Client Setup (Next.js)

```typescript
// lib/apollo-client.ts
import { ApolloClient, InMemoryCache, HttpLink } from '@apollo/client';

const client = new ApolloClient({
  link: new HttpLink({
    uri: process.env.NEXT_PUBLIC_GRAPHQL_URL || 'http://localhost:8080/graphql',
    credentials: 'include' // Important: sends cookies
  }),
  cache: new InMemoryCache()
});

export default client;
```

### Usage in Component

```typescript
import { gql, useMutation, useQuery } from '@apollo/client';

const LOGIN = gql`
  mutation Login($input: LoginInput!) {
    login(input: $input) {
      token
      user {
        id
        email
      }
    }
  }
`;

const ME = gql`
  query Me {
    me {
      id
      email
      firstName
    }
  }
`;

function LoginForm() {
  const [login] = useMutation(LOGIN);
  const { data } = useQuery(ME);

  const handleLogin = async (email: string, password: string) => {
    const result = await login({
      variables: { input: { email, password } }
    });
    // Token automatically stored in HTTP-only cookie
    // No need to manually save it
  };
}
```

## Commands

```bash
make g-gen        # Regenerate GraphQL code
make build        # Build binary
make run          # Run server
make test         # Run all tests
make lint         # Run linter
```

## Endpoints

- `/graphql` - GraphQL API
- `/playground` - GraphQL Playground (dev only)
- `/health` - Health check
- `/metrics` - Prometheus metrics

