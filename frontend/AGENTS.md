<!-- @format -->

Use aliases for import

Use - Jest. Don't use vitest

Don't use console.log directly. Use ours - import { logger } from "@/shared/lib";

Use yarn

Always keep in mind - don't create GOD COMPONENT, after each iteration - ask yourself am i created god component? Do i use hooks for logic?

Naming Conventions

1. Directories:

   - Use kebab-case
   - Example: scenario-editor, flow-canvas, node-library

2. React Components (TSX files):

   - Use PascalCase
   - One component per file
   - Example: ScenarioEditor.tsx, ErrorBoundary.tsx

3. Hooks:

   - File name: kebab-case starting with "use-"
   - Hook name inside file: camelCase starting with "use"
   - Example:
     - File: use-sidebar-keyboard.ts
     - Export: export function useSidebarKeyboard() {}

4. Utilities / Helpers:

   - Use camelCase for file names
   - Example: formatNodeData.ts, validateSchema.ts

5. Models / State / Types:

   - Use camelCase for file names
   - Use PascalCase for exported types and classes
   - Example:
     - File: nodeSettingsModel.ts
     - Type: NodeSettings

6. FSD Layers (entities, features, widgets, pages):

   - Directory names: kebab-case
   - Each layer exports via an index.ts barrel

7. Test Files:
   - Use **tests** directories
   - Test files should mirror the filename of the unit they test
