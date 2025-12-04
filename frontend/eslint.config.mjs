import { defineConfig, globalIgnores } from "eslint/config";
import nextVitals from "eslint-config-next/core-web-vitals";
import nextTs from "eslint-config-next/typescript";

const eslintConfig = defineConfig([
  ...nextVitals,
  ...nextTs,

  // Custom rules
  {
    rules: {
      // Allow unused vars/args if they start with _
      "@typescript-eslint/no-unused-vars": [
        "warn",
        {
          argsIgnorePattern: "^_",
          varsIgnorePattern: "^_",
          caughtErrorsIgnorePattern: "^_",
        },
      ],
      // Allow anonymous default exports (common pattern in config files)
      "import/no-anonymous-default-export": "off",
      // Allow refs in callbacks for Apollo Client setContext pattern
      // This is a false positive - we're creating a callback that reads ref LATER, not during render
      "react-hooks/refs": "off",
    },
  },

  // FSD Architecture Rules - Enforce layer dependencies
  // Exclude stories and tests from FSD rules
  {
    files: ["src/**/*.{ts,tsx}"],
    ignores: ["**/*.stories.{ts,tsx}", "**/*.test.{ts,tsx}", "**/__tests__/**"],
    rules: {
      "no-restricted-imports": [
        "error",
        {
          patterns: [
            {
              group: ["@/features/*", "@/widgets/*", "@/app/*"],
              message: "Shared layer cannot import from upper layers (features, widgets, app)",
            },
          ],
        },
      ],
    },
  },
  {
    files: ["src/shared/**/*.{ts,tsx}"],
    ignores: ["**/*.stories.{ts,tsx}", "**/*.test.{ts,tsx}", "**/__tests__/**"],
    rules: {
      "no-restricted-imports": [
        "error",
        {
          patterns: [
            {
              group: ["@/entities/*", "@/features/*", "@/widgets/*", "@/app/*"],
              message: "Shared layer cannot import from upper layers",
            },
          ],
        },
      ],
    },
  },
  {
    files: ["src/entities/**/*.{ts,tsx}"],
    ignores: ["**/*.stories.{ts,tsx}", "**/*.test.{ts,tsx}", "**/__tests__/**"],
    rules: {
      "no-restricted-imports": [
        "error",
        {
          patterns: [
            {
              group: ["@/features/*", "@/widgets/*", "@/app/*"],
              message: "Entities layer cannot import from upper layers (features, widgets, app)",
            },
            {
              group: ["@/entities/**/api/*", "@/entities/**/model/*", "@/entities/**/ui/*", "@/entities/**/lib/*"],
              message: "Import from entity's public API (index.ts), not from internal segments (api/, model/, ui/, lib/)",
            },
          ],
        },
      ],
    },
  },
  {
    files: ["src/features/**/*.{ts,tsx}"],
    ignores: ["**/*.stories.{ts,tsx}", "**/*.test.{ts,tsx}", "**/__tests__/**"],
    rules: {
      "no-restricted-imports": [
        "error",
        {
          patterns: [
            {
              group: ["@/widgets/*", "@/app/*"],
              message: "Features layer cannot import from upper layers (widgets, app)",
            },
            {
              group: ["@/entities/**/api/*", "@/entities/**/model/*", "@/entities/**/ui/*", "@/entities/**/lib/*"],
              message: "Import from entity's public API (index.ts), not from internal segments (api/, model/, ui/, lib/)",
            },
            {
              group: ["@/features/**/api/*", "@/features/**/model/*", "@/features/**/ui/*", "@/features/**/lib/*"],
              message: "Import from feature's public API (index.ts), not from internal segments (api/, model/, ui/, lib/)",
            },
          ],
        },
      ],
    },
  },
  {
    files: ["src/widgets/**/*.{ts,tsx}"],
    ignores: ["**/*.stories.{ts,tsx}", "**/*.test.{ts,tsx}", "**/__tests__/**"],
    rules: {
      "no-restricted-imports": [
        "error",
        {
          patterns: [
            {
              group: ["@/app/*"],
              message: "Widgets layer cannot import from app layer",
            },
            {
              group: ["@/entities/**/api/*", "@/entities/**/model/*", "@/entities/**/ui/*", "@/entities/**/lib/*"],
              message: "Import from entity's public API (index.ts), not from internal segments (api/, model/, ui/, lib/)",
            },
            {
              group: ["@/features/**/api/*", "@/features/**/model/*", "@/features/**/ui/*", "@/features/**/lib/*"],
              message: "Import from feature's public API (index.ts), not from internal segments (api/, model/, ui/, lib/)",
            },
          ],
        },
      ],
    },
  },
  {
    files: ["src/app/**/*.{ts,tsx}"],
    ignores: ["**/*.stories.{ts,tsx}", "**/*.test.{ts,tsx}", "**/__tests__/**"],
    rules: {
      "no-restricted-imports": [
        "error",
        {
          patterns: [
            {
              group: ["@/entities/**/api/*", "@/entities/**/model/*", "@/entities/**/ui/*", "@/entities/**/lib/*"],
              message: "Import from entity's public API (index.ts), not from internal segments (api/, model/, ui/, lib/)",
            },
            {
              group: ["@/features/**/api/*", "@/features/**/model/*", "@/features/**/ui/*", "@/features/**/lib/*"],
              message: "Import from feature's public API (index.ts), not from internal segments (api/, model/, ui/, lib/)",
            },
            {
              group: ["@/widgets/**/ui/*", "@/widgets/**/model/*"],
              message: "Import from widget's public API (index.ts), not from internal segments (ui/, model/)",
            },
          ],
        },
      ],
    },
  },

  // Allow <img> in UntitledUI base components
  {
    files: ["src/shared/base/**/*.tsx", "src/shared/base/**/*.ts"],
    rules: {
      "@next/next/no-img-element": "off",
    },
  },

  // Allow require() in CommonJS config files
  {
    files: ["**/*.cjs"],
    rules: {
      "@typescript-eslint/no-require-imports": "off",
    },
  },

  // Allow any in tests
  {
    files: ["**/__tests__/**/*", "**/*.test.{ts,tsx}", "**/*.spec.{ts,tsx}"],
    rules: {
      "@typescript-eslint/no-explicit-any": "off",
    },
  },

  // Override default ignores of eslint-config-next.
  globalIgnores([
    // Default ignores of eslint-config-next:
    ".next/**",
    "out/**",
    "build/**",
    "next-env.d.ts",
    // Ignore lago-front legacy code
    "lago-front/**",
  ]),
]);

export default eslintConfig;
