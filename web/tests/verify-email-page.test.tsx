import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import VerifyEmail from "@/pages/VerifyEmail";

const mocks = vi.hoisted(() => ({
  verifyEmail: vi.fn(),
  initialize: vi.fn().mockResolvedValue(undefined),
}));

vi.mock("@/connect", () => ({
  authServiceClient: { verifyEmail: mocks.verifyEmail },
}));
vi.mock("@/contexts/AuthContext", () => ({
  useAuth: () => ({ currentUser: undefined, initialize: mocks.initialize }),
}));
vi.mock("@/contexts/InstanceContext", () => ({
  useInstance: () => ({ generalSetting: {}, profile: { needsSetup: false }, initialize: vi.fn() }),
}));
vi.mock("@/components/AuthFooter", () => ({ default: () => null }));
vi.mock("@/utils/i18n", () => ({ useTranslate: () => (key: string) => key }));

const renderAt = (path: string) =>
  render(
    <MemoryRouter initialEntries={[path]}>
      <VerifyEmail />
    </MemoryRouter>,
  );

describe("<VerifyEmail>", () => {
  // Mocks are reset between tests by the vitest config (mockReset: true).
  it("verifies the token from the link", async () => {
    mocks.verifyEmail.mockResolvedValue({});
    renderAt("/auth/verify-email?user=vera&token=tok");

    expect(await screen.findByText("auth.email-verified-title")).toBeInTheDocument();
    expect(mocks.verifyEmail).toHaveBeenCalledWith({ name: "users/vera", token: "tok" });
  });

  it("shows the server's reason when the link is rejected", async () => {
    mocks.verifyEmail.mockImplementation(() => Promise.reject(new Error("this link is invalid or has expired")));
    renderAt("/auth/verify-email?user=vera&token=tok");

    expect(await screen.findByText("auth.verify-link-invalid-title")).toBeInTheDocument();
    expect(screen.getByText("this link is invalid or has expired")).toBeInTheDocument();
  });

  it("treats a link without parameters as invalid without calling the API", async () => {
    renderAt("/auth/verify-email");
    expect(await screen.findByText("auth.verify-link-invalid-title")).toBeInTheDocument();
    expect(mocks.verifyEmail).not.toHaveBeenCalled();
  });
});
