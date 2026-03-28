import { client } from "@/shared/api/generated/client";
import type { SystemConfig } from "@/types";

export const getSystemConfig = (): Promise<SystemConfig> => client.getConfig();

export const updateSystemConfig = (
  value: SystemConfig,
): Promise<SystemConfig> => client.putConfig({ body: value });
