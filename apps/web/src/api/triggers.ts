import { client } from "@/shared/api/generated/client";
import type {
  ListTriggersResponse,
  TriggerCreateInput,
  TriggerFireInput,
  TriggerFireResult,
} from "@/types";

export const listTriggers = (): Promise<ListTriggersResponse> => client.listTriggers();

export const createTrigger = (payload: TriggerCreateInput) =>
  client.createTrigger({ body: payload });

export const deleteTrigger = (id: number) =>
  client.deleteTrigger({ path: { id } });

export const fireTrigger = (
  id: number,
  payload?: TriggerFireInput,
): Promise<TriggerFireResult> =>
  client.fireTrigger({ path: { id }, body: payload });
