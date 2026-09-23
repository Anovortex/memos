import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import EmailVerificationBanner from "@/components/EmailVerificationBanner";
import { User_Role } from "@/types/proto/api/v1/user_service_pb";

const mocks = vi.hoisted(() => ({
  currentUser: undefined as unknown,
  sendEmailVerification: vi.fn().mockResolvedValue({}),
}));

vi.mock("@/hooks/useCurrentUser", () => ({ default: () => mocks.currentUser }));
vi.mock("@/connect", () => ({
  authServiceClient: { sendEmailVerification: mocks.sendEmailVerification },
}));
vi.mock("react-hot-toast", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));
vi.mock("@/utils/i18n", () => ({ useTranslate: () => (key: string) => key }));

describe("<EmailVerificationBanner>", () => {
  it("renders for an unverified user and resends on click", async () => {
    mocks.currentUser = { role: User_Role.USER, email: "a@example.com", emailVerified: false };
    render(<EmailVerificationBanner />);

    expect(screen.getByText("auth.verify-email-banner")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "auth.resend-verification" }));
    await waitFor(() => expect(mocks.sendEmailVerification).toHaveBeenCalledTimes(1));
    expect(await screen.findByRole("button", { name: "auth.verification-sent" })).toBeDisabled();
  });

  it("stays hidden for verified users, admins, signed-out visitors, and users without an email", () => {
    for (const user of [
      { role: User_Role.USER, email: "a@example.com", emailVerified: true },
      { role: User_Role.ADMIN, email: "a@example.com", emailVerified: false },
      { role: User_Role.USER, email: "", emailVerified: false },
      undefined,
    ]) {
      mocks.currentUser = user;
      const { container, unmount } = render(<EmailVerificationBanner />);
      expect(container).toBeEmptyDOMElement();
      unmount();
    }
  });
});
