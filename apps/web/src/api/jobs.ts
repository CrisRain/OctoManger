import { client } from "@/shared/api/generated/client";
import type {
  JobDefinition,
  JobDefinitionCreateInput,
  JobExecution,
  ListJobDefinitionsResponse,
  ListJobExecutionsResponse,
} from "@/types";

export const listJobDefinitions = (): Promise<ListJobDefinitionsResponse> =>
  client.listJobDefinitions();

export const listJobExecutions = (): Promise<ListJobExecutionsResponse> =>
  client.listJobExecutions();

export const createJobDefinition = (
  payload: JobDefinitionCreateInput,
): Promise<JobDefinition> => client.createJobDefinition({ body: payload });

export const enqueueJobExecution = (id: number): Promise<JobExecution> =>
  client.enqueueJobExecution({ path: { id } });
