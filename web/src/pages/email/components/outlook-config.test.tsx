import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import {
  DEFAULT_OUTLOOK_CONFIG,
  OutlookConfigPanel,
  type OutlookOAuthConfig,
} from "./outlook-config";

vi.mock("sonner", () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}));

function buildConfig(overrides: Partial<OutlookOAuthConfig> = {}): OutlookOAuthConfig {
  return {
    ...DEFAULT_OUTLOOK_CONFIG,
    clientId: "app-client-id",
    redirectUri: `${window.location.origin}/oauth/callback`,
    ...overrides,
  };
}

describe("OutlookConfigPanel", () => {
  it("keeps advanced settings collapsed until the user asks for them", () => {
    render(
      <OutlookConfigPanel
        config={buildConfig()}
        configLoading={false}
        onSave={vi.fn().mockResolvedValue(undefined)}
      />,
    );

    expect(screen.queryByLabelText("应用密钥（可选）")).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /高级设置 \/ 更多参数/ }));

    expect(screen.getByLabelText("应用密钥（可选）")).toBeInTheDocument();
  });

  it("shows friendly validation before saving incomplete setup", async () => {
    const onSave = vi.fn().mockResolvedValue(undefined);

    render(
      <OutlookConfigPanel
        config={buildConfig({ clientId: "" })}
        configLoading={false}
        onSave={onSave}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "保存并继续" }));

    expect(await screen.findByText("请粘贴微软后台里的 Application (client) ID。")).toBeInTheDocument();
    await waitFor(() => expect(onSave).not.toHaveBeenCalled());
  });
});
