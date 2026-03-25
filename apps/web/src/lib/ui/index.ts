import type { App } from "vue";
import { toKebabCase } from "./utils";

export { cx, toKebabCase, flattenNodes, optionValue, getFromPath } from "./utils";

export { default as UiButton } from "./components/UiButton.vue";
export { default as UiCard } from "./components/UiCard.vue";
export { default as UiDivider } from "./components/UiDivider.vue";
export { default as UiDrawer } from "./components/UiDrawer.vue";
export { default as UiDropdown } from "./components/UiDropdown.vue";
export { default as UiEmpty } from "./components/UiEmpty.vue";
export { default as UiForm } from "./components/UiForm.vue";
export { default as UiFormItem } from "./components/UiFormItem.vue";
export { default as UiInput } from "./components/UiInput.vue";
export { default as UiInputNumber } from "./components/UiInputNumber.vue";
export { default as UiModal } from "./components/UiModal.vue";
export { default as UiOption } from "./components/UiOption.vue";
export { default as UiPopconfirm } from "./components/UiPopconfirm.vue";
export { default as UiSelect } from "./components/UiSelect.vue";
export { default as UiSkeleton } from "./components/UiSkeleton.vue";
export { default as UiSkeletonLine } from "./components/UiSkeletonLine.vue";
export { default as UiSpace } from "./components/UiSpace.vue";
export { default as UiSpin } from "./components/UiSpin.vue";
export { default as UiSwitch } from "./components/UiSwitch.vue";
export { default as UiTabPane } from "./components/UiTabPane.vue";
export { default as UiTable } from "./components/UiTable.vue";
export { default as UiTableColumn } from "./components/UiTableColumn.vue";
export { default as UiTabs } from "./components/UiTabs.vue";
export { default as UiTag } from "./components/UiTag.vue";
export { default as UiTextarea } from "./components/UiTextarea.vue";

import UiButton from "./components/UiButton.vue";
import UiCard from "./components/UiCard.vue";
import UiDivider from "./components/UiDivider.vue";
import UiDrawer from "./components/UiDrawer.vue";
import UiDropdown from "./components/UiDropdown.vue";
import UiEmpty from "./components/UiEmpty.vue";
import UiForm from "./components/UiForm.vue";
import UiFormItem from "./components/UiFormItem.vue";
import UiInput from "./components/UiInput.vue";
import UiInputNumber from "./components/UiInputNumber.vue";
import UiModal from "./components/UiModal.vue";
import UiOption from "./components/UiOption.vue";
import UiPopconfirm from "./components/UiPopconfirm.vue";
import UiSelect from "./components/UiSelect.vue";
import UiSkeleton from "./components/UiSkeleton.vue";
import UiSkeletonLine from "./components/UiSkeletonLine.vue";
import UiSpace from "./components/UiSpace.vue";
import UiSpin from "./components/UiSpin.vue";
import UiSwitch from "./components/UiSwitch.vue";
import UiTabPane from "./components/UiTabPane.vue";
import UiTable from "./components/UiTable.vue";
import UiTableColumn from "./components/UiTableColumn.vue";
import UiTabs from "./components/UiTabs.vue";
import UiTag from "./components/UiTag.vue";
import UiTextarea from "./components/UiTextarea.vue";

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
  UiSpace,
  UiSpin,
  UiSwitch,
  UiTabPane,
  UiTable,
  UiTableColumn,
  UiTabs,
  UiTag,
  UiTextarea,
} as const;

export function installUiComponents(app: App) {
  Object.entries(UI_PRIMITIVES).forEach(([name, component]) => {
    app.component(name, component);
    app.component(toKebabCase(name), component);
  });
}
