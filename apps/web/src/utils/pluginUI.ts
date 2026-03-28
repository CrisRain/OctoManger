import type { Account, PluginUIButton, PluginUIFormField } from "@/types";

export interface PluginAccountReference {
  id: number;
  identifier: string;
}

export interface ResolvedPluginUIButtonInput {
  params: Record<string, unknown>;
  account?: PluginAccountReference;
}

interface ResolvePluginUIButtonOptions {
  bindAccountContext?: boolean;
}

function normalizeFieldType(field: PluginUIFormField): string {
  return String(field.type ?? "string").trim().toLowerCase() || "string";
}

function normalizeFieldLabel(field: PluginUIFormField): string {
  const label = String(field.label ?? "").trim();
  if (label) {
    return label;
  }
  return humanizeFieldName(field.name);
}

function humanizeFieldName(name: string): string {
  const compact = String(name).trim();
  if (!compact) {
    return "";
  }

  const words = compact
    .replace(/([a-z0-9])([A-Z])/g, "$1 $2")
    .replace(/[_-]+/g, " ")
    .trim()
    .split(/\s+/)
    .filter(Boolean)
    .map((word) => {
      const upper = word.toUpperCase();
      if (["ID", "URL", "API", "IP", "RPC", "JSON", "HTTP", "HTTPS"].includes(upper)) {
        return upper;
      }
      return upper.slice(0, 1) + word.slice(1);
    });

  return words.join(" ");
}

function isEmptyStringValue(value: unknown): boolean {
  return typeof value === "string" && value.trim() === "";
}

function isMissingFieldValue(field: PluginUIFormField, value: unknown): boolean {
  const type = normalizeFieldType(field);
  if (type === "boolean" || type === "switch") {
    return value === undefined || value === null;
  }
  if (type === "number" || type === "integer") {
    return value === undefined || value === null || value === "";
  }
  if (type === "account" || type === "account_ref") {
    return value === undefined || value === null || String(value).trim() === "";
  }
  return value === undefined || value === null || isEmptyStringValue(value);
}

function coerceDefaultNumber(value: unknown): number | undefined {
  if (value === undefined || value === null || value === "") {
    return undefined;
  }
  const num = Number(value);
  return Number.isFinite(num) ? num : undefined;
}

export function pluginFieldLabel(field: PluginUIFormField): string {
  return normalizeFieldLabel(field);
}

export function isPluginSecretField(field: PluginUIFormField): boolean {
  const type = normalizeFieldType(field);
  return type === "password";
}

export function formatPluginAccountOption(account: Account): string {
  const parts = [account.identifier];
  if (account.account_type_key) {
    parts.push(account.account_type_key);
  }
  parts.push(`#${account.id}`);
  return parts.join(" · ");
}

export function filterPluginFieldAccounts(
  accounts: Account[],
  field: PluginUIFormField,
  fallbackTypeKey = "",
): Account[] {
  const accountTypeKey = String(field.account_type_key ?? fallbackTypeKey).trim();
  if (!accountTypeKey) {
    return accounts;
  }
  return accounts.filter((account) => account.account_type_key === accountTypeKey);
}

export function createPluginUIFieldDefaultValue(field: PluginUIFormField): unknown {
  const type = normalizeFieldType(field);
  const fieldDefault = field.default;

  if (fieldDefault !== undefined) {
    if (type === "boolean" || type === "switch") {
      return Boolean(fieldDefault);
    }
    if (type === "number" || type === "integer") {
      return coerceDefaultNumber(fieldDefault);
    }
    if (type === "account" || type === "account_ref") {
      if (typeof fieldDefault === "number" || typeof fieldDefault === "string") {
        return String(fieldDefault);
      }
      if (fieldDefault && typeof fieldDefault === "object" && "id" in (fieldDefault as Record<string, unknown>)) {
        return String((fieldDefault as Record<string, unknown>).id ?? "");
      }
      return "";
    }
    if (type === "json") {
      if (typeof fieldDefault === "string") {
        return fieldDefault;
      }
      return JSON.stringify(fieldDefault, null, 2);
    }
    return String(fieldDefault);
  }

  if (type === "boolean" || type === "switch") {
    return false;
  }
  if (type === "number" || type === "integer") {
    return undefined;
  }
  return "";
}

export function createPluginUIButtonFormState(button: PluginUIButton): Record<string, unknown> {
  const values: Record<string, unknown> = {};
  for (const field of button.form) {
    values[field.name] = createPluginUIFieldDefaultValue(field);
  }
  return values;
}

function resolveAccountReference(value: unknown, accounts: Account[], field: PluginUIFormField): PluginAccountReference {
  const matched = accounts.find((account) => String(account.id) === String(value).trim());
  if (!matched) {
    throw new Error(`${normalizeFieldLabel(field)} 选择的账号不存在`);
  }
  return {
    id: matched.id,
    identifier: matched.identifier,
  };
}

function applyNumericConstraints(field: PluginUIFormField, value: number): number {
  const label = normalizeFieldLabel(field);
  if (typeof field.min === "number" && value < field.min) {
    throw new Error(`${label} 不能小于 ${field.min}`);
  }
  if (typeof field.max === "number" && value > field.max) {
    throw new Error(`${label} 不能大于 ${field.max}`);
  }
  return value;
}

function resolveFieldParamValue(field: PluginUIFormField, value: unknown): unknown {
  const type = normalizeFieldType(field);
  const label = normalizeFieldLabel(field);

  if (type === "number" || type === "integer") {
    const numeric = Number(value);
    if (!Number.isFinite(numeric)) {
      throw new Error(`${label} 必须是有效数字`);
    }
    if (type === "integer" && !Number.isInteger(numeric)) {
      throw new Error(`${label} 必须是整数`);
    }
    return applyNumericConstraints(field, numeric);
  }

  if (type === "boolean" || type === "switch") {
    return Boolean(value);
  }

  if (type === "json") {
    if (typeof value !== "string") {
      return value;
    }
    try {
      return JSON.parse(value);
    } catch {
      throw new Error(`${label} 必须是合法 JSON`);
    }
  }

  if (typeof value === "string") {
    return value.trim();
  }

  return value;
}

export function resolvePluginUIButtonInput(
  button: PluginUIButton,
  values: Record<string, unknown>,
  accounts: Account[],
  options: ResolvePluginUIButtonOptions = {},
): ResolvedPluginUIButtonInput {
  const params: Record<string, unknown> = { ...(button.params ?? {}) };
  let account: PluginAccountReference | undefined;

  for (const field of button.form) {
    const value = values[field.name];
    const missing = isMissingFieldValue(field, value);
    if (field.required && missing) {
      throw new Error(`${normalizeFieldLabel(field)} 为必填项`);
    }
    if (missing) {
      continue;
    }

    const type = normalizeFieldType(field);
    if (type === "account" || type === "account_ref") {
      const reference = resolveAccountReference(value, accounts, field);
      const bind = String(field.bind ?? "").trim().toLowerCase();
      if (options.bindAccountContext && bind !== "param" && bind !== "params") {
        account = reference;
      } else {
        params[field.name] = reference;
      }
      continue;
    }

    params[field.name] = resolveFieldParamValue(field, value);
  }

  return { params, account };
}
