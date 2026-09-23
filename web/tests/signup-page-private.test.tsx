import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import SignUp from "@/pages/SignUp";

const state = vi.hoisted(() => ({
  profile: { needsSetup: false, instanceUrl: "" },
}));

vi.mock("@/connect", () => ({
  authServiceClient: { signIn: vi.fn() },
  userServiceClient: { createUser: vi.fn() },
}));
vi.mock("@/contexts/AuthContext", () => ({ useAuth: () => ({ initialize: vi.fn() }) }));
vi.mock("@/contexts/InstanceContext", () => ({
  useInstance: () => ({
    generalSetting: { disallowPasswordAuth: false, disallowUserRegistration: false },
    profile: state.profile,
    initialize: vi.fn(),
  }),
}));
vi.mock("@/hooks/useIdentityProviderQueries", () => ({
  useIdentityProviderList: () => ({ identityProviderList: [], isLoading: false }),
}));
vi.mock("@/hooks/useNavigateTo", () => ({ default: () => vi.fn() }));
vi.mock("@/components/AuthFooter", () => ({ default: () => null }));
vi.mock("@/utils/i18n", () => ({ useTranslate: () => (key: string) => key }));

const renderPage = () =>
  render(
    <MemoryRouter>
      <SignUp />
    </MemoryRouter>,
  );

describe("<SignUp> on a private instance", () => {
  it("shows sign-ups as closed once the instance is set up and has no instance URL", () => {
    state.profile = { needsSetup: false, instanceUrl: "" };
    renderPage();
    expect(screen.getByText("auth.signups-closed-title")).toBeInTheDocument();
    expect(screen.queryByPlaceholderText("common.username")).not.toBeInTheDocument();
  });

  it("shows the form once the instance URL makes it public", () => {
    state.profile = { needsSetup: false, instanceUrl: "https://notes.example.com" };
    renderPage();
    expect(screen.getByPlaceholderText("common.username")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("common.email")).toBeInTheDocument();
  });
});
