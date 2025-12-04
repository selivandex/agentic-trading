/** @format */

import type { Screen, DomElement } from "@/shared/api/generated/graphql";
import { DomElementKind } from "@/shared/api/generated/graphql";

/**
 * Re-export GraphQL types for entity model
 */
export type { Screen, DomElement };
export { DomElementKind };
export type { DomElementKind as DomElementKindType };

/**
 * Additional entity-specific types
 */
export interface CreateScreenData {
  name: string;
  projectId: string;
}

export interface UpdateScreenData {
  screenId: string;
  name?: string;
}


