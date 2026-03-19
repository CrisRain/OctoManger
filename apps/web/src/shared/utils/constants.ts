/**
 * 系统常量定义
 * 集中管理所有常量，便于维护
 */

/**
 * 作业执行状态
 */
export const JOB_EXECUTION_STATUS = {
  PENDING: "pending",
  RUNNING: "running",
  DONE: "done",
  SUCCESS: "success",
  FAILED: "failed",
  CANCELLED: "cancelled",
} as const;

export type JobExecutionStatus = (typeof JOB_EXECUTION_STATUS)[keyof typeof JOB_EXECUTION_STATUS];

/**
 * 作业执行状态显示配置
 */
export const JOB_EXECUTION_STATUS_CONFIG: Record<
  JobExecutionStatus,
  { label: string; color: string; icon?: string }
> = {
  pending: { label: "等待中", color: "gray" },
  running: { label: "执行中", color: "blue" },
  done: { label: "已完成", color: "cyan" },
  success: { label: "成功", color: "green" },
  failed: { label: "失败", color: "red" },
  cancelled: { label: "已取消", color: "gray" },
};

/**
 * Agent状态
 */
export const AGENT_STATE = {
  STOPPED: "stopped",
  STARTING: "starting",
  RUNNING: "running",
  STOPPING: "stopping",
  ERROR: "error",
} as const;

export type AgentState = (typeof AGENT_STATE)[keyof typeof AGENT_STATE];

/**
 * Agent状态显示配置
 */
export const AGENT_STATE_CONFIG: Record<
  AgentState,
  { label: string; color: string }
> = {
  stopped: { label: "已停止", color: "gray" },
  starting: { label: "启动中", color: "blue" },
  running: { label: "运行中", color: "green" },
  stopping: { label: "停止中", color: "orange" },
  error: { label: "错误", color: "red" },
};

/**
 * 账号状态
 */
export const ACCOUNT_STATUS = {
  ACTIVE: "active",
  INACTIVE: "inactive",
  ERROR: "error",
} as const;

export type AccountStatus = (typeof ACCOUNT_STATUS)[keyof typeof ACCOUNT_STATUS];

/**
 * 账号状态显示配置
 */
export const ACCOUNT_STATUS_CONFIG: Record<
  AccountStatus,
  { label: string; color: string }
> = {
  active: { label: "活跃", color: "green" },
  inactive: { label: "停用", color: "gray" },
  error: { label: "错误", color: "red" },
};

/**
 * 邮箱提供商
 */
export const EMAIL_PROVIDER = {
  GMAIL: "gmail",
  OUTLOOK: "outlook",
  IMAP: "imap",
} as const;

export type EmailProvider = (typeof EMAIL_PROVIDER)[keyof typeof EMAIL_PROVIDER];

/**
 * 邮箱提供商显示配置
 */
export const EMAIL_PROVIDER_CONFIG: Record<
  EmailProvider,
  { label: string; icon?: string }
> = {
  gmail: { label: "Gmail" },
  outlook: { label: "Outlook" },
  imap: { label: "IMAP" },
};

/**
 * 分页默认值
 */
export const PAGINATION = {
  DEFAULT_PAGE_SIZE: 20,
  PAGE_SIZE_OPTIONS: [10, 20, 50, 100],
} as const;

/**
 * 刷新间隔（毫秒）
 */
export const REFRESH_INTERVALS = {
  FAST: 3000, // 3秒 - 实时数据
  NORMAL: 10000, // 10秒 - 一般数据
  SLOW: 30000, // 30秒 - 慢速数据
} as const;

/**
 * 本地存储键
 */
export const STORAGE_KEYS = {
  ADMIN_KEY: "octo_admin_key",
  THEME: "octo_theme",
  SIDEBAR_COLLAPSED: "octo_sidebar_collapsed",
} as const;

/**
 * 时间格式
 */
export const DATE_FORMATS = {
  FULL: "YYYY-MM-DD HH:mm:ss",
  DATE: "YYYY-MM-DD",
  TIME: "HH:mm:ss",
  MONTH: "YYYY-MM",
} as const;

/**
 * 定时任务Cron预设
 */
export const CRON_PRESETS = [
  { label: "每分钟", value: "* * * * *" },
  { label: "每5分钟", value: "*/5 * * * *" },
  { label: "每15分钟", value: "*/15 * * * *" },
  { label: "每30分钟", value: "*/30 * * * *" },
  { label: "每小时", value: "0 * * * *" },
  { label: "每天0点", value: "0 0 * * *" },
  { label: "每周一0点", value: "0 0 * * 1" },
  { label: "每月1号0点", value: "0 0 1 * *" },
] as const;

/**
 * 常用时区
 */
export const COMMON_TIMEZONES = [
  { label: "北京时间 (UTC+8)", value: "Asia/Shanghai" },
  { label: "UTC (UTC+0)", value: "UTC" },
  { label: "东京时间 (UTC+9)", value: "Asia/Tokyo" },
  { label: "纽约时间 (UTC-5)", value: "America/New_York" },
  { label: "伦敦时间 (UTC+0)", value: "Europe/London" },
] as const;

/**
 * 表格操作列宽度
 */
export const TABLE_COLUMN_WIDTHS = {
  INDEX: 60,
  STATUS: 100,
  ACTIONS: 150,
  DATE: 180,
} as const;

/**
 * 动画时长（毫秒）
 */
export const ANIMATION_DURATION = {
  FAST: 150,
  NORMAL: 300,
  SLOW: 500,
} as const;
