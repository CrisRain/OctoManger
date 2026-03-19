/**
 * 统一的确认对话框系统
 * 使用浏览器原生 confirm/prompt，避免依赖第三方 UI 组件库。
 */
export const useConfirm = () => {
  const confirm = (
    content: string,
    title = "确认操作",
    options?: {
      okText?: string;
      cancelText?: string;
      okButtonProps?: { status?: "danger" | "success" | "warning" | "normal" };
    },
  ): Promise<boolean> => {
    const prefix = title ? `${title}\n\n` : "";
    const hint = options?.okText ? `\n\n确认: ${options.okText}` : "";
    return Promise.resolve(window.confirm(`${prefix}${content}${hint}`));
  };

  const confirmDanger = (
    content: string,
    title = "危险操作",
    okText = "确认删除",
  ): Promise<boolean> => {
    return confirm(content, title, {
      okText,
      okButtonProps: { status: "danger" },
    });
  };

  const confirmDelete = (itemName?: string): Promise<boolean> => {
    const content = itemName
      ? `确定要删除"${itemName}"吗？此操作无法撤销。`
      : "确定要删除吗？此操作无法撤销。";
    return confirmDanger(content, "确认删除");
  };

  const confirmStop = (itemName?: string): Promise<boolean> => {
    const content = itemName
      ? `确定要停止"${itemName}"吗？`
      : "确定要停止吗？";
    return confirm(content, "确认停止", {
      okText: "停止",
      okButtonProps: { status: "warning" },
    });
  };

  const confirmStart = (itemName?: string): Promise<boolean> => {
    const content = itemName
      ? `确定要启动"${itemName}"吗？`
      : "确定要启动吗？";
    return confirm(content, "确认启动", {
      okText: "启动",
      okButtonProps: { status: "success" },
    });
  };

  const confirmBatch = (
    action: string,
    count: number,
  ): Promise<boolean> => {
    return confirm(
      `确定要对选中的 ${count} 项执行"${action}"操作吗？`,
      `批量${action}`,
      {
        okText: "确定",
        okButtonProps: { status: "normal" },
      },
    );
  };

  const confirmWithInput = (
    content: string,
    title = "确认操作",
    placeholder = "请输入确认",
    options?: {
      requiredText?: string;
      okText?: string;
    },
  ): Promise<boolean> => {
    const requirement = options?.requiredText
      ? `\n\n请输入 ${options.requiredText} 来确认`
      : `\n\n${placeholder}`;
    const value = window.prompt(`${title}\n\n${content}${requirement}`, "");

    if (value === null) {
      return Promise.resolve(false);
    }

    if (options?.requiredText) {
      return Promise.resolve(value === options.requiredText);
    }

    return Promise.resolve(value.trim().length > 0);
  };

  const confirmDangerWithInput = (
    content: string,
    confirmText: string,
    title = "危险操作",
  ): Promise<boolean> => {
    return confirmWithInput(content, title, `请输入 ${confirmText} 确认`, {
      requiredText: confirmText,
      okText: "确认删除",
    });
  };

  return {
    confirm,
    confirmDanger,
    confirmDelete,
    confirmStop,
    confirmStart,
    confirmBatch,
    confirmWithInput,
    confirmDangerWithInput,
  };
};
