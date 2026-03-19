import type { VNode } from "vue";
import { Fragment } from "vue";

export function cx(...values: Array<unknown>): string {
  return values
    .flatMap((value) => {
      if (!value) return [];
      if (Array.isArray(value)) return [cx(...value)];
      if (typeof value === "object") {
        return Object.entries(value as Record<string, unknown>)
          .filter(([, enabled]) => Boolean(enabled))
          .map(([key]) => key);
      }
      return [String(value)];
    })
    .filter(Boolean)
    .join(" ");
}

export function toKebabCase(value: string) {
  return value
    .replace(/([a-z0-9])([A-Z])/g, "$1-$2")
    .replace(/([A-Z])([A-Z][a-z])/g, "$1-$2")
    .toLowerCase();
}

export function getFromPath(source: unknown, path: string | undefined): unknown {
  if (!path || source == null) return source;
  return path.split(".").reduce<unknown>((acc, key) => {
    if (acc && typeof acc === "object") {
      return (acc as Record<string, unknown>)[key];
    }
    return undefined;
  }, source);
}

export function normalizeWidth(width: unknown): string | undefined {
  if (typeof width === "number") return `${width}px`;
  if (typeof width === "string") return width;
  return undefined;
}

export function flattenNodes(nodes: VNode[] = []): VNode[] {
  const result: VNode[] = [];
  for (const node of nodes) {
    if (!node) continue;
    if (node.type === Fragment && Array.isArray(node.children)) {
      result.push(...flattenNodes(node.children as VNode[]));
      continue;
    }
    result.push(node);
  }
  return result;
}

export function optionValue(option: HTMLOptionElement): unknown {
  const withRaw = option as HTMLOptionElement & { _value?: unknown };
  return withRaw._value !== undefined ? withRaw._value : option.value;
}

export const TONE_CLASS: Record<string, string> = {
  success: "text-emerald-700 bg-emerald-50 border-emerald-200",
  error:   "text-red-700 bg-red-50 border-red-200",
  warning: "text-amber-700 bg-amber-50 border-amber-200",
  info:    "text-sky-700 bg-sky-50 border-sky-200",
  default: "text-slate-700 bg-white border-slate-200",
};

export const TAG_TONE_CLASS: Record<string, string> = {
  gray:    "text-slate-600 bg-slate-100 border-slate-200",
  twblue:  "text-sky-700 bg-sky-100 border-sky-200",
  blue:    "text-sky-700 bg-sky-100 border-sky-200",
  green:   "text-emerald-700 bg-emerald-100 border-emerald-200",
  red:     "text-red-700 bg-red-100 border-red-200",
  orange:  "text-amber-700 bg-amber-100 border-amber-200",
  warning: "text-amber-700 bg-amber-100 border-amber-200",
  success: "text-emerald-700 bg-emerald-100 border-emerald-200",
  danger:  "text-red-700 bg-red-100 border-red-200",
};

export const BUTTON_SIZE_CLASS: Record<string, string> = {
  mini:   "h-7 px-2.5 text-xs",
  small:  "h-9 px-3.5 text-[13px]",
  medium: "h-10 px-5 text-sm",
  large:  "h-12 px-6 text-[15px]",
};

export const BUTTON_SIZE_MARKER_CLASS: Record<string, string> = {
  mini:   "ui-btn-size-mini",
  small:  "ui-btn-size-small",
  medium: "",
  large:  "",
};
