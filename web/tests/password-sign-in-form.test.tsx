import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import PasswordSignInForm from "@/components/PasswordSignInForm";
import { User_Role } from "@/types/proto/api/v1/user_service_pb";

const mocks = vi.hoisted(() => ({
  signIn: vi.fn(),
  navigateTo: vi.fn(),
}));

vi.mock("@/auth-state", () => ({ setAccessToken: vi.fn() }));

vi.mock("@/connect", () => ({
  authServiceClient: { signIn: mocks.signIn },
}));

vi.mock("@/contexts/AuthContext", () => ({
  useAuth: () => ({ initialize: vi.fn() }),
}));

vi.mock("@/hooks/useNavigateTo", () => ({
  default: () => mocks.navigateTo,
}));

vi.mock("@/utils/i18n", () => ({
  useTranslate: () => (key: string) => key,
}));

const signInAs = (role: User_Role, redirectPath?: string) => {
  mocks.signIn.mockResolvedValue({ user: { role }, accessToken: "" });
  render(<PasswordSignInForm redirectPath={redirectPath} />);
  fireEvent.change(screen.getByPlaceholderText("common.username"), { target: { value: "someone" } });
  fireEvent.change(screen.getByPlaceholderText("common.password"), { target: { value: "password123" } });
  fireEvent.click(screen.getByRole("button", { name: "common.sign-in" }));
};

describe("<PasswordSignInForm>", () => {
  it("does not prefill seeded demo credentials", () => {
    render(<PasswordSignInForm />);

    expect(screen.getByPlaceholderText("common.username")).toHaveValue("");
    expect(screen.getByPlaceholderText("common.password")).toHaveValue("");
  });

  it("lands the operator on the dashboard", async () => {
    signInAs(User_Role.ADMIN);

    await waitFor(() => expect(mocks.navigateTo).toHaveBeenCalledWith("/dashboard", { replace: true }));
  });

  it("lands a member on their notes", async () => {
    signInAs(User_Role.USER);

    await waitFor(() => expect(mocks.navigateTo).toHaveBeenCalledWith("/", { replace: true }));
  });

  it("keeps an explicit redirect for the operator", async () => {
    signInAs(User_Role.ADMIN, "/memos/shares/abc");

    await waitFor(() => expect(mocks.navigateTo).toHaveBeenCalledWith("/memos/shares/abc", { replace: true }));
  });
});
