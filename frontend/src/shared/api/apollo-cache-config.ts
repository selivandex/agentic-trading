/**
 * Apollo Cache Configuration
 *
 * Configures InMemoryCache with type policies for:
 * - Relay-style pagination (cursor-based)
 * - Entity key fields
 * - Field merge strategies
 * - localStorage persistence with versioning
 *
 * Cache versioning:
 * - Increment CACHE_VERSION when changing GraphQL schema or cache structure
 * - Old cache is automatically cleared when version mismatch detected
 *
 * @format
 */

import { InMemoryCache } from "@apollo/client";
import { relayStylePagination } from "@apollo/client/utilities";

/**
 * Cache version
 * 
 * Increment this version when:
 * - GraphQL schema changes (added/removed/renamed fields)
 * - Cache type policies change
 * - Data structure incompatibility
 * 
 * When version changes, old cache is automatically invalidated.
 * 
 * Version history:
 * - v1: Initial implementation
 * - v2: Added cache key variables for organization/project separation
 * - v3: Fixed keyArgs for organization field to properly separate cache by org+project
 * - v4: Added screenId to macros keyArgs to cache macros per screen
 * - v5: Added proper cache separation by userId, organizationId, projectId for all queries
 */
export const CACHE_VERSION = 5;

/**
 * Cache configuration with type policies
 */
export const createApolloCache = () => {
  const cache = new InMemoryCache({
    typePolicies: {
      Query: {
        fields: {
          // Organization field - cache by organization/project combination
          // This ensures different org+project combinations are cached separately
          // Uses __cacheKey_* variables from useProjectContext to separate cache entries
          organization: {
            keyArgs: ['__cacheKey_organizationId', '__cacheKey_projectId'],
            merge(existing, incoming) {
              // Always replace with incoming data
              return incoming;
            },
          },

          // Relay-style pagination for scenarios - cache by scope and project_id
          scenarios: relayStylePagination(["scope", "__cacheKey_projectId"]),

          // Relay-style pagination for projects - cache by user_id and organization_id
          projects: relayStylePagination(["__cacheKey_userId", "__cacheKey_organizationId"]),

          // Relay-style pagination for organizations - cache by user_id
          organizations: relayStylePagination(["__cacheKey_userId"]),

          // Relay-style pagination for screens - cache by project_id
          screens: relayStylePagination(["scope", "__cacheKey_projectId"]),

          // Relay-style pagination for app users - cache by project_id
          appUsers: relayStylePagination(["__cacheKey_projectId"]),

          // Relay-style pagination for memberships - cache by organization_id
          memberships: relayStylePagination(["__cacheKey_organizationId"]),

          // Relay-style pagination for invitations - cache by status and organization_id
          invitations: relayStylePagination(["status", "__cacheKey_organizationId"]),

          // Relay-style pagination for api keys - cache by project_id
          apiKeys: relayStylePagination(["__cacheKey_projectId"]),

          // Relay-style pagination for languages - cache by project_id
          languages: relayStylePagination(["__cacheKey_projectId"]),

          // Relay-style pagination for macros - cache by screenId and project_id
          macros: relayStylePagination(["screenId", "__cacheKey_projectId"]),

          // Relay-style pagination for audiences - cache by project_id
          audiences: relayStylePagination(["__cacheKey_projectId"]),
        },
      },

      // Entity type policies
      User: {
        keyFields: ["id"],
      },

      Organization: {
        keyFields: ["id"],
        fields: {
          // Relay-style pagination for projects
          projects: relayStylePagination(),
        },
      },

      Project: {
        keyFields: ["id"],
        fields: {
          // Relay-style pagination for scenarios
          scenarios: relayStylePagination(),
        },
      },

      Scenario: {
        keyFields: ["id"],
        fields: {
          // Merge nodes array for flow - always replace with incoming
          nodes: {
            merge(_existing: unknown[] | undefined, incoming: unknown[]) {
              // Validate incoming is array
              if (!Array.isArray(incoming)) {
                console.error("Invalid nodes data:", incoming);
                return _existing ?? [];
              }
              return incoming;
            },
          },
          // Merge edges array for flow - always replace with incoming
          edges: {
            merge(_existing: unknown[] | undefined, incoming: unknown[]) {
              // Validate incoming is array
              if (!Array.isArray(incoming)) {
                console.error("Invalid edges data:", incoming);
                return _existing ?? [];
              }
              return incoming;
            },
          },
        },
      },

      Screen: {
        keyFields: ["id"],
      },

      Language: {
        keyFields: ["id"],
      },

      Macro: {
        keyFields: ["id"],
      },

      Membership: {
        keyFields: ["id"],
      },

      Invitation: {
        keyFields: ["id"],
      },

      ApiKey: {
        keyFields: ["id"],
      },

      AppUser: {
        keyFields: ["id"],
      },

      Audience: {
        keyFields: ["id"],
      },

      Role: {
        keyFields: ["key"], // Role uses 'key' as identifier
      },
    },
  });

  return cache;
};

/**
 * Initialize cache with localStorage persistence and versioning
 * 
 * Restores cache from localStorage on startup for instant UI.
 * Cache is automatically saved after each GraphQL operation.
 * 
 * Cache invalidation happens on:
 * - Version mismatch (CACHE_VERSION changed)
 * - User logout (clearCache called)
 * - Manual cache clear
 * - Background refetch updates stale data (cache-and-network policy)
 */
export const initializeCache = () => {
  const cache = createApolloCache();

  // Restore cache from localStorage on startup (client-side only)
  if (typeof window !== "undefined") {
    try {
      // Check cache version first
      const storedVersion = localStorage.getItem("apollo-cache-version");
      const currentVersion = String(CACHE_VERSION);
      
      if (storedVersion !== currentVersion) {
        // Version mismatch - clear old cache
        if (process.env.NODE_ENV === "development") {
          console.log(
            `🔄 Cache version mismatch (stored: ${storedVersion}, current: ${currentVersion}) - clearing cache`
          );
        }
        localStorage.removeItem("apollo-cache-persist");
        localStorage.setItem("apollo-cache-version", currentVersion);
      } else {
        // Version matches - restore cache
        const cachedData = localStorage.getItem("apollo-cache-persist");
        if (cachedData) {
          cache.restore(JSON.parse(cachedData));
          
          if (process.env.NODE_ENV === "development") {
            console.log(`✅ Apollo cache restored from localStorage (v${currentVersion})`);
          }
        }
      }
    } catch (error) {
      console.error("Failed to restore Apollo cache:", error);
      // Clear corrupted cache
      localStorage.removeItem("apollo-cache-persist");
      localStorage.removeItem("apollo-cache-version");
    }
  }

  if (process.env.NODE_ENV === "development") {
    console.log(`✅ Apollo cache initialized with localStorage persistence (v${CACHE_VERSION})`);
  }

  return cache;
};

