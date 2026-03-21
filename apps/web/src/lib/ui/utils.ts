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
  success: "text-emerald-700 bg-emerald-50/85 border-emerald-200/80",
  error:   "text-rose-700 bg-rose-50/85 border-rose-200/80",
  warning: "text-amber-700 bg-amber-50/90 border-amber-200/80",
  info:    "text-blue-700 bg-blue-50/85 border-blue-200/80",
  default: "text-slate-700 bg-white/70 border-white/80",
};

export const TAG_TONE_CLASS: Record<string, string> = {
  gray:    "text-slate-600 bg-slate-100/88 border-white/90",
  twblue:  "text-blue-700 bg-blue-50/92 border-blue-200/80",
  blue:    "text-blue-700 bg-blue-50/92 border-blue-200/80",
  green:   "text-emerald-700 bg-emerald-50/92 border-emerald-200/80",
  red:     "text-rose-700 bg-rose-50/92 border-rose-200/80",
  orange:  "text-amber-700 bg-amber-50/92 border-amber-200/80",
  warning: "text-amber-700 bg-amber-50/92 border-amber-200/80",
  success: "text-emerald-700 bg-emerald-50/92 border-emerald-200/80",
  danger:  "text-rose-700 bg-rose-50/92 border-rose-200/80",
};

export const BUTTON_SIZE_CLASS: Record<string, string> = {
  mini:   "min-h-[2.15em] px-3 text-[0.72rem]",
  small:  "min-h-[2.5em] px-4 text-[0.8rem]",
  medium: "min-h-[2.9em] px-5 text-[0.92rem]",
  large:  "min-h-[3.25em] px-6 text-[0.98rem]",
};

export const BUTTON_SIZE_MARKER_CLASS: Record<string, string> = {
  mini:   "ui-btn-size-mini",
  small:  "ui-btn-size-small",
  medium: "",
  large:  "",
};

export const INPUT_SIZE_CLASS: Record<string, string> = {
  mini:   "min-h-[2.15em] px-3 text-[0.78rem]",
  small:  "min-h-[2.5em] px-3 text-[0.84rem]",
  medium: "min-h-[2.75em] px-3 text-[0.92rem]",
  large:  "min-h-[3em] px-4 text-[0.98rem]",
};
