import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import CredentialFields from "@/components/CredentialFields";

vi.mock("@/utils/i18n", () => ({
  useTranslate: () => (key: string) => key,
}));

describe("<CredentialFields>", () => {
  it("renders only username and password by default", () => {
    render(
      <CredentialFields
        idPrefix="signin"
        username=""
        password=""
        passwordAutoComplete="current-password"
        onUsernameChange={vi.fn()}
        onPasswordChange={vi.fn()}
      />,
    );

    expect(screen.getByPlaceholderText("common.username")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("common.password")).toBeInTheDocument();
    expect(screen.queryByPlaceholderText("common.email")).not.toBeInTheDocument();
  });

  it("renders a required email field when an email handler is provided", () => {
    render(
      <CredentialFields
        idPrefix="signup"
        username=""
        email=""
        password=""
        passwordAutoComplete="new-password"
        onUsernameChange={vi.fn()}
        onEmailChange={vi.fn()}
        onPasswordChange={vi.fn()}
      />,
    );

    const email = screen.getByPlaceholderText("common.email");
    expect(email).toHaveAttribute("type", "email");
    expect(email).toBeRequired();
  });

  it("locks only the email field when it is read-only", () => {
    render(
      <CredentialFields
        idPrefix="signup"
        username=""
        email="invited@example.com"
        password=""
        passwordAutoComplete="new-password"
        emailReadOnly
        onUsernameChange={vi.fn()}
        onEmailChange={vi.fn()}
        onPasswordChange={vi.fn()}
      />,
    );

    expect(screen.getByPlaceholderText("common.email")).toHaveAttribute("readonly");
    expect(screen.getByPlaceholderText("common.username")).not.toHaveAttribute("readonly");
    expect(screen.getByPlaceholderText("common.password")).not.toHaveAttribute("readonly");
  });
});
