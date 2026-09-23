import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import UserMenu from "@/components/UserMenu";

const mocks = vi.hoisted(() => ({
  navigateTo: vi.fn(),
  setMobileOpen: vi.fn(),
  logout: vi.fn(),
  notifications: [] as Array<{ status: number }>,
  currentUser: { name: "users/steven", username: "steven", displayName: "Steven", role: 1 },
  userPackageSetting: undefined as { plan: number } | undefined,
}));

vi.mock("@/components/InviteUserDialog", () => ({
  default: () => <div>invite dialog</div>,
}));

vi.mock("@/contexts/AppSidebarContext", () => ({
  useAppSidebar: () => ({ setMobileOpen: mocks.setMobileOpen }),
}));

vi.mock("@/contexts/AuthContext", () => ({
  useAuth: () => ({
    userGeneralSetting: undefined,
    userPackageSetting: mocks.userPackageSetting,
    refetchSettings: vi.fn(),
    logout: mocks.logout,
  }),
}));

vi.mock("@/hooks/useCurrentUser", () => ({
  default: () => mocks.currentUser,
}));

vi.mock("@/hooks/useLiveMemoRefresh", () => ({
  useSSEConnectionStatus: () => "connected",
}));

vi.mock("@/hooks/useNavigateTo", () => ({
  default: () => mocks.navigateTo,
}));

vi.mock("@/hooks/useUserQueries", () => ({
  useNotifications: () => ({ data: mocks.notifications }),
  useUpdateUserGeneralSetting: () => ({ mutate: vi.fn() }),
}));

vi.mock("@/utils/i18n", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/utils/i18n")>();
  return {
    ...actual,
    getLocaleWithFallback: () => "en",
    loadLocale: vi.fn(),
    useTranslate: () => (key: string) => key,
  };
});

describe("User menu", () => {
  beforeEach(() => {
    mocks.navigateTo.mockReset();
    mocks.setMobileOpen.mockReset();
    mocks.logout.mockReset();
    mocks.notifications = [];
    mocks.currentUser = { name: "users/steven", username: "steven", displayName: "Steven", role: 1 };
    mocks.userPackageSetting = undefined;
  });

  it("keeps the note surfaces out of the operator's menu", async () => {
    mocks.currentUser = { name: "users/noviledger", username: "noviledger", displayName: "Noviledger", role: 2 };

    render(
      <MemoryRouter initialEntries={["/dashboard"]}>
        <UserMenu />
      </MemoryRouter>,
    );
    fireEvent.click(screen.getByRole("button", { name: /Noviledger/ }));

    await screen.findByRole("menuitem", { name: "common.settings" });
    expect(screen.queryByRole("menuitem", { name: "common.profile" })).not.toBeInTheDocument();
    expect(screen.queryByRole("menuitem", { name: "common.inbox" })).not.toBeInTheDocument();
    expect(screen.queryByRole("menuitem", { name: "common.archived" })).not.toBeInTheDocument();
    expect(screen.queryByRole("menuitem", { name: "setting.member.invite" })).not.toBeInTheDocument();
    expect(screen.getAllByRole("separator")).toHaveLength(1);
  });

  it("offers Invite to a member on Teams, and only then", async () => {
    render(
      <MemoryRouter initialEntries={["/"]}>
        <UserMenu />
      </MemoryRouter>,
    );
    fireEvent.click(screen.getByRole("button", { name: /Steven/ }));
    await screen.findByRole("menuitem", { name: "common.profile" });
    expect(screen.queryByRole("menuitem", { name: "setting.member.invite" })).not.toBeInTheDocument();
    expect(screen.queryByText("invite dialog")).not.toBeInTheDocument();
  });

  it("shows Invite and mounts the dialog for a member on Teams", async () => {
    mocks.userPackageSetting = { plan: 2 };

    render(
      <MemoryRouter initialEntries={["/"]}>
        <UserMenu />
      </MemoryRouter>,
    );
    expect(screen.getByText("invite dialog")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Steven/ }));

    const invite = await screen.findByRole("menuitem", { name: "setting.member.invite" });
    const archived = screen.getByRole("menuitem", { name: "common.archived" });
    expect(archived.compareDocumentPosition(invite) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });

  it("groups Inbox and Archived with Profile and marks Archived active", async () => {
    render(
      <MemoryRouter initialEntries={["/archived"]}>
        <UserMenu />
      </MemoryRouter>,
    );

    fireEvent.click(screen.getByRole("button", { name: /Steven/ }));

    const profile = await screen.findByRole("menuitem", { name: "common.profile" });
    const inbox = screen.getByRole("menuitem", { name: "common.inbox" });
    const archived = screen.getByRole("menuitem", { name: "common.archived" });
    expect(profile.compareDocumentPosition(inbox) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(inbox.compareDocumentPosition(archived) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(archived).toHaveAttribute("aria-current", "page");
    expect(screen.getAllByRole("separator")).toHaveLength(2);

    fireEvent.click(archived);
    expect(mocks.setMobileOpen).toHaveBeenCalledWith(false);
    expect(mocks.navigateTo).toHaveBeenCalledWith("/archived");
  });

  it("uses a full-bleed footer trigger, inset menu width, and preserves the Inbox unread state", async () => {
    mocks.notifications = [{ status: 1 }, { status: 1 }, { status: 2 }];
    render(
      <MemoryRouter initialEntries={["/Inbox/"]}>
        <UserMenu />
      </MemoryRouter>,
    );

    const trigger = screen.getByRole("button", { name: "Steven, common.more, 2 inbox.unread" });
    expect(trigger).toHaveClass("h-9", "w-full", "gap-1", "px-5");
    expect(trigger).not.toHaveClass("rounded-md");
    expect(trigger.firstElementChild).toHaveClass("size-5");
    expect(trigger.querySelector(".lucide-ellipsis-vertical")).not.toBeNull();
    expect(trigger.querySelector(".lucide-chevrons-up-down")).toBeNull();
    expect(trigger.querySelector("[data-inbox-unread-indicator]")).not.toBeNull();
    fireEvent.click(trigger);

    const inbox = await screen.findByRole("menuitem", { name: "common.inbox, 2 inbox.unread" });
    const menu = screen.getByRole("menu");
    expect(menu).toHaveClass("w-[calc(var(--anchor-width)-1.5rem)]");
    expect(menu).not.toHaveClass("min-w-56");
    expect(inbox).toHaveAttribute("aria-current", "page");
    expect(inbox).toHaveTextContent("common.inbox");
    expect(inbox).toHaveTextContent("2");

    fireEvent.click(inbox);
    expect(mocks.setMobileOpen).toHaveBeenCalledWith(false);
    expect(mocks.navigateTo).toHaveBeenCalledWith("/inbox");
  });

  it("keeps the collapsed trigger on the artwork rail", () => {
    render(
      <MemoryRouter initialEntries={["/"]}>
        <UserMenu collapsed />
      </MemoryRouter>,
    );

    const trigger = screen.getByRole("button", { name: /Steven/ });
    expect(trigger).toHaveClass("ms-3", "size-9", "rounded-md", "p-2");
    expect(trigger).not.toHaveClass("w-full");
    expect(trigger).not.toHaveClass("px-5");
    expect(trigger).not.toHaveClass("rounded-none");
  });
});
