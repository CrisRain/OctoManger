export type ConfigValue =
  | string
  | number
  | boolean
  | null
  | ConfigValue[]
  | { [key: string]: ConfigValue };

export type ConfigValueType =
  | "string"
  | "number"
  | "boolean"
  | "object"
  | "array"
  | "null";

export const CONFIG_VALUE_TYPE_OPTIONS: Array<{
  label: string;
  value: ConfigValueType;
}> = [
  { label: "文本", value: "string" },
  { label: "数字", value: "number" },
  { label: "开关", value: "boolean" },
  { label: "对象", value: "object" },
  { label: "数组", value: "array" },
  { label: "空值", value: "null" },
];

export function isConfigObject(
  value: ConfigValue,
): value is Record<string, ConfigValue> {
  return value !== null && typeof value === "object" && !Array.isArray(value);
}

export function configValueTypeOf(value: ConfigValue): ConfigValueType {
  if (value === null) {
    return "null";
  }
  if (Array.isArray(value)) {
    return "array";
  }
  switch (typeof value) {
    case "string":
      return "string";
    case "number":
      return "number";
    case "boolean":
      return "boolean";
    default:
      return "object";
  }
}

export function createConfigValue(type: ConfigValueType): ConfigValue {
  switch (type) {
    case "string":
      return "";
    case "number":
      return 0;
    case "boolean":
      return false;
    case "array":
      return [];
    case "null":
      return null;
    case "object":
    default:
      return {};
  }
}

export function normalizeConfigValue(value: unknown): ConfigValue {
  if (
    value === null ||
    typeof value === "string" ||
    typeof value === "number" ||
    typeof value === "boolean"
  ) {
    return value;
  }

  if (Array.isArray(value)) {
    return value.map((item) => normalizeConfigValue(item));
  }

  if (value && typeof value === "object") {
    return Object.fromEntries(
      Object.entries(value as Record<string, unknown>).map(([key, item]) => [
        key,
        normalizeConfigValue(item),
      ]),
    );
  }

  return String(value ?? "");
}

export function cloneConfigValue<T extends ConfigValue>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T;
}

export function configValuesEqual(left: ConfigValue, right: ConfigValue): boolean {
  return JSON.stringify(left) === JSON.stringify(right);
}

export function createUniqueObjectKey(
  value: Record<string, ConfigValue>,
  base = "field",
): string {
  if (!(base in value)) {
    return base;
  }

  let index = 1;
  let next = `${base}_${index}`;
  while (next in value) {
    index += 1;
    next = `${base}_${index}`;
  }
  return next;
}
