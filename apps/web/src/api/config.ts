import { client } from "@/shared/api/generated/client";
import type { SetConfigRequestBody, SystemConfigValue } from "@/types";

export const getConfig = (key: string): Promise<SystemConfigValue> =>
  client.getConfig({ path: { key } });

export const setConfig = (
  key: string,
  value: SetConfigRequestBody["value"],
): Promise<SystemConfigValue> => client.setConfig({ path: { key }, body: { value } });
