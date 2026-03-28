import { client } from "@/shared/api/generated/client";
import type { DashboardSummary, SystemLogEntry, SystemStatus } from "@/types";

export const getSystemStatus = (): Promise<SystemStatus> => client.getSystemStatus();

export const getDashboardSummary = (): Promise<DashboardSummary> =>
  client.getDashboardSummary();

export async function getSystemLogs(limit = 200): Promise<SystemLogEntry[]> {
  const data = await client.getSystemLogs({ query: { limit } });
  return Array.isArray(data?.items) ? data.items : [];
}
