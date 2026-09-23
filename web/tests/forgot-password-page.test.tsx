import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import ForgotPassword from "@/pages/ForgotPassword";

const mocks = vi.hoisted(() => ({
  requestPasswordReset: vi.fn().mockResolvedValue({}),
}));

vi.mock("@/connect", () => ({
  authServiceClient: { requestPasswordReset: mocks.requestPasswordReset },
}));
vi.mock("@/contexts/InstanceContext", () => ({
  useInstance: () => ({ generalSetting: {}, profile: { needsSetup: false }, initialize: vi.fn() }),
}));
vi.mock("@/components/AuthFooter", () => ({ default: () => null }));
vi.mock("@/utils/i18n", () => ({ useTranslate: () => (key: string) => key }));

describe("<ForgotPassword>", () => {
  it("requests a reset link and shows the sent state", async () => {
    render(
      <MemoryRouter>
        <ForgotPassword />
      </MemoryRouter>,
    );

    fireEvent.change(screen.getByPlaceholderText("common.email"), { target: { value: " rex@example.com " } });
    fireEvent.click(screen.getByRole("button", { name: "auth.send-reset-link" }));

    await waitFor(() => expect(mocks.requestPasswordReset).toHaveBeenCalledWith({ email: "rex@example.com" }));
    expect(await screen.findByText("auth.reset-link-sent-title")).toBeInTheDocument();
  });
});
