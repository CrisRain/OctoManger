/**
 * Composables统一导出
 * 提供所有组合式函数的集中访问
 */

// 消息提示
export * from "./useMessage";

// 错误处理
export * from "./useErrorHandler";

// 确认对话框
export * from "./useConfirm";

// 原有composables
export { useSystemStatus, useDashboardSnapshot, useSystemLogs, type DashboardSummary } from "./useDashboard";
export { useAccountTypes } from "./useAccountTypes";
export { useAccounts } from "./useAccounts";
export { useAgents } from "./useAgents";
export { useEmailAccounts } from "./useEmailAccounts";
export { useEmailAccountsList } from "./useEmailAccountsList";
export { useSystemConfigs } from "./useSystemConfigs";
export * from "./useJobs";
export { usePlugins } from "./usePlugins";
export { useTriggers } from "./useTriggers";
export { useEventStream } from "./useEventStream";
