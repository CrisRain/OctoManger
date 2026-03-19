import { apiRoutes } from "@/shared/api/generated/client";

export const getJobExecutionEventsUrl = (id: number): string =>
  apiRoutes.streamJobExecutionEvents({ path: { id } });

export const getAgentEventsUrl = (id: number): string =>
  apiRoutes.streamAgentEvents({ path: { id } });
