import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import ResetPassword from "@/pages/ResetPassword";

const mocks = vi.hoisted(() => ({
  resetPassword: vi.fn().mockResolvedValue({}),
  toastError: vi.fn(),
}));

vi.mock("react-hot-toast", () => ({ toast: { error: mocks.toastError } }));
vi.mock("@/connect", () => ({
  authServiceClient: { resetPassword: mocks.resetPassword },
}));
vi.mock("@/contexts/InstanceContext", () => ({
  useInstance: () => ({ generalSetting: {}, profile: { needsSetup: false }, initialize: vi.fn() }),
}));
vi.mock("@/components/AuthFooter", () => ({ default: () => null }));
vi.mock("@/utils/i18n", () => ({ useTranslate: () => (key: string) => key }));

const renderAt = (path: string) =>
  render(
    <MemoryRouter initialEntries={[path]}>
      <ResetPassword />
    </MemoryRouter>,
  );

describe("<ResetPassword>", () => {
  beforeEach(() => {
    mocks.resetPassword.mockClear();
    mocks.toastError.mockClear();
  });

  it("explains that a link without user and token is unusable", () => {
    renderAt("/auth/reset-password");
    expect(screen.getByText("auth.verify-link-invalid-title")).toBeInTheDocument();
  });

  it("rejects mismatched passwords without calling the API", () => {
    renderAt("/auth/reset-password?user=rex&token=tok");
    fireEvent.change(screen.getByPlaceholderText("auth.new-password"), { target: { value: "newpassword1" } });
    fireEvent.change(screen.getByPlaceholderText("auth.repeat-new-password"), { target: { value: "different1" } });
    fireEvent.click(screen.getByRole("button", { name: "auth.reset-password" }));

    expect(mocks.toastError).toHaveBeenCalledWith("auth.passwords-do-not-match");
    expect(mocks.resetPassword).not.toHaveBeenCalled();
  });

  it("submits the token from the link and shows the done state", async () => {
    renderAt("/auth/reset-password?user=rex&token=tok");
    fireEvent.change(screen.getByPlaceholderText("auth.new-password"), { target: { value: "newpassword1" } });
    fireEvent.change(screen.getByPlaceholderText("auth.repeat-new-password"), { target: { value: "newpassword1" } });
    fireEvent.click(screen.getByRole("button", { name: "auth.reset-password" }));

    await waitFor(() => expect(mocks.resetPassword).toHaveBeenCalledWith({ name: "users/rex", token: "tok", newPassword: "newpassword1" }));
    expect(await screen.findByText("auth.password-updated-title")).toBeInTheDocument();
  });
});
