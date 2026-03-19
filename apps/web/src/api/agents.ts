import { client } from "@/shared/api/generated/client";
import type { AgentCreateInput, AgentStatus, ListAgentsResponse } from "@/types";

export const listAgents = (): Promise<ListAgentsResponse> => client.listAgents();

export const createAgent = (payload: AgentCreateInput) =>
  client.createAgent({ body: payload });

export const startAgent = (id: number) => client.startAgent({ path: { id } });

export const stopAgent = (id: number) => client.stopAgent({ path: { id } });

export const getAgentStatus = (id: number): Promise<AgentStatus> =>
  client.getAgentStatus({ path: { id } });
