// 状态统一映射库 — 所有状态的中文标签、颜色和样式 class

/** 状态值 → 中文显示标签 */
export const STATUS_LABEL: Record<string, string> = {
  // 通用
  active: "已启用",
  inactive: "已停用",
  pending: "等待中",

  // 任务 / 执行
  running: "运行中",
  done: "已完成",
  success: "已完成",
  failed: "已失败",
  cancelled: "已取消",
  error: "错误",

  // Agent
  stopping: "停止中",
  stopped: "已停止",
};

/** 状态值 → UI 标签颜色属性 */
export const STATUS_COLOR: Record<string, string> = {
  active: "green",
  inactive: "gray",
  pending: "gray",
  running: "blue",
  done: "green",
  success: "green",
  failed: "red",
  cancelled: "gray",
  error: "red",
  stopping: "blue",
  stopped: "gray",
};

/** 状态值 → 状态点 CSS class（对应全局 .status-dot-large 变体） */
export const STATUS_DOT_CLASS: Record<string, string> = {
  active: "online",
  inactive: "neutral",
  pending: "neutral",
  running: "running",
  done: "online",
  success: "online",
  failed: "offline",
  cancelled: "neutral",
  error: "offline",
  stopping: "neutral",
  stopped: "neutral",
};

export function getStatusLabel(status: string): string {
  return STATUS_LABEL[status] ?? status;
}

export function getStatusColor(status: string): string {
  return STATUS_COLOR[status] ?? "gray";
}

export function getStatusDotClass(status: string): string {
  return STATUS_DOT_CLASS[status] ?? "neutral";
}
