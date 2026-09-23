import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import SignUp from "@/pages/SignUp";

const mocks = vi.hoisted(() => ({
  createUser: vi.fn(),
  signIn: vi.fn(),
}));

vi.mock("@/auth-state", () => ({ setAccessToken: vi.fn() }));
vi.mock("@/connect", () => ({
  authServiceClient: { signIn: mocks.signIn },
  userServiceClient: { createUser: mocks.createUser },
}));
vi.mock("@/contexts/AuthContext", () => ({ useAuth: () => ({ initialize: vi.fn() }) }));
vi.mock("@/contexts/InstanceContext", () => ({
  useInstance: () => ({
    generalSetting: { disallowPasswordAuth: false, disallowUserRegistration: true },
    profile: { needsSetup: false, instanceUrl: "https://notes.example.com" },
    initialize: vi.fn(),
  }),
}));
vi.mock("@/hooks/useIdentityProviderQueries", () => ({
  useIdentityProviderList: () => ({ identityProviderList: [], isLoading: false }),
}));
vi.mock("@/hooks/useNavigateTo", () => ({ default: () => vi.fn() }));
vi.mock("@/components/AuthFooter", () => ({ default: () => null }));
vi.mock("@/utils/i18n", () => ({ useTranslate: () => (key: string) => key }));

// Structurally a JWT; the page only reads the payload, the server checks the signature.
const encode = (value: object) =>
  btoa(JSON.stringify(value))
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/, "");
const inviteToken = `${encode({ alg: "HS256", typ: "JWT" })}.${encode({ type: "invite", email: "invited@example.com" })}.signature`;

const renderPage = (path: string) =>
  render(
    <MemoryRouter initialEntries={[path]}>
      <SignUp />
    </MemoryRouter>,
  );

describe("<SignUp> with an invite on a closed instance", () => {
  it("shows sign-ups as closed without an invite", () => {
    renderPage("/auth/signup");

    expect(screen.getByText("auth.signups-closed-title")).toBeInTheDocument();
    expect(screen.queryByPlaceholderText("common.username")).not.toBeInTheDocument();
  });

  it("opens the form with the invited email locked in", () => {
    renderPage(`/auth/signup?invite=${inviteToken}`);

    expect(screen.getByText("auth.invited")).toBeInTheDocument();
    const email = screen.getByPlaceholderText("common.email");
    expect(email).toHaveValue("invited@example.com");
    expect(email).toHaveAttribute("readonly");
    expect(screen.getByPlaceholderText("common.username")).not.toHaveAttribute("readonly");
  });

  it("sends the invite token with the sign-up request", async () => {
    mocks.createUser.mockResolvedValue({});
    mocks.signIn.mockResolvedValue({ accessToken: "" });
    renderPage(`/auth/signup?invite=${inviteToken}`);

    fireEvent.change(screen.getByPlaceholderText("common.username"), { target: { value: "invited" } });
    fireEvent.change(screen.getByPlaceholderText("common.password"), { target: { value: "password123" } });
    fireEvent.click(screen.getByRole("button", { name: "common.sign-up" }));

    await waitFor(() => expect(mocks.createUser).toHaveBeenCalledTimes(1));
    expect(mocks.createUser).toHaveBeenCalledWith(
      expect.objectContaining({
        inviteToken,
        user: expect.objectContaining({ username: "invited", email: "invited@example.com" }),
      }),
    );
  });
});
