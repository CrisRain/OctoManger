import { client } from "@/shared/api/generated/client";
import type { DashboardSummary, SystemStatus } from "@/types";

export const getSystemStatus = (): Promise<SystemStatus> => client.getSystemStatus();

export const getDashboardSummary = (): Promise<DashboardSummary> =>
  client.getDashboardSummary();
