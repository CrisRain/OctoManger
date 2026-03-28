import { client } from "@/shared/api/generated/client";
import type {
  ExecutePluginActionResult,
  ListPluginsResponse,
  Plugin,
  PluginRuntimeConfigInput,
  PluginRuntimeConfig,
  PluginSyncResult,
} from "@/types";

export const listPlugins = (): Promise<ListPluginsResponse> => client.listPlugins();

export const getPlugin = (key: string): Promise<Plugin> =>
  client.getPlugin({ path: { key } });

export const getPluginSettings = (key: string): Promise<Record<string, unknown>> =>
  client.getPluginSettings({ path: { key } });

export const updatePluginSettings = (
  key: string,
  values: Record<string, unknown>,
): Promise<{ saved: boolean }> =>
  client.putPluginSettings({ path: { key }, body: values });

export const getPluginRuntimeConfig = (key: string): Promise<PluginRuntimeConfig> =>
  client.getPluginRuntimeConfig({ path: { key } });

export const updatePluginRuntimeConfig = (
  key: string,
  value: PluginRuntimeConfigInput,
): Promise<PluginRuntimeConfig> =>
  client.putPluginRuntimeConfig({ path: { key }, body: value });

export const syncPlugins = (): Promise<PluginSyncResult> => client.syncPlugins();

export const executePluginAction = (
  key: string,
  action: string,
  params?: Record<string, unknown>,
  spec?: Record<string, unknown>,
  account?: { id?: number; identifier?: string },
): Promise<ExecutePluginActionResult> =>
  client.executePluginAction({ path: { key, action }, body: { params, spec, account } });
