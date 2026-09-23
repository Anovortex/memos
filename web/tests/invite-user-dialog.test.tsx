import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import InviteUserDialog from "@/components/InviteUserDialog";

const mocks = vi.hoisted(() => ({
  currentUser: { name: "users/noviledger", role: 2 },
  createUserInvite: vi.fn(),
  copy: vi.fn(),
  toastSuccess: vi.fn(),
  toastError: vi.fn(),
}));

vi.mock("copy-to-clipboard", () => ({ default: mocks.copy }));
vi.mock("react-hot-toast", () => ({ toast: { success: mocks.toastSuccess, error: mocks.toastError } }));
vi.mock("@/connect", () => ({ userServiceClient: { createUserInvite: mocks.createUserInvite } }));
vi.mock("@/utils/i18n", () => ({ useTranslate: () => (key: string) => key }));
vi.mock("@/hooks/useCurrentUser", () => ({ default: () => mocks.currentUser }));

const link = "https://notes.example.com/auth/signup?invite=token";

const invite = (emailSent: boolean) => ({
  email: "invited@example.com",
  token: "token",
  link,
  emailSent,
  expireTime: timestampFromDate(new Date("2030-01-08T00:00:00Z")),
});

const requestInvite = async () => {
  render(<InviteUserDialog open onOpenChange={vi.fn()} />);
  fireEvent.change(screen.getByPlaceholderText("common.email"), { target: { value: "invited@example.com" } });
  fireEvent.click(screen.getByRole("button", { name: "setting.member.invite" }));
  await screen.findByDisplayValue(link);
};

describe("<InviteUserDialog>", () => {
  it("tells the operator the link creates an operator", () => {
    mocks.currentUser = { name: "users/noviledger", role: 2 };

    render(<InviteUserDialog open onOpenChange={vi.fn()} />);

    expect(screen.getByText("setting.member.invite-operator-title")).toBeInTheDocument();
    expect(screen.getByText("setting.member.invite-operator-description")).toBeInTheDocument();
  });

  it("tells a member the link creates a member", () => {
    mocks.currentUser = { name: "users/alice", role: 1 };

    render(<InviteUserDialog open onOpenChange={vi.fn()} />);

    expect(screen.getByText("setting.member.invite-title")).toBeInTheDocument();
    expect(screen.getByText("setting.member.invite-description")).toBeInTheDocument();
  });

  it("requests an invite for the email and shows the emailed link", async () => {
    mocks.createUserInvite.mockResolvedValue(invite(true));

    await requestInvite();

    expect(mocks.createUserInvite).toHaveBeenCalledWith({ email: "invited@example.com" });
    expect(screen.getByText("setting.member.invite-sent")).toBeInTheDocument();
    expect(screen.queryByPlaceholderText("common.email")).not.toBeInTheDocument();
  });

  it("says when the link was not emailed and copies it", async () => {
    mocks.createUserInvite.mockResolvedValue(invite(false));

    await requestInvite();
    fireEvent.click(screen.getByRole("button", { name: "common.copy" }));

    expect(screen.getByText("setting.member.invite-not-sent")).toBeInTheDocument();
    expect(mocks.copy).toHaveBeenCalledWith(link);
    expect(mocks.toastSuccess).toHaveBeenCalledWith("message.copied");
  });
});
