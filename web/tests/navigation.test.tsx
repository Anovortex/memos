import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { User_Role } from "@/types/proto/api/v1/user_service_pb";

const mocks = vi.hoisted(() => ({
  currentUser: undefined as { name: string; role: number } | undefined,
}));

vi.mock("@/hooks/useCurrentUser", () => ({
  default: () => mocks.currentUser,
}));

vi.mock("@/hooks/useUserQueries", () => ({
  useNotifications: () => ({ data: [] }),
}));

vi.mock("@/utils/i18n", () => ({
  useTranslate: () => (key: string) => key,
}));

vi.mock("@/components/MemosLogo", () => ({
  default: () => null,
}));

vi.mock("@/components/UserMenu", () => ({
  default: () => null,
}));

import Navigation from "@/components/Navigation";

const renderNavigation = () =>
  render(
    <MemoryRouter>
      <Navigation />
    </MemoryRouter>,
  );

// The first link wraps the logo and has no text; the rest are the rail entries.
const railTitles = () =>
  screen
    .getAllByRole("link")
    .map((link) => link.textContent)
    .filter(Boolean);

describe("<Navigation>", () => {
  beforeEach(() => {
    mocks.currentUser = undefined;
  });

  it("gives the operator only the dashboard and settings", () => {
    mocks.currentUser = { name: "users/noviledger", role: User_Role.ADMIN };

    renderNavigation();

    expect(railTitles()).toEqual(["common.dashboard", "common.settings"]);
    expect(screen.getAllByRole("link")[0]).toHaveAttribute("href", "/dashboard");
  });

  it("keeps the note surfaces for a member", () => {
    mocks.currentUser = { name: "users/alice", role: User_Role.USER };

    renderNavigation();

    expect(railTitles()).toEqual(["common.memos", "common.explore", "common.attachments", "common.inbox"]);
    expect(screen.getAllByRole("link")[0]).toHaveAttribute("href", "/");
  });

  it("offers explore, about and sign-in to a visitor", () => {
    renderNavigation();

    expect(railTitles()).toEqual(["common.explore", "common.about", "common.sign-in"]);
  });
});
