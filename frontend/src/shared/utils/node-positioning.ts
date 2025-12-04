/** @format */

import type { Node as ReactFlowNode } from "reactflow";

/**
 * Utilities for calculating node positions on canvas
 *
 * Handles:
 * - Trigger positioning (center of canvas)
 * - Action positioning (below parent with offset)
 * - Placeholder positioning
 * - Branch positioning (for conditions)
 */

// -----------------------------------------------------------------------------
// Constants
// -----------------------------------------------------------------------------

/**
 * Default node dimensions
 */
export const NODE_DIMENSIONS = {
  TRIGGER: { width: 280, height: 100 },
  ACTION: { width: 260, height: 90 },
  PLACEHOLDER: { width: 280, height: 120 },
} as const;

/**
 * Spacing between nodes
 */
export const NODE_SPACING = {
  VERTICAL: 150, // Vertical gap between nodes
  HORIZONTAL: 200, // Horizontal gap for branches
  TRIGGER_TOP: 100, // Top margin for trigger
} as const;

// -----------------------------------------------------------------------------
// Position Calculators
// -----------------------------------------------------------------------------

/**
 * Calculate position for trigger node
 * Triggers are centered in the canvas
 * React Flow with fitView will auto-center this node in viewport
 */
export const calculateTriggerPosition = (): { x: number; y: number } => {
  return {
    x: 250, // Center in canvas coordinate space
    y: 100,
  };
};

/**
 * Calculate position for action node below parent
 *
 * @param parentNode - Parent node (React Flow Node)
 * @param siblingIndex - Index if multiple siblings (for branches)
 * @param totalSiblings - Total number of siblings (for centering branches)
 */
export const calculateActionPosition = (
  parentNode: ReactFlowNode,
  siblingIndex: number = 0,
  totalSiblings: number = 1
): { x: number; y: number } => {
  // Extract x/y from React Flow Node
  const parentX = parentNode.position.x;
  const parentY = parentNode.position.y;

  // Vertical position is always below parent
  const y = parentY + NODE_DIMENSIONS.ACTION.height + NODE_SPACING.VERTICAL;

  // Horizontal position depends on branching
  if (totalSiblings === 1) {
    // Single child - center below parent
    return {
      x: parentX,
      y,
    };
  }

  // Multiple children - distribute horizontally
  const totalWidth = (totalSiblings - 1) * NODE_SPACING.HORIZONTAL;
  const startX = parentX - totalWidth / 2;

  return {
    x: startX + siblingIndex * NODE_SPACING.HORIZONTAL,
    y,
  };
};

/**
 * Calculate position for placeholder
 *
 * @param parentNode - Parent node (if action placeholder) - React Flow Node
 * @param placeholderType - Type of placeholder
 */
export const calculatePlaceholderPosition = (
  parentNode?: ReactFlowNode,
  placeholderType: "trigger" | "action" = "trigger"
): { x: number; y: number } => {
  if (placeholderType === "trigger" || !parentNode) {
    // Trigger placeholder - same as trigger position
    return calculateTriggerPosition();
  }

  // Action placeholder - below parent
  return calculateActionPosition(parentNode);
};

/**
 * Calculate positions for multiple branch nodes (if/else, switch)
 *
 * @param parentNode - Parent condition node - React Flow Node
 * @param branchCount - Number of branches
 */
export const calculateBranchPositions = (
  parentNode: ReactFlowNode,
  branchCount: number
): Array<{ x: number; y: number; branchIndex: number }> => {
  const positions: Array<{ x: number; y: number; branchIndex: number }> = [];

  for (let i = 0; i < branchCount; i++) {
    const position = calculateActionPosition(parentNode, i, branchCount);
    positions.push({
      ...position,
      branchIndex: i,
    });
  }

  return positions;
};

/**
 * Calculate optimal position to avoid overlapping
 *
 * @param desiredPosition - Desired position
 * @param existingNodes - Existing nodes on canvas (React Flow Nodes)
 * @param padding - Minimum distance from other nodes
 */
export const calculateNonOverlappingPosition = (
  desiredPosition: { x: number; y: number },
  existingNodes: ReactFlowNode[],
  padding: number = 20
): { x: number; y: number } => {
  const position = { ...desiredPosition };
  let attempts = 0;
  const maxAttempts = 50;

  while (attempts < maxAttempts) {
    const overlaps = existingNodes.some((node) => {
      // Extract x/y from React Flow Node
      const nodeX = node.position.x;
      const nodeY = node.position.y;

      const distance = Math.sqrt(
        Math.pow(nodeX - position.x, 2) + Math.pow(nodeY - position.y, 2)
      );

      return distance < NODE_DIMENSIONS.ACTION.width + padding;
    });

    if (!overlaps) {
      break;
    }

    // Try offset position
    position.x += NODE_SPACING.HORIZONTAL / 4;
    position.y += NODE_SPACING.VERTICAL / 4;
    attempts++;
  }

  return position;
};

/**
 * Get bounding box for a node
 *
 * @param node - React Flow Node
 */
export const getNodeBounds = (
  node: ReactFlowNode
): { x: number; y: number; width: number; height: number } => {
  // Extract x/y from React Flow Node
  const x = node.position.x;
  const y = node.position.y;

  // Extract width/height from React Flow Node
  const width =
    (node.style?.width ? Number(node.style.width) : undefined) ||
    (node.width as number | undefined) ||
    NODE_DIMENSIONS.ACTION.width;
  const height =
    (node.style?.height ? Number(node.style.height) : undefined) ||
    (node.height as number | undefined) ||
    NODE_DIMENSIONS.ACTION.height;

  return {
    x,
    y,
    width,
    height,
  };
};

/**
 * Check if two nodes overlap
 *
 * @param node1 - First node (React Flow Node)
 * @param node2 - Second node (React Flow Node)
 */
export const doNodesOverlap = (
  node1: ReactFlowNode,
  node2: ReactFlowNode
): boolean => {
  const bounds1 = getNodeBounds(node1);
  const bounds2 = getNodeBounds(node2);

  return !(
    bounds1.x + bounds1.width < bounds2.x ||
    bounds2.x + bounds2.width < bounds1.x ||
    bounds1.y + bounds1.height < bounds2.y ||
    bounds2.y + bounds2.height < bounds1.y
  );
};
