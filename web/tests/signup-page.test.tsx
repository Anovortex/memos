import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import SignUp from "@/pages/SignUp";

const mocks = vi.hoisted(() => ({
  createUser: vi.fn(),
  signIn: vi.fn(),
  toastError: vi.fn(),
}));

vi.mock("react-hot-toast", () => ({
  toast: { error: mocks.toastError },
}));

vi.mock("@/connect", () => ({
  authServiceClient: { signIn: mocks.signIn },
  userServiceClient: { createUser: mocks.createUser },
}));

vi.mock("@/contexts/AuthContext", () => ({
  useAuth: () => ({ initialize: vi.fn() }),
}));

vi.mock("@/contexts/InstanceContext", () => ({
  useInstance: () => ({
    generalSetting: { disallowPasswordAuth: false, disallowUserRegistration: false },
    profile: { needsSetup: true },
    initialize: vi.fn(),
  }),
}));

vi.mock("@/hooks/useIdentityProviderQueries", () => ({
  useIdentityProviderList: () => ({ identityProviderList: [], isLoading: false }),
}));

vi.mock("@/hooks/useNavigateTo", () => ({
  default: () => vi.fn(),
}));

vi.mock("@/components/AuthFooter", () => ({ default: () => null }));

vi.mock("@/utils/i18n", () => ({
  useTranslate: () => (key: string) => key,
}));

describe("<SignUp>", () => {
  it("explains that only a username is accepted without calling the API", () => {
    render(
      <MemoryRouter>
        <SignUp />
      </MemoryRouter>,
    );

    fireEvent.change(screen.getByPlaceholderText("common.username"), { target: { value: "ahkhan.dev@gmail.com" } });
    fireEvent.change(screen.getByPlaceholderText("common.password"), { target: { value: "password123" } });
    fireEvent.click(screen.getByRole("button", { name: "auth.create-admin-account" }));

    expect(mocks.toastError).toHaveBeenCalledWith("auth.username-email-not-allowed");
    expect(mocks.createUser).not.toHaveBeenCalled();
    expect(mocks.signIn).not.toHaveBeenCalled();
  });
});
