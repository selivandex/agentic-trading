/** @format */

"use client";

import { useMemo } from "react";
import { useMutation } from "@apollo/client";
import { Crud } from "@/shared/ui/crud";
import { agentCrudConfig, type Agent } from "@/entities/agent";
import {
  SET_AGENT_ACTIVE_MUTATION,
  BATCH_DELETE_AGENTS_MUTATION,
} from "@/entities/agent";
import { logger } from "@/shared/lib";
import { Power01, FileCheck02, Trash01 } from "@untitledui/icons";

/**
 * AgentManager Component
 *
 * Enhanced Crud wrapper for Agents with custom actions
 */
export function AgentManager() {
  const [setAgentActive] = useMutation(SET_AGENT_ACTIVE_MUTATION);
  const [batchDeleteAgents] = useMutation(BATCH_DELETE_AGENTS_MUTATION);

  const enhancedConfig = useMemo(() => {
    // Action for activating agent
    const activateAction = {
      key: "activate",
      label: "Activate",
      icon: Power01,
      onClick: async (agent: Agent) => {
        try {
          await setAgentActive({
            variables: {
              id: agent.id,
              isActive: true,
            },
            refetchQueries: ["GetAllAgents"],
          });
          logger.info("Agent activated", { agentId: agent.id });
        } catch (error) {
          logger.error("Failed to activate agent", {
            error,
            agentId: agent.id,
          });
          throw error;
        }
      },
      hidden: (agent: Agent) => agent.isActive === true,
    };

    // Action for deactivating agent
    const deactivateAction = {
      key: "deactivate",
      label: "Deactivate",
      icon: Power01,
      onClick: async (agent: Agent) => {
        try {
          await setAgentActive({
            variables: {
              id: agent.id,
              isActive: false,
            },
            refetchQueries: ["GetAllAgents"],
          });
          logger.info("Agent deactivated", { agentId: agent.id });
        } catch (error) {
          logger.error("Failed to deactivate agent", {
            error,
            agentId: agent.id,
          });
          throw error;
        }
      },
      hidden: (agent: Agent) => agent.isActive === false,
    };

    // Action for viewing system prompt
    const viewPromptAction = {
      key: "view-prompt",
      label: "View System Prompt",
      icon: FileCheck02,
      onClick: async (agent: Agent) => {
        logger.info("View prompt clicked", { agentId: agent.id });
        // This could be enhanced with a modal showing the prompt
        alert(`System Prompt for ${agent.name}:\n\n${agent.systemPrompt}`);
      },
    };

    return {
      ...agentCrudConfig,
      enableSelection: true,

      // Custom row actions
      actions: [activateAction, deactivateAction, viewPromptAction],

      // Bulk actions
      bulkActions: [
        {
          key: "activate-selected",
          label: "Activate Selected",
          icon: Power01,
          onClick: async (agent: Agent) => {
            // Single action fallback
            await setAgentActive({
              variables: { id: agent.id, isActive: true },
              refetchQueries: ["GetAllAgents"],
            });
          },
          onBatchClick: async (agents: Agent[]) => {
            try {
              await Promise.all(
                agents
                  .filter((agent) => !agent.isActive)
                  .map((agent) =>
                    setAgentActive({
                      variables: {
                        id: agent.id,
                        isActive: true,
                      },
                      refetchQueries: ["GetAllAgents"],
                    })
                  )
              );
              logger.info("Bulk activated agents", { count: agents.length });
            } catch (error) {
              logger.error("Failed to bulk activate agents", { error });
              throw error;
            }
          },
        },
        {
          key: "deactivate-selected",
          label: "Deactivate Selected",
          icon: Power01,
          onClick: async (agent: Agent) => {
            // Single action fallback
            await setAgentActive({
              variables: { id: agent.id, isActive: false },
              refetchQueries: ["GetAllAgents"],
            });
          },
          onBatchClick: async (agents: Agent[]) => {
            try {
              await Promise.all(
                agents
                  .filter((agent) => agent.isActive)
                  .map((agent) =>
                    setAgentActive({
                      variables: {
                        id: agent.id,
                        isActive: false,
                      },
                      refetchQueries: ["GetAllAgents"],
                    })
                  )
              );
              logger.info("Bulk deactivated agents", { count: agents.length });
            } catch (error) {
              logger.error("Failed to bulk deactivate agents", { error });
              throw error;
            }
          },
        },
        {
          key: "bulk-delete",
          label: "Delete Selected",
          icon: Trash01,
          onClick: async (agent: Agent) => {
            // Single action fallback
            const confirmed = confirm(
              `Are you sure you want to delete this agent? This action cannot be undone.`
            );
            if (!confirmed) return;

            await batchDeleteAgents({
              variables: { ids: [agent.id] },
              refetchQueries: ["GetAllAgents"],
            });
          },
          onBatchClick: async (agents: Agent[]) => {
            const confirmed = confirm(
              `Are you sure you want to delete ${agents.length} agent(s)? This action cannot be undone.`
            );
            if (!confirmed) return;

            try {
              await batchDeleteAgents({
                variables: {
                  ids: agents.map((agent) => agent.id),
                },
                refetchQueries: ["GetAllAgents"],
              });
              logger.info("Bulk deleted agents", { count: agents.length });
            } catch (error) {
              logger.error("Failed to bulk delete agents", { error });
              throw error;
            }
          },
          destructive: true,
        },
      ],
    };
  }, [setAgentActive, batchDeleteAgents]);

  return <Crud config={enhancedConfig} />;
}
