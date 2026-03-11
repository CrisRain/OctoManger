import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { EmailAccountCreate } from "./email-account-create";
import { DEFAULT_OUTLOOK_CONFIG } from "./outlook-config";

vi.mock("@/lib/api", () => ({
  api: {
    buildOutlookAuthorizeURL: vi.fn(),
    exchangeOutlookCode: vi.fn(),
    createEmailAccount: vi.fn(),
  },
  extractErrorMessage: (error: unknown) => (error instanceof Error ? error.message : String(error)),
}));

vi.mock("sonner", () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}));

describe("EmailAccountCreate", () => {
  it("guides the user back to setup when Outlook connection is not ready", () => {
    const onOpenSetup = vi.fn();

    render(
      <EmailAccountCreate
        config={{
          ...DEFAULT_OUTLOOK_CONFIG,
          clientId: "",
          redirectUri: "",
        }}
        onSuccess={vi.fn()}
        onOpenSetup={onOpenSetup}
      />,
    );

    expect(screen.getByText("当前还不能直接添加邮箱")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "开始授权并添加" })).toBeDisabled();

    fireEvent.click(screen.getByRole("button", { name: "去完成连接设置" }));

    expect(onOpenSetup).toHaveBeenCalledTimes(1);
  });
});
