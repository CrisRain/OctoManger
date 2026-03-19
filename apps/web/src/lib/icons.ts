import { defineComponent, h, type App } from "vue";

type SvgNode = {
  tag: string;
  attrs: Record<string, string | number>;
};

function createSvgIcon(name: string, nodes: SvgNode[], spin = false) {
  return defineComponent({
    name,
    inheritAttrs: false,
    setup(_, { attrs }) {
      return () =>
        h(
          "svg",
          {
            ...attrs,
            viewBox: "0 0 24 24",
            fill: "none",
            xmlns: "http://www.w3.org/2000/svg",
            stroke: "currentColor",
            "stroke-width": 1.9,
            "stroke-linecap": "round",
            "stroke-linejoin": "round",
            class: [
              "ui-icon inline-flex h-[1em] w-[1em] shrink-0 items-center justify-center leading-none",
              spin ? "animate-spin" : "",
              attrs.class,
            ],
            "aria-hidden": "true",
          },
          nodes.map((node, index) => h(node.tag, { key: index, ...node.attrs })),
        );
    },
  });
}

function toKebabCase(value: string) {
  return value
    .replace(/([a-z0-9])([A-Z])/g, "$1-$2")
    .replace(/([A-Z])([A-Z][a-z])/g, "$1-$2")
    .toLowerCase();
}

const path = (d: string, extra: Record<string, string | number> = {}): SvgNode => ({
  tag: "path",
  attrs: { d, ...extra },
});
const circle = (cx: number, cy: number, r: number, extra: Record<string, string | number> = {}): SvgNode => ({
  tag: "circle",
  attrs: { cx, cy, r, ...extra },
});
const rect = (x: number, y: number, width: number, height: number, rx = 0, extra: Record<string, string | number> = {}): SvgNode => ({
  tag: "rect",
  attrs: { x, y, width, height, rx, ...extra },
});
const line = (x1: number, y1: number, x2: number, y2: number, extra: Record<string, string | number> = {}): SvgNode => ({
  tag: "line",
  attrs: { x1, y1, x2, y2, ...extra },
});
const polyline = (points: string, extra: Record<string, string | number> = {}): SvgNode => ({
  tag: "polyline",
  attrs: { points, ...extra },
});
const polygon = (points: string, extra: Record<string, string | number> = {}): SvgNode => ({
  tag: "polygon",
  attrs: { points, ...extra },
});

export const IconApps = createSvgIcon("IconApps", [
  rect(3.5, 3.5, 7, 7, 1.5),
  rect(13.5, 3.5, 7, 7, 1.5),
  rect(3.5, 13.5, 7, 7, 1.5),
  rect(13.5, 13.5, 7, 7, 1.5),
]);

export const IconArrowRight = createSvgIcon("IconArrowRight", [
  line(5, 12, 19, 12),
  polyline("12 5 19 12 12 19"),
]);

