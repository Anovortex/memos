import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  currentUser: { name: "users/secondary-admin", role: 2 },
  fetchSettings: vi.fn(),
}));

vi.mock("@/hooks/useCurrentUser", () => ({
  default: () => mocks.currentUser,
}));

vi.mock("@/hooks/useMediaQuery", () => ({
  default: () => true,
}));

vi.mock("@/contexts/InstanceContext", () => ({
  useInstance: () => ({
    profile: { admin: { name: "users/owner", role: 2 } },
    fetchSettings: mocks.fetchSettings,
  }),
}));

vi.mock("@/utils/i18n", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/utils/i18n")>()),
  useTranslate: () => (key: string) => key,
}));

vi.mock("@/components/Settings/settingSections", () => ({
  DEFAULT_SETTING_SECTION: "my-account",
  isSettingSectionKey: (value: string) => value === "my-account" || value === "resource-stats",
  SETTINGS_SECTIONS: [
    {
      key: "my-account",
      scope: "basic",
      labelKey: "setting.my-account",
      icon: () => null,
      component: () => <div>My account section</div>,
    },
    {
      key: "resource-stats",
      scope: "admin",
      labelKey: "setting.resource-stats.label",
      icon: () => null,
      component: () => <div>Resource stats section</div>,
    },
  ],
}));

import Setting from "@/pages/Setting";

describe("<Setting> owner access", () => {
  beforeEach(() => {
    mocks.fetchSettings.mockClear();
  });

  it("hides resource usage from an admin who is not the original owner", () => {
    render(
      <MemoryRouter initialEntries={["/setting#resource-stats"]}>
        <Setting />
      </MemoryRouter>,
    );

    expect(screen.queryByText("setting.resource-stats.label")).not.toBeInTheDocument();
    expect(screen.queryByText("Resource stats section")).not.toBeInTheDocument();
  });
});
