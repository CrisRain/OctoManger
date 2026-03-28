import { client } from "@/shared/api/generated/client";
import type { SetupInitializeResult, SetupStatus, SystemConfig } from "@/types";

export const getSetupStatus = (): Promise<SetupStatus> => client.getSetupStatus();

export const initializeSetup = (value: SystemConfig): Promise<SetupInitializeResult> =>
  client.initializeSetup({ body: value });
