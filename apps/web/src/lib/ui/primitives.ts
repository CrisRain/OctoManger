import {
  Fragment,
  Teleport,
  computed,
  defineComponent,
  h,
  onBeforeUnmount,
  onMounted,
  ref,
  watch,
  type App,
  type PropType,
  type VNode,
} from "vue";

function cx(...values: Array<unknown>): string[] {
  return values.flatMap((value) => {
    if (!value) return [];
    if (Array.isArray(value)) return cx(...value);
    if (typeof value === "object") {
      return Object.entries(value as Record<string, unknown>)
        .filter(([, enabled]) => Boolean(enabled))
        .map(([key]) => key);
    }
    return [String(value)];
  });
}

function toKebabCase(value: string) {
  return value
    .replace(/([a-z0-9])([A-Z])/g, "$1-$2")
    .replace(/([A-Z])([A-Z][a-z])/g, "$1-$2")
    .toLowerCase();
}

function getFromPath(source: unknown, path: string | undefined) {
  if (!path || source == null) return source;
  return path.split(".").reduce<unknown>((acc, key) => {
    if (acc && typeof acc === "object") {
      return (acc as Record<string, unknown>)[key];
    }
    return undefined;
  }, source);
}

function flattenNodes(nodes: VNode[] = []): VNode[] {
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

function optionValue(option: HTMLOptionElement) {
  const withRaw = option as HTMLOptionElement & { _value?: unknown };
  return withRaw._value !== undefined ? withRaw._value : option.value;
}

const TONE_CLASS: Record<string, string> = {
  success: "text-emerald-700 bg-emerald-50 border-emerald-200",
  error: "text-red-700 bg-red-50 border-red-200",
  warning: "text-amber-700 bg-amber-50 border-amber-200",
  info: "text-sky-700 bg-sky-50 border-sky-200",
  default: "text-slate-700 bg-white border-slate-200",
};

const BUTTON_SIZE_CLASS: Record<string, string> = {
  mini: "h-6 px-2 text-xs",
  small: "h-8 px-3 text-[13px]",
  medium: "h-9 px-4 text-sm",
  large: "h-11 px-5 text-[15px]",
};

const BUTTON_SIZE_MARKER_CLASS: Record<string, string> = {
  mini: "ui-btn-size-mini",
  small: "ui-btn-size-small",
  medium: "",
  large: "",
};

const TAG_TONE_CLASS: Record<string, string> = {
  gray: "text-slate-600 bg-slate-100 border-slate-200",
  twblue: "text-sky-700 bg-sky-100 border-sky-200",
  blue: "text-sky-700 bg-sky-100 border-sky-200",
  green: "text-emerald-700 bg-emerald-100 border-emerald-200",
  red: "text-red-700 bg-red-100 border-red-200",
  orange: "text-amber-700 bg-amber-100 border-amber-200",
  warning: "text-amber-700 bg-amber-100 border-amber-200",
  success: "text-emerald-700 bg-emerald-100 border-emerald-200",
  danger: "text-red-700 bg-red-100 border-red-200",
};

export const UiButton = defineComponent({
  name: "UiButton",
  inheritAttrs: false,
  props: {
    type: { type: String, default: "secondary" },
    size: { type: String, default: "medium" },
    status: { type: String, default: "normal" },
    disabled: { type: Boolean, default: false },
    loading: { type: Boolean, default: false },
    htmlType: { type: String, default: "button" },
  },
  emits: ["click"],
  setup(props, { slots, attrs, emit }) {
    return () => {
      const classAttr = (attrs.class as string | undefined) ?? "";
      const nativeType = (attrs.type as string | undefined) ?? props.htmlType;
      const disabled = props.disabled || props.loading;

      const tone =
        props.status === "danger"
          ? TONE_CLASS.error
          : props.status === "warning"
            ? TONE_CLASS.warning
            : props.status === "success"
              ? TONE_CLASS.success
              : TONE_CLASS.default;

      const typeClass =
        props.type === "primary"
          ? "border-transparent bg-teal-700 text-white shadow-sm hover:bg-teal-800"
          : props.type === "text"
            ? "border-transparent bg-transparent hover:bg-slate-100"
            : props.type === "outline"
              ? "border-slate-300 bg-white hover:border-teal-600 hover:text-teal-700"
              : "border-slate-300 bg-white hover:bg-slate-50";

      return h(
        "button",
        {
          ...attrs,
          type: nativeType,
          disabled,
          class: cx(
            "ui-btn inline-flex items-center justify-center gap-2 rounded-xl border font-semibold transition",
            BUTTON_SIZE_CLASS[props.size] ?? BUTTON_SIZE_CLASS.medium,
            BUTTON_SIZE_MARKER_CLASS[props.size] ?? "",
            tone,
            props.type === "primary" && "ui-btn-primary",
            (props.type === "secondary" || (!["primary", "text", "outline"].includes(props.type))) &&
              "ui-btn-secondary",
            typeClass,
            disabled && "cursor-not-allowed opacity-60",
            classAttr,
          ),
          onClick: (event: MouseEvent) => {
            if (disabled) {
              event.preventDefault();
              return;
            }
            emit("click", event);
          },
        },
        [
          props.loading
            ? h("span", {
                class: "h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent",
              })
            : slots.icon?.(),
          slots.default?.(),
        ],
      );
    };
  },
});

export const UiCard = defineComponent({
  name: "UiCard",
  props: {
    title: { type: String, default: "" },
    bordered: { type: Boolean, default: true },
  },
  setup(props, { slots, attrs }) {
    return () => {
      const hasHeader = Boolean(props.title || slots.title || slots.extra);
      return h(
        "section",
        {
          ...attrs,
          class: cx(
            "ui-card rounded-2xl border border-slate-200 bg-white/90 shadow-sm",
            !props.bordered && "border-transparent shadow-none",
            attrs.class as string,
          ),
        },
        [
          hasHeader
            ? h("header", { class: "ui-card-header flex items-center justify-between gap-3 border-b border-slate-200 px-5 py-4" }, [
                h("div", { class: "ui-card-header-title font-semibold text-slate-900" }, slots.title?.() ?? props.title),
                slots.extra?.(),
              ])
            : null,
          h("div", { class: "ui-card-body p-5" }, slots.default?.()),
        ],
      );
    };
  },
});

export const UiDivider = defineComponent({
  name: "UiDivider",
  props: {
    orientation: { type: String as PropType<"left" | "center" | "right">, default: "center" },
  },
  setup(props, { slots, attrs }) {
    return () => {
      const content = slots.default?.();
      if (!content || content.length === 0) {
        return h("hr", {
          ...attrs,
          class: cx("ui-divider my-4 border-0 border-t border-slate-200", attrs.class as string),
        });
      }

      const alignClass =
        props.orientation === "left"
          ? "justify-start"
          : props.orientation === "right"
            ? "justify-end"
            : "justify-center";

      return h(
        "div",
        {
          ...attrs,
          class: cx("ui-divider my-4 flex items-center gap-3", alignClass, attrs.class as string),
        },
        [
          h("span", { class: "h-px flex-1 bg-slate-200" }),
          h("span", { class: "text-xs font-semibold uppercase tracking-wider text-slate-500" }, content),
          h("span", { class: "h-px flex-1 bg-slate-200" }),
        ],
      );
    };
  },
});

export const UiEmpty = defineComponent({
  name: "UiEmpty",
  props: {
    description: { type: String, default: "暂无数据" },
  },
  setup(props, { slots, attrs }) {
    return () =>
      h(
        "div",
        {
          ...attrs,
          class: cx("ui-empty flex flex-col items-center gap-3 py-10 text-center", attrs.class as string),
        },
        [
          h(
            "div",
            { class: "flex h-12 w-12 items-center justify-center rounded-full bg-slate-100 text-slate-400" },
            h("span", { class: "text-xl" }, "∅"),
          ),
          h("p", { class: "text-sm text-slate-500" }, props.description),
          slots.default?.(),
        ],
      );
  },
});

export const UiSpin = defineComponent({
  name: "UiSpin",
  props: {
    loading: { type: Boolean, default: true },
    size: { type: [Number, String], default: "1.5em" },
    tip: { type: String, default: "" },
  },
  setup(props, { slots, attrs }) {
    const normalizeSize = (value: number | string) =>
      typeof value === "number" ? `${value / 16}em` : String(value);

    const renderSpinner = () =>
      h("span", {
        class: "ui-spin-icon inline-block animate-spin rounded-full border-2 border-slate-300 border-t-teal-600",
        style: {
          inlineSize: normalizeSize(props.size),
          blockSize: normalizeSize(props.size),
        },
      });

    return () => {
      if (slots.default) {
        return h("div", { class: cx("relative", attrs.class as string) }, [
          slots.default?.(),
          props.loading
            ? h("div", { class: "absolute inset-0 flex flex-col items-center justify-center gap-2 bg-white/70 backdrop-blur-[1px]" }, [
                renderSpinner(),
                props.tip ? h("span", { class: "text-xs text-slate-500" }, props.tip) : null,
              ])
            : null,
        ]);
      }

      if (!props.loading) return null;

      return h("div", { class: cx("ui-spin inline-flex items-center gap-2", attrs.class as string) }, [
        renderSpinner(),
        props.tip ? h("span", { class: "text-xs text-slate-500" }, props.tip) : null,
      ]);
    };
  },
});

export const UiForm = defineComponent({
  name: "UiForm",
  setup(_, { slots, attrs }) {
    return () =>
      h(
        "div",
        {
          ...attrs,
          class: cx("ui-form space-y-5", attrs.class as string),
        },
        slots.default?.(),
      );
  },
});

export const UiFormItem = defineComponent({
  name: "UiFormItem",
  props: {
    label: { type: String, default: "" },
    required: { type: Boolean, default: false },
    validateStatus: { type: String, default: "" },
    help: { type: String, default: "" },
  },
  setup(props, { slots, attrs }) {
    const hasLabel = computed(() => Boolean(props.label || slots.label));

    return () =>
      h(
        "div",
        {
          ...attrs,
          class: cx("ui-form-item space-y-2", attrs.class as string),
        },
        [
          hasLabel.value
            ? h(
                "label",
                { class: "ui-form-item-label ui-form-item-label-col inline-flex w-full items-center gap-1 text-sm font-semibold text-slate-700" },
                slots.label?.() ?? [
                  props.label,
                  props.required ? h("span", { class: "text-red-500" }, "*") : null,
                ],
              )
            : null,
          h("div", { class: "ui-form-item-wrapper-col ui-form-item-body ui-form-item-content-flex" }, slots.default?.()),
          props.help
            ? h(
                "div",
                {
                  class: cx(
                    "ui-form-item-help text-xs",
                    props.validateStatus === "error" ? "text-red-600" : "text-slate-500",
                  ),
                },
                props.help,
              )
            : null,
        ],
      );
  },
});

export const UiInput = defineComponent({
  name: "UiInput",
  inheritAttrs: false,
  props: {
    modelValue: { type: [String, Number] as PropType<string | number | null | undefined>, default: "" },
    type: { type: String, default: "text" },
    placeholder: { type: String, default: "" },
    disabled: { type: Boolean, default: false },
    allowClear: { type: Boolean, default: false },
  },
  emits: ["update:modelValue", "change", "focus", "blur", "clear"],
  setup(props, { slots, attrs, emit }) {
    return () => {
      const { class: classAttr, style, ...inputAttrs } = attrs;
      const value = props.modelValue ?? "";

      return h(
        "div",
        {
          class: cx(
            "ui-input-wrapper flex items-center gap-2 rounded-xl border border-slate-300 bg-white px-3 focus-within:border-teal-700 focus-within:ring-4 focus-within:ring-teal-700/10",
            props.disabled && "bg-slate-100",
            classAttr as string,
          ),
          style,
        },
        [
          slots.prefix?.(),
          h("input", {
            ...inputAttrs,
            class: cx(
              "ui-input h-9 w-full border-0 bg-transparent px-0 text-sm text-slate-900 outline-none placeholder:text-slate-400",
              (inputAttrs as Record<string, unknown>).class as string,
            ),
            value,
            type: props.type,
            placeholder: props.placeholder,
            disabled: props.disabled,
            onInput: (event: Event) => {
              const nextValue = (event.target as HTMLInputElement).value;
              emit("update:modelValue", nextValue);
            },
            onChange: (event: Event) => {
              emit("change", (event.target as HTMLInputElement).value);
            },
            onFocus: (event: FocusEvent) => emit("focus", event),
            onBlur: (event: FocusEvent) => emit("blur", event),
          }),
          props.allowClear && String(value).length > 0 && !props.disabled
            ? h(
                "button",
                {
                  type: "button",
                  class: "text-slate-400 transition hover:text-slate-600",
                  onClick: (event: MouseEvent) => {
                    event.preventDefault();
                    event.stopPropagation();
                    emit("update:modelValue", "");
                    emit("clear");
                  },
                },
                "×",
              )
            : null,
          slots.suffix?.(),
        ],
      );
    };
  },
});

export const UiTextarea = defineComponent({
  name: "UiTextarea",
  inheritAttrs: false,
  props: {
    modelValue: { type: [String, Number] as PropType<string | number | null | undefined>, default: "" },
    placeholder: { type: String, default: "" },
    rows: { type: Number, default: 3 },
    autoSize: { type: [Boolean, Object] as PropType<boolean | { minRows?: number; maxRows?: number }>, default: false },
    readonly: { type: Boolean, default: false },
    disabled: { type: Boolean, default: false },
  },
  emits: ["update:modelValue", "change", "focus", "blur"],
  setup(props, { attrs, emit }) {
    return () => {
      const { class: classAttr, style, ...textareaAttrs } = attrs;
      const autoSize = typeof props.autoSize === "object" ? props.autoSize : undefined;
      const minRows = autoSize?.minRows ?? props.rows;

      return h(
        "div",
        {
          class: cx(
            "ui-textarea-wrapper rounded-xl border border-slate-300 bg-white px-3 py-2 focus-within:border-teal-700 focus-within:ring-4 focus-within:ring-teal-700/10",
            classAttr as string,
          ),
          style,
        },
        [
          h("textarea", {
            ...textareaAttrs,
            class: "ui-textarea w-full resize-y border-0 bg-transparent text-sm text-slate-900 outline-none placeholder:text-slate-400",
            value: props.modelValue ?? "",
            placeholder: props.placeholder,
            rows: minRows,
            readonly: props.readonly,
            disabled: props.disabled,
            onInput: (event: Event) => {
              emit("update:modelValue", (event.target as HTMLTextAreaElement).value);
            },
            onChange: (event: Event) => {
              emit("change", (event.target as HTMLTextAreaElement).value);
            },
            onFocus: (event: FocusEvent) => emit("focus", event),
            onBlur: (event: FocusEvent) => emit("blur", event),
          }),
        ],
      );
    };
  },
});

export const UiInputNumber = defineComponent({
  name: "UiInputNumber",
  inheritAttrs: false,
  props: {
    modelValue: { type: [String, Number] as PropType<string | number | null | undefined>, default: undefined },
    min: { type: Number, default: undefined },
    max: { type: Number, default: undefined },
    step: { type: Number, default: 1 },
    placeholder: { type: String, default: "" },
    disabled: { type: Boolean, default: false },
  },
  emits: ["update:modelValue", "change", "focus", "blur"],
  setup(props, { attrs, emit }) {
    return () => {
      const { class: classAttr, style, ...inputAttrs } = attrs;
      return h(
        "div",
        {
          class: cx(
            "ui-input-number flex items-center rounded-xl border border-slate-300 bg-white px-3 focus-within:border-teal-700 focus-within:ring-4 focus-within:ring-teal-700/10",
            classAttr as string,
          ),
          style,
        },
        [
          h("input", {
            ...inputAttrs,
            class: "h-9 w-full border-0 bg-transparent text-sm text-slate-900 outline-none placeholder:text-slate-400",
            type: "number",
            value: props.modelValue ?? "",
            min: props.min,
            max: props.max,
            step: props.step,
            placeholder: props.placeholder,
            disabled: props.disabled,
            onInput: (event: Event) => {
              const raw = (event.target as HTMLInputElement).value;
              if (raw === "") {
                emit("update:modelValue", undefined);
                return;
              }
              const next = Number(raw);
              if (Number.isNaN(next)) return;
              emit("update:modelValue", next);
            },
            onChange: (event: Event) => {
              const raw = (event.target as HTMLInputElement).value;
              const next = raw === "" ? undefined : Number(raw);
              emit("change", Number.isNaN(next) ? undefined : next);
            },
            onFocus: (event: FocusEvent) => emit("focus", event),
            onBlur: (event: FocusEvent) => emit("blur", event),
          }),
        ],
      );
    };
  },
});

export const UiOption = defineComponent({
  name: "UiOption",
  props: {
    value: { type: null as unknown as PropType<any>, required: false },
    disabled: { type: Boolean, default: false },
  },
  setup(props, { slots, attrs }) {
    return () =>
      h(
        "option",
        {
          ...attrs,
          value: props.value as never,
          disabled: props.disabled,
        },
        slots.default?.(),
      );
  },
});

export const UiSelect = defineComponent({
  name: "UiSelect",
  inheritAttrs: false,
  props: {
    modelValue: { type: null as unknown as PropType<any>, default: undefined },
    placeholder: { type: String, default: "" },
    disabled: { type: Boolean, default: false },
    allowClear: { type: Boolean, default: false },
    multiple: { type: Boolean, default: false },
  },
  emits: ["update:modelValue", "change", "focus", "blur", "clear"],
  setup(props, { slots, attrs, emit }) {
    return () => {
      const { class: classAttr, style, ...selectAttrs } = attrs;

      return h(
        "div",
        {
          class: cx(
            "ui-select-view relative flex items-center rounded-xl border border-slate-300 bg-white px-3 focus-within:border-teal-700 focus-within:ring-4 focus-within:ring-teal-700/10",
            !props.multiple && "ui-select-view-single",
            props.disabled && "bg-slate-100",
            classAttr as string,
          ),
          style,
        },
        [
          h(
            "select",
            {
              ...selectAttrs,
              class: "h-9 w-full appearance-none border-0 bg-transparent pr-7 text-sm text-slate-900 outline-none",
              value: (props.modelValue ?? "") as never,
              disabled: props.disabled,
              multiple: props.multiple,
              onChange: (event: Event) => {
                const target = event.target as HTMLSelectElement;
                if (props.multiple) {
                  const values = Array.from(target.selectedOptions).map((option) => optionValue(option));
                  emit("update:modelValue", values);
                  emit("change", values);
                  return;
                }
                const selectedOption = target.options[target.selectedIndex];
                const nextValue = selectedOption ? optionValue(selectedOption) : target.value;
                emit("update:modelValue", nextValue);
                emit("change", nextValue);
              },
              onFocus: (event: FocusEvent) => emit("focus", event),
              onBlur: (event: FocusEvent) => emit("blur", event),
            },
            [
              props.placeholder && !props.multiple
                ? h(
                    "option",
                    {
                      value: "",
                      disabled: true,
                      selected: props.modelValue === "" || props.modelValue == null,
                    },
                    props.placeholder,
                  )
                : null,
              slots.default?.(),
            ],
          ),
          props.allowClear && !props.multiple && props.modelValue
            ? h(
                "button",
                {
                  type: "button",
                  class: "mr-1 text-slate-400 transition hover:text-slate-600",
                  onClick: (event: MouseEvent) => {
                    event.preventDefault();
                    event.stopPropagation();
                    emit("update:modelValue", "");
                    emit("change", "");
                    emit("clear");
                  },
                },
                "×",
              )
            : h("span", { class: "pointer-events-none absolute right-3 text-slate-400" }, "▾"),
        ],
      );
    };
  },
});

export const UiSwitch = defineComponent({
  name: "UiSwitch",
  props: {
    modelValue: { type: Boolean, default: false },
    disabled: { type: Boolean, default: false },
    checkedText: { type: String, default: "" },
    uncheckedText: { type: String, default: "" },
  },
  emits: ["update:modelValue", "change"],
  setup(props, { attrs, emit }) {
    return () => {
      const checked = props.modelValue;
      return h(
        "button",
        {
          ...attrs,
          type: "button",
          class: cx(
            "ui-switch inline-flex h-7 min-w-12 items-center rounded-full border px-1 transition",
            checked ? "ui-switch-checked border-teal-600 bg-teal-600" : "border-slate-300 bg-slate-200",
            props.disabled && "cursor-not-allowed opacity-60",
            attrs.class as string,
          ),
          onClick: (event: MouseEvent) => {
            if (props.disabled) {
              event.preventDefault();
              return;
            }
            const next = !checked;
            emit("update:modelValue", next);
            emit("change", next);
          },
        },
        [
          h("span", {
            class: cx(
              "h-5 w-5 rounded-full bg-white shadow transition-transform",
              checked ? "translate-x-5" : "translate-x-0",
            ),
          }),
          props.checkedText || props.uncheckedText
            ? h(
                "span",
                {
                  class: cx("ml-1 mr-1 text-[11px] font-semibold", checked ? "text-white" : "text-slate-600"),
                },
                checked ? props.checkedText : props.uncheckedText,
              )
            : null,
        ],
      );
    };
  },
});

export const UiTag = defineComponent({
  name: "UiTag",
  props: {
    color: { type: String, default: "gray" },
    size: { type: String, default: "medium" },
    closable: { type: Boolean, default: false },
  },
  emits: ["close"],
  setup(props, { attrs, slots, emit }) {
    return () =>
      h(
        "span",
        {
          ...attrs,
          class: cx(
            "ui-tag inline-flex items-center gap-1 rounded-full border px-2 py-0.5 font-semibold",
            props.size === "small" ? "text-[11px]" : "text-xs",
            TAG_TONE_CLASS[props.color] ?? TAG_TONE_CLASS.gray,
            attrs.class as string,
          ),
        },
        [
          slots.icon?.(),
          slots.default?.(),
          props.closable
            ? h(
                "button",
                {
                  type: "button",
                  class: "ml-1 text-current/70 transition hover:text-current",
                  onClick: (event: MouseEvent) => {
                    event.preventDefault();
                    event.stopPropagation();
                    emit("close", event);
                  },
                },
                "×",
              )
            : null,
        ],
      );
  },
});

export const UiModal = defineComponent({
  name: "UiModal",
  props: {
    visible: { type: Boolean, default: false },
    title: { type: String, default: "" },
    footer: { type: [Boolean, String] as PropType<boolean | string>, default: true },
    okText: { type: String, default: "确定" },
    cancelText: { type: String, default: "取消" },
    okLoading: { type: Boolean, default: false },
    closable: { type: Boolean, default: true },
    maskClosable: { type: Boolean, default: true },
  },
  emits: ["update:visible", "ok", "cancel", "close"],
  setup(props, { attrs, slots, emit }) {
    const close = () => {
      emit("update:visible", false);
      emit("cancel");
      emit("close");
    };

    return () => {
      if (!props.visible) return null;

      return h(Teleport, { to: "body" }, [
        h(
          "div",
          {
            class: cx("ui-modal fixed inset-0 z-[500] flex items-center justify-center bg-slate-900/40 p-4", attrs.class as string),
            onClick: (event: MouseEvent) => {
              if (props.maskClosable && event.target === event.currentTarget) {
                close();
              }
            },
          },
          [
            h("div", { class: "ui-modal-simple w-full max-h-[90vh] overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-2xl", style: { maxInlineSize: "var(--modal-inline-size)" } }, [
              props.title || slots.title
                ? h("header", { class: "ui-modal-header flex items-center justify-between border-b border-slate-200 px-6 py-4" }, [
                    h("h3", { class: "text-base font-semibold text-slate-900" }, slots.title?.() ?? props.title),
                    props.closable
                      ? h(
                          "button",
                          {
                            type: "button",
                            class: "rounded-lg p-1 text-slate-400 transition hover:bg-slate-100 hover:text-slate-700",
                            onClick: close,
                          },
                          h("svg", {
                            class: "h-[1em] w-[1em]",
                            viewBox: "0 0 24 24",
                            fill: "none",
                            stroke: "currentColor",
                            "stroke-width": "2",
                            "stroke-linecap": "round",
                            "stroke-linejoin": "round",
                          }, [
                            h("path", { d: "M18 6 6 18" }),
                            h("path", { d: "m6 6 12 12" }),
                          ]),
                        )
                      : null,
                  ])
                : null,
              h("div", { class: "ui-modal-body overflow-auto px-6 py-5", style: { maxBlockSize: "var(--modal-body-block-size)" } }, slots.default?.()),
              props.footer === false
                ? null
                : h("footer", { class: "ui-modal-footer flex items-center justify-end gap-3 border-t border-slate-200 px-6 py-4" }, [
                    slots.footer
                      ? slots.footer()
                      : [
                          h(
                            UiButton,
                            {
                              onClick: close,
                            },
                            { default: () => props.cancelText },
                          ),
                          h(
                            UiButton,
                            {
                              type: "primary",
                              loading: props.okLoading,
                              onClick: () => emit("ok"),
                            },
                            { default: () => props.okText },
                          ),
                        ],
                  ]),
            ]),
          ],
        ),
      ]);
    };
  },
});

export const UiDrawer = defineComponent({
  name: "UiDrawer",
  props: {
    visible: { type: Boolean, default: false },
    placement: { type: String as PropType<"left" | "right" | "top" | "bottom">, default: "right" },
    closable: { type: Boolean, default: true },
    header: { type: Boolean, default: true },
    footer: { type: Boolean, default: false },
    title: { type: String, default: "" },
    maskClosable: { type: Boolean, default: true },
  },
  emits: ["update:visible", "cancel", "close"],
  setup(props, { attrs, slots, emit }) {
    const close = () => {
      emit("update:visible", false);
      emit("cancel");
      emit("close");
    };

    const panelClass = computed(() => {
      if (props.placement === "left") return "h-full max-w-full rounded-r-2xl";
      if (props.placement === "top") return "w-full rounded-b-2xl";
      if (props.placement === "bottom") return "mt-auto w-full rounded-t-2xl";
      return "ml-auto h-full max-w-full rounded-l-2xl";
    });

    const panelStyle = computed(() => {
      if (props.placement === "top" || props.placement === "bottom") {
        return { blockSize: "var(--drawer-block-size)" };
      }
      return { inlineSize: "var(--drawer-inline-size)" };
    });

    return () => {
      if (!props.visible) return null;

      return h(Teleport, { to: "body" }, [
        h(
          "div",
          {
            class: cx("ui-drawer-container fixed inset-0 z-[450] flex bg-slate-900/40", attrs.class as string),
            onClick: (event: MouseEvent) => {
              if (props.maskClosable && event.target === event.currentTarget) {
                close();
              }
            },
          },
          [
            h(
              "section",
              {
                class: cx("ui-drawer flex w-full max-w-full flex-col bg-white shadow-2xl", panelClass.value),
                style: panelStyle.value,
                onClick: (event: Event) => event.stopPropagation(),
              },
              [
                props.header
                  ? h("header", { class: "ui-drawer-header flex items-center justify-between border-b border-slate-200 px-5 py-4" }, [
                      h("h3", { class: "text-base font-semibold text-slate-900" }, slots.title?.() ?? props.title),
                      props.closable
                        ? h(
                            "button",
                            {
                              type: "button",
                              class: "rounded-lg p-1 text-slate-400 transition hover:bg-slate-100 hover:text-slate-700",
                              onClick: close,
                            },
                            h("svg", {
                              class: "h-[1em] w-[1em]",
                              viewBox: "0 0 24 24",
                              fill: "none",
                              stroke: "currentColor",
                              "stroke-width": "2",
                              "stroke-linecap": "round",
                              "stroke-linejoin": "round",
                            }, [
                              h("path", { d: "M18 6 6 18" }),
                              h("path", { d: "m6 6 12 12" }),
                            ]),
                          )
                        : null,
                    ])
                  : null,
                h("div", { class: "ui-drawer-body min-h-0 flex-1 overflow-auto p-5" }, slots.default?.()),
                props.footer ? h("footer", { class: "border-t border-slate-200 px-5 py-4" }, slots.footer?.()) : null,
              ],
            ),
          ],
        ),
      ]);
    };
  },
});

export const UiDropdown = defineComponent({
  name: "UiDropdown",
  props: {
    visible: { type: Boolean, default: undefined },
  },
  emits: ["update:visible", "popupVisibleChange", "click"],
  setup(props, { slots, attrs, emit }) {
    const rootRef = ref<HTMLElement | null>(null);
    const internalVisible = ref(false);

    const mergedVisible = computed(() =>
      typeof props.visible === "boolean" ? props.visible : internalVisible.value,
    );

    const setVisible = (next: boolean) => {
      if (typeof props.visible !== "boolean") {
        internalVisible.value = next;
      }
      emit("update:visible", next);
      emit("popupVisibleChange", next);
    };

    const handleDocumentClick = (event: MouseEvent) => {
      if (!mergedVisible.value) return;
      const target = event.target as Node | null;
      if (target && rootRef.value?.contains(target)) return;
      setVisible(false);
    };

    onMounted(() => {
      document.addEventListener("click", handleDocumentClick, true);
    });

    onBeforeUnmount(() => {
      document.removeEventListener("click", handleDocumentClick, true);
    });

    return () =>
      h(
        "div",
        {
          ...attrs,
          ref: rootRef,
          class: cx("relative inline-flex", mergedVisible.value && "ui-dropdown-open", attrs.class as string),
          onClick: (event: MouseEvent) => emit("click", event),
        },
        [
          h(
            "div",
            {
              class: "inline-flex",
              onClick: (event: MouseEvent) => {
                event.stopPropagation();
                setVisible(!mergedVisible.value);
              },
            },
            slots.default?.(),
          ),
          mergedVisible.value
            ? h(
                "div",
                {
                  class: "ui-dropdown-popup absolute right-0 top-full z-[600] mt-2 min-w-[180px] rounded-2xl border border-slate-200 bg-white p-2 shadow-xl",
                  onClick: (event: Event) => event.stopPropagation(),
                },
                slots.content?.(),
              )
            : null,
        ],
      );
  },
});

export const UiPopconfirm = defineComponent({
  name: "UiPopconfirm",
  props: {
    content: { type: String, default: "确认执行该操作？" },
    title: { type: String, default: "" },
  },
  emits: ["ok", "cancel"],
  setup(props, { slots, emit, attrs }) {
    return () =>
      h(
        "span",
        {
          ...attrs,
          class: cx("inline-flex", attrs.class as string),
          onClick: (event: MouseEvent) => {
            event.preventDefault();
            event.stopPropagation();
            const text = props.title ? `${props.title}\n\n${props.content}` : props.content;
            if (window.confirm(text)) {
              emit("ok");
            } else {
              emit("cancel");
            }
          },
        },
        slots.default?.(),
      );
  },
});

export const UiSkeletonLine = defineComponent({
  name: "UiSkeletonLine",
  props: {
    rows: { type: Number, default: 1 },
    lineHeight: { type: [Number, String], default: "0.875em" },
    lineSpacing: { type: [Number, String], default: "0.5em" },
    widths: { type: Array as PropType<Array<number | string>>, default: () => ["100%"] },
  },
  setup(props, { attrs }) {
    const normalizeLength = (value: number | string) =>
      typeof value === "number" ? `${value / 16}em` : String(value);

    return () =>
      h(
        "div",
        {
          ...attrs,
          class: cx("space-y-2", attrs.class as string),
        },
        Array.from({ length: props.rows }).map((_, index) =>
          h("div", {
            key: index,
            class: "ui-skeleton-line-row h-3 animate-pulse rounded bg-slate-200",
            style: {
              blockSize: normalizeLength(props.lineHeight),
              marginTop: index === 0 ? "0" : normalizeLength(props.lineSpacing),
              width: String(props.widths[index] ?? props.widths[props.widths.length - 1] ?? "100%"),
            },
          }),
        ),
      );
  },
});

export const UiSkeleton = defineComponent({
  name: "UiSkeleton",
  props: {
    animation: { type: Boolean, default: true },
  },
  setup(props, { slots, attrs }) {
    return () =>
      h(
        "div",
        {
          ...attrs,
          class: cx("ui-skeleton", props.animation && "animate-pulse", attrs.class as string),
        },
        slots.default?.(),
      );
  },
});

interface ParsedColumn {
  key: string;
  title?: string;
  dataIndex?: string;
  align?: "left" | "center" | "right";
  slotName?: string;
  cell?: (args: { record: unknown; rowIndex: number; column: ParsedColumn }) => VNode[];
}

export const UiTableColumn = defineComponent({
  name: "UiTableColumn",
  props: {
    title: { type: String, default: "" },
    dataIndex: { type: String, default: "" },
    align: { type: String as PropType<"left" | "center" | "right">, default: "left" },
  },
  setup() {
    return () => null;
  },
});

export const UiTable = defineComponent({
  name: "UiTable",
  props: {
    data: { type: Array as PropType<unknown[]>, default: () => [] },
    columns: { type: Array as PropType<Array<Record<string, unknown>>>, default: undefined },
    loading: { type: Boolean, default: false },
    pagination: { type: [Boolean, Object] as PropType<boolean | Record<string, unknown>>, default: false },
    rowKey: { type: [String, Function] as PropType<string | ((record: unknown) => string | number)>, default: "id" },
  },
  setup(props, { slots, attrs }) {
    const currentPage = ref(1);

    const pageSize = computed(() => {
      if (props.pagination && typeof props.pagination === "object") {
        const size = Number(
          (props.pagination as Record<string, unknown>).pageSize ??
            (props.pagination as Record<string, unknown>).defaultPageSize ??
            20,
        );
        return Number.isFinite(size) && size > 0 ? size : 20;
      }
      return 20;
    });

    const paginationEnabled = computed(() => Boolean(props.pagination));

    const allRows = computed(() => (Array.isArray(props.data) ? props.data : []));

    const pagedRows = computed(() => {
      if (!paginationEnabled.value) return allRows.value;
      const start = (currentPage.value - 1) * pageSize.value;
      return allRows.value.slice(start, start + pageSize.value);
    });

    const totalPages = computed(() => Math.max(1, Math.ceil(allRows.value.length / pageSize.value)));

    watch([allRows, totalPages], () => {
      if (currentPage.value > totalPages.value) {
        currentPage.value = totalPages.value;
      }
    });

    const parsedColumns = computed<ParsedColumn[]>(() => {
      if (Array.isArray(props.columns) && props.columns.length > 0) {
        return props.columns.map((column, index) => ({
          key: String(column.key ?? column.dataIndex ?? index),
          title: String(column.title ?? ""),
          dataIndex: typeof column.dataIndex === "string" ? column.dataIndex : undefined,
          align: (column.align as ParsedColumn["align"]) ?? "left",
          slotName: typeof column.slotName === "string" ? column.slotName : undefined,
        }));
      }

      const source = flattenNodes(slots.columns?.() ?? slots.default?.() ?? []);

      return source
        .filter((node) => {
          const type = node.type as { name?: string };
          return type === UiTableColumn || type?.name === "UiTableColumn";
        })
        .map((node, index) => {
          const nodeProps = (node.props ?? {}) as Record<string, unknown>;
          const children = node.children as Record<string, (...args: any[]) => VNode[]> | null;
          const titleSlot = children?.title;

          return {
            key: String(node.key ?? nodeProps.dataIndex ?? index),
            title:
              typeof nodeProps.title === "string"
                ? nodeProps.title
                : titleSlot
                  ? (titleSlot().map((item) => (typeof item.children === "string" ? item.children : "")).join("") || "")
                  : "",
            dataIndex: typeof nodeProps.dataIndex === "string" ? nodeProps.dataIndex : undefined,
            align: (nodeProps.align as ParsedColumn["align"]) ?? "left",
            cell: children?.cell
              ? (args) => children.cell?.(args) ?? []
              : undefined,
          } satisfies ParsedColumn;
        });
    });

    const renderCell = (column: ParsedColumn, record: unknown, rowIndex: number) => {
      const slotName = column.slotName;
      if (slotName && slots[slotName]) {
        return slots[slotName]?.({ record, rowIndex, column });
      }

      if (column.cell) {
        return column.cell({ record, rowIndex, column });
      }

      const value = getFromPath(record, column.dataIndex);
      if (value == null || value === "") return "-";
      return String(value);
    };

    const keyForRow = (record: unknown, index: number) => {
      if (typeof props.rowKey === "function") {
        return String(props.rowKey(record));
      }

      if (record && typeof record === "object") {
        const maybe = (record as Record<string, unknown>)[props.rowKey];
        if (maybe != null) return String(maybe);
      }
      return String(index);
    };

    return () => {
      if (props.loading) {
        return h("div", { class: cx("ui-table-container flex min-h-[12em] items-center justify-center", attrs.class as string) }, [
          h(UiSpin, { loading: true, tip: "加载中..." }),
        ]);
      }

      const columns = parsedColumns.value;

      return h("div", { ...attrs, class: cx("ui-table-container ui-table-content overflow-x-auto", attrs.class as string) }, [
        h("table", { class: "ui-table ui-table-element min-w-full border-collapse text-sm" }, [
          columns.length
            ? h(
                "thead",
                h(
                  "tr",
                  columns.map((column) =>
                    h(
                      "th",
                      {
                        key: column.key,
                        class: "ui-table-th ui-table-column border-b border-slate-200 px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-slate-500",
                        style: {
                          textAlign: column.align ?? "left",
                        },
                      },
                      column.title,
                    ),
                  ),
                ),
              )
            : null,
          h(
            "tbody",
            pagedRows.value.length > 0
              ? pagedRows.value.map((record, rowIndex) =>
                  h(
                    "tr",
                    {
                      key: keyForRow(record, rowIndex),
                      class: "ui-table-tr hover:bg-slate-50",
                    },
                    columns.map((column) =>
                      h(
                        "td",
                        {
                          key: `${column.key}-${rowIndex}`,
                          class: "ui-table-td ui-table-column border-b border-slate-200 px-4 py-3 align-top text-slate-700",
                          style: {
                            textAlign: column.align ?? "left",
                          },
                        },
                        renderCell(column, record, rowIndex),
                      ),
                    ),
                  ),
                )
              : h(
                  "tr",
                  h(
                    "td",
                    {
                      class: "ui-table-td px-4 py-10",
                      colspan: Math.max(columns.length, 1),
                    },
                    slots.empty?.() ?? h(UiEmpty, { description: "暂无数据" }),
                  ),
                ),
          ),
        ]),
        paginationEnabled.value && totalPages.value > 1
          ? h("div", { class: "ui-pagination flex items-center justify-end gap-2 px-4 py-3" }, [
              h(
                UiButton,
                {
                  size: "small",
                  disabled: currentPage.value <= 1,
                  onClick: () => {
                    currentPage.value = Math.max(1, currentPage.value - 1);
                  },
                },
                { default: () => "上一页" },
              ),
              h("span", { class: "text-xs text-slate-500" }, `${currentPage.value} / ${totalPages.value}`),
              h(
                UiButton,
                {
                  size: "small",
                  disabled: currentPage.value >= totalPages.value,
                  onClick: () => {
                    currentPage.value = Math.min(totalPages.value, currentPage.value + 1);
                  },
                },
                { default: () => "下一页" },
              ),
            ])
          : null,
      ]);
    };
  },
});

export const UiTabPane = defineComponent({
  name: "UiTabPane",
  props: {
    title: { type: String, default: "" },
  },
  setup() {
    return () => null;
  },
});

interface ParsedTab {
  key: string;
  title: VNode[] | string;
  content: () => VNode[];
}

export const UiTabs = defineComponent({
  name: "UiTabs",
  props: {
    activeKey: { type: [String, Number] as PropType<string | number | undefined>, default: undefined },
    destroyOnHide: { type: Boolean, default: true },
  },
  emits: ["update:activeKey", "change"],
  setup(props, { slots, attrs, emit }) {
    const internalActive = ref<string>("");

    const tabs = computed<ParsedTab[]>(() => {
      const source = flattenNodes(slots.default?.() ?? []);
      const list = source
        .filter((node) => {
          const type = node.type as { name?: string };
          return type === UiTabPane || type?.name === "UiTabPane";
        })
        .map((node, index) => {
          const tabKey = String(node.key ?? index);
          const nodeProps = (node.props ?? {}) as Record<string, unknown>;
          const children = node.children as Record<string, (...args: any[]) => VNode[]> | null;
          const titleSlot = children?.title;
          const title = nodeProps.title ? String(nodeProps.title) : titleSlot ? titleSlot() : tabKey;
          return {
            key: tabKey,
            title,
            content: () => children?.default?.() ?? [],
          } satisfies ParsedTab;
        });
      return list;
    });

    watch(
      tabs,
      (nextTabs) => {
        if (nextTabs.length === 0) {
          internalActive.value = "";
          return;
        }
        const fallback = nextTabs[0].key;
        const current = props.activeKey != null ? String(props.activeKey) : internalActive.value;
        const exists = nextTabs.some((tab) => tab.key === current);
        if (!exists) {
          internalActive.value = fallback;
        }
      },
      { immediate: true },
    );

    const currentKey = computed(() => {
      if (props.activeKey != null) return String(props.activeKey);
      return internalActive.value;
    });

    const setActive = (key: string) => {
      if (props.activeKey == null) {
        internalActive.value = key;
      }
      emit("update:activeKey", key);
      emit("change", key);
    };

    return () => {
      const list = tabs.value;
      const active = currentKey.value || list[0]?.key;

      return h("div", { ...attrs, class: cx("ui-tabs", attrs.class as string) }, [
        h("div", { class: "ui-tabs-nav flex items-center border-b border-slate-200" }, [
          h(
            "div",
            { class: "ui-tabs-nav-tab-list flex items-center gap-1" },
            list.map((tab) =>
              h(
                "button",
                {
                  type: "button",
                  class: cx(
                    "ui-tabs-tab rounded-t-xl px-4 py-2 text-sm font-semibold transition",
                    tab.key === active
                      ? "ui-tabs-tab-active bg-white text-teal-700"
                      : "text-slate-500 hover:text-slate-700",
                  ),
                  onClick: () => setActive(tab.key),
                },
                h("span", { class: "ui-tabs-tab-title" }, tab.title),
              ),
            ),
          ),
        ]),
        h(
          "div",
          { class: "ui-tabs-content" },
          h(
            "div",
            { class: "ui-tabs-content-list" },
            props.destroyOnHide
              ? list
                  .filter((tab) => tab.key === active)
                  .map((tab) =>
                    h(
                      "section",
                      {
                        key: tab.key,
                        class: "ui-tabs-content-item ui-tabs-pane py-4",
                      },
                      tab.content(),
                    ),
                  )
              : list.map((tab) =>
                  h(
                    "section",
                    {
                      key: tab.key,
                      class: cx("ui-tabs-content-item ui-tabs-pane py-4", tab.key === active ? "block" : "hidden"),
                    },
                    tab.content(),
                  ),
                ),
          ),
        ),
      ]);
    };
  },
});

export const UiSpace = defineComponent({
  name: "UiSpace",
  props: {
    size: { type: [String, Number], default: 8 },
    direction: { type: String as PropType<"horizontal" | "vertical">, default: "horizontal" },
    wrap: { type: Boolean, default: true },
  },
  setup(props, { slots, attrs }) {
    return () =>
      h(
        "div",
        {
          ...attrs,
          class: cx(
            "ui-space inline-flex",
            props.direction === "vertical" && "ui-space-vertical",
            props.direction === "vertical" ? "flex-col" : "flex-row",
            props.wrap && props.direction !== "vertical" && "flex-wrap",
            attrs.class as string,
          ),
          style: {
            ...(attrs.style as Record<string, unknown>),
            gap: typeof props.size === "number" ? `${props.size}px` : String(props.size),
          },
        },
        slots.default?.(),
      );
  },
});

export const UI_PRIMITIVES = {
  UiButton,
  UiCard,
  UiDivider,
  UiDrawer,
  UiDropdown,
  UiEmpty,
  UiForm,
  UiFormItem,
  UiInput,
  UiInputNumber,
  UiModal,
  UiOption,
  UiPopconfirm,
  UiSelect,
  UiSkeleton,
  UiSkeletonLine,
  UiSpin,
  UiSwitch,
  UiTabPane,
  UiTable,
  UiTableColumn,
  UiTabs,
  UiTag,
  UiTextarea,
  UiSpace,
} as const;

export function installUiComponents(app: App) {
  Object.entries(UI_PRIMITIVES).forEach(([name, component]) => {
    app.component(name, component);
    app.component(toKebabCase(name), component);
  });
}