export const IconCheck = createSvgIcon("IconCheck", [polyline("4.5 12.5 9.5 17 19.5 7")]);
export const IconCheckCircle = createSvgIcon("IconCheckCircle", [circle(12, 12, 9), polyline("8 12 11 15 16 9")]);
export const IconClockCircle = createSvgIcon("IconClockCircle", [circle(12, 12, 9), polyline("12 7.5 12 12 15.5 14")]);
export const IconClose = createSvgIcon("IconClose", [line(6, 6, 18, 18), line(18, 6, 6, 18)]);
export const IconCloseCircle = createSvgIcon("IconCloseCircle", [circle(12, 12, 9), line(9, 9, 15, 15), line(15, 9, 9, 15)]);
export const IconCloud = createSvgIcon("IconCloud", [path("M7 18h10a4 4 0 0 0 .4-7.98A6.5 6.5 0 0 0 5.2 11.3 3.7 3.7 0 0 0 7 18Z")]);
export const IconCodeBlock = createSvgIcon("IconCodeBlock", [polyline("8 7 3 12 8 17"), polyline("16 7 21 12 16 17"), line(13, 5, 11, 19)]);
export const IconCommand = createSvgIcon("IconCommand", [
  rect(4, 4, 6, 6, 2),
  rect(14, 4, 6, 6, 2),
  rect(4, 14, 6, 6, 2),
  rect(14, 14, 6, 6, 2),
]);
export const IconCopy = createSvgIcon("IconCopy", [rect(9, 9, 10, 10, 2), rect(5, 5, 10, 10, 2)]);
export const IconDashboard = createSvgIcon("IconDashboard", [
  rect(3, 4, 8, 7, 1.5),
  rect(13, 4, 8, 12, 1.5),
  rect(3, 13, 8, 7, 1.5),
]);
export const IconDelete = createSvgIcon("IconDelete", [
  path("M4 7h16"),
  path("M9 7V4h6v3"),
  path("M7 7l1 12h8l1-12"),
  line(10, 11, 10, 17),
  line(14, 11, 14, 17),
]);
export const IconDownload = createSvgIcon("IconDownload", [line(12, 4, 12, 15), polyline("7 10 12 15 17 10"), line(5, 20, 19, 20)]);
export const IconEdit = createSvgIcon("IconEdit", [path("M4 20h4l10-10-4-4L4 16v4Z"), path("M12.5 7.5l4 4")]);
export const IconEmail = createSvgIcon("IconEmail", [rect(3, 5, 18, 14, 2), polyline("4 7 12 13 20 7")]);
export const IconExclamationCircle = createSvgIcon("IconExclamationCircle", [circle(12, 12, 9), line(12, 7.5, 12, 13), circle(12, 16.8, 0.8, { fill: "currentColor", stroke: "none" })]);
export const IconEye = createSvgIcon("IconEye", [path("M2.5 12s3.5-6 9.5-6 9.5 6 9.5 6-3.5 6-9.5 6-9.5-6-9.5-6Z"), circle(12, 12, 2.5)]);
export const IconEyeInvisible = createSvgIcon("IconEyeInvisible", [path("M3 3l18 18"), path("M10.7 6.2A10.8 10.8 0 0 1 12 6c6 0 9.5 6 9.5 6a17.4 17.4 0 0 1-4.2 4.7"), path("M6.4 6.8A17 17 0 0 0 2.5 12S6 18 12 18c1.7 0 3.2-.4 4.5-1.1"), path("M9.9 9.9A3 3 0 0 0 14.1 14.1")]);
export const IconFile = createSvgIcon("IconFile", [path("M7 3h7l5 5v13H7z"), polyline("14 3 14 8 19 8")]);
export const IconFilter = createSvgIcon("IconFilter", [path("M4 5h16l-6 7v5l-4 2v-7L4 5Z")]);
export const IconFolder = createSvgIcon("IconFolder", [path("M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8.5a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V7Z")]);
export const IconHistory = createSvgIcon("IconHistory", [path("M3.5 12A8.5 8.5 0 1 0 6 6"), polyline("3 4 3 9 8 9"), polyline("12 8 12 12 15 14")]);
export const IconImport = createSvgIcon("IconImport", [polyline("12 4 12 15 8 11"), polyline("12 15 16 11"), path("M4 20h16")]);
export const IconInfoCircle = createSvgIcon("IconInfoCircle", [circle(12, 12, 9), line(12, 10.5, 12, 16), circle(12, 7.5, 0.8, { fill: "currentColor", stroke: "none" })]);
export const IconKey = createSvgIcon("IconKey", [circle(8, 12, 3.5), path("M11.5 12H21"), line(17, 12, 17, 9), line(19, 12, 19, 10)]);
export const IconLayers = createSvgIcon("IconLayers", [polygon("12 3 3 8 12 13 21 8"), polyline("3 12 12 17 21 12"), polyline("3 16 12 21 21 16")]);
export const IconLeft = createSvgIcon("IconLeft", [line(19, 12, 5, 12), polyline("12 5 5 12 12 19")]);
export const IconLink = createSvgIcon("IconLink", [path("M10 14l4-4"), path("M7.5 16.5 5 19a3 3 0 0 1-4-4l2.5-2.5"), path("M16.5 7.5 19 5a3 3 0 1 1 4 4l-2.5 2.5")]);
export const IconLoading = createSvgIcon("IconLoading", [path("M12 3a9 9 0 1 0 9 9")], true);
export const IconLock = createSvgIcon("IconLock", [rect(5, 11, 14, 10, 2), path("M8 11V8a4 4 0 1 1 8 0v3")]);
export const IconMenu = createSvgIcon("IconMenu", [line(4, 7, 20, 7), line(4, 12, 20, 12), line(4, 17, 20, 17)]);
export const IconMore = createSvgIcon("IconMore", [circle(6, 12, 1.5, { fill: "currentColor", stroke: "none" }), circle(12, 12, 1.5, { fill: "currentColor", stroke: "none" }), circle(18, 12, 1.5, { fill: "currentColor", stroke: "none" })]);
export const IconPlayArrow = createSvgIcon("IconPlayArrow", [polygon("8 6 18 12 8 18", { fill: "currentColor", stroke: "none" })]);
export const IconPlus = createSvgIcon("IconPlus", [line(12, 5, 12, 19), line(5, 12, 19, 12)]);
export const IconPause = createSvgIcon("IconPause", [rect(7, 6, 3.5, 12, 1, { fill: "currentColor", stroke: "none" }), rect(13.5, 6, 3.5, 12, 1, { fill: "currentColor", stroke: "none" })]);
export const IconRefresh = createSvgIcon("IconRefresh", [path("M20 11a8 8 0 0 0-14-4"), polyline("6 3 6 8 11 8"), path("M4 13a8 8 0 0 0 14 4"), polyline("18 21 18 16 13 16")]);
export const IconRight = createSvgIcon("IconRight", [line(5, 12, 19, 12), polyline("12 5 19 12 12 19")]);
export const IconRobot = createSvgIcon("IconRobot", [rect(5, 8, 14, 10, 3), line(12, 3, 12, 8), circle(9, 12, 1), circle(15, 12, 1), line(9, 16, 15, 16), line(7, 21, 7, 18), line(17, 21, 17, 18), line(2.5, 11, 5, 11), line(19, 11, 21.5, 11)]);
export const IconSafe = createSvgIcon("IconSafe", [path("M12 3 5.5 5.5v5.8c0 4.1 2.8 7.9 6.5 9.7 3.7-1.8 6.5-5.6 6.5-9.7V5.5L12 3Z"), polyline("9 12.5 11.3 14.8 15.5 10.3")]);
export const IconSave = createSvgIcon("IconSave", [rect(4, 4, 16, 16, 2), rect(8, 4, 7, 5, 1, { fill: "currentColor", stroke: "none", opacity: 0.2 }), rect(8, 14, 8, 5, 1.5), line(16, 4, 16, 9)]);
export const IconSchedule = createSvgIcon("IconSchedule", [rect(4, 5, 16, 15, 2), line(8, 3, 8, 7), line(16, 3, 16, 7), line(4, 10, 20, 10), polyline("12 13 12 16 14.5 17.5")]);
export const IconSearch = createSvgIcon("IconSearch", [circle(11, 11, 6), line(16, 16, 21, 21)]);
export const IconSettings = createSvgIcon("IconSettings", [circle(12, 12, 3), path("M19.4 15a1 1 0 0 0 .2 1.1l.1.1a2 2 0 0 1-2.8 2.8l-.1-.1a1 1 0 0 0-1.1-.2 1 1 0 0 0-.6.9V20a2 2 0 0 1-4 0v-.2a1 1 0 0 0-.7-.9 1 1 0 0 0-1 .2l-.2.1a2 2 0 1 1-2.8-2.8l.1-.1a1 1 0 0 0 .2-1.1 1 1 0 0 0-.9-.6H4a2 2 0 0 1 0-4h.2a1 1 0 0 0 .9-.7 1 1 0 0 0-.2-1L4.8 8a2 2 0 1 1 2.8-2.8l.1.1a1 1 0 0 0 1.1.2h.1a1 1 0 0 0 .6-.9V4a2 2 0 0 1 4 0v.2a1 1 0 0 0 .7.9 1 1 0 0 0 1-.2l.2-.1A2 2 0 1 1 19 7.6l-.1.1a1 1 0 0 0-.2 1.1v.1a1 1 0 0 0 .9.6h.2a2 2 0 0 1 0 4h-.2a1 1 0 0 0-.9.6Z")]);
export const IconStop = createSvgIcon("IconStop", [rect(7, 7, 10, 10, 2, { fill: "currentColor", stroke: "none" })]);
export const IconSync = createSvgIcon("IconSync", [path("M20 7v5h-5"), path("M4 17v-5h5"), path("M7 17a7 7 0 0 0 11-3"), path("M17 7A7 7 0 0 0 6 10")]);
export const IconThunderbolt = createSvgIcon("IconThunderbolt", [polygon("13 2 5 14 11 14 10 22 19 9 13 9", { fill: "currentColor", stroke: "none" })]);
export const IconTool = createSvgIcon("IconTool", [path("M14 6a4 4 0 0 0 4.9 4.9l-8.8 8.8a2 2 0 1 1-2.8-2.8l8.8-8.8A4 4 0 0 0 14 6Z"), path("M14 6 18 2l4 4-4 4")]);
export const IconUser = createSvgIcon("IconUser", [circle(12, 8, 4), path("M5 20a7 7 0 0 1 14 0")]);
export const IconCode = createSvgIcon("IconCode", [polyline("9 7 5 12 9 17"), polyline("15 7 19 12 15 17")]);

export const UI_ICONS = {
  IconApps,
  IconArrowRight,
  IconCheck,
  IconCheckCircle,
  IconClockCircle,
  IconClose,
  IconCloseCircle,
  IconCloud,
  IconCode,
  IconCodeBlock,
  IconCommand,
  IconCopy,
  IconDashboard,
  IconDelete,
  IconDownload,
  IconEdit,
  IconEmail,
  IconExclamationCircle,
  IconEye,
  IconEyeInvisible,
  IconFile,
  IconFilter,
  IconFolder,
  IconHistory,
  IconImport,
  IconInfoCircle,
  IconKey,
  IconLayers,
  IconLeft,
  IconLink,
  IconLoading,
  IconLock,
  IconMenu,
  IconMore,
  IconPause,
  IconPlayArrow,
  IconPlus,
  IconRefresh,
  IconRight,
  IconRobot,
  IconSafe,
  IconSave,
  IconSchedule,
  IconSearch,
  IconSettings,
  IconStop,
  IconSync,
  IconThunderbolt,
  IconTool,
  IconUser,
} as const;

export function installIcons(app: App) {
  Object.entries(UI_ICONS).forEach(([name, component]) => {
    app.component(name, component);
    app.component(toKebabCase(name), component);
  });
}
