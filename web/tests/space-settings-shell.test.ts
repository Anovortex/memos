import { AstroidIcon } from "lucide-react";
import { describe, expect, it, vi } from "vitest";
import { isSettingSectionKey, SETTINGS_SECTIONS } from "@/components/Settings/settingSections";
import enTranslation from "@/locales/en.json";

// Upstream's Spaces UI is hidden by default (lib/features.ts); these tests describe it switched on.
vi.mock("@/lib/features", () => ({ SPACES_ENABLED: true }));

describe("Spaces settings shell", () => {
  it("registers Spaces as a basic settings section", () => {
    expect(isSettingSectionKey("spaces")).toBe(true);
    expect(SETTINGS_SECTIONS.find((section) => section.key === "spaces")).toMatchObject({
      scope: "basic",
      labelKey: "setting.spaces.label",
      icon: AstroidIcon,
    });
  });

  it("defines the English settings copy used as the locale fallback", () => {
    expect(enTranslation.setting.spaces).toMatchObject({
      description: "View spaces you belong to and manage members and invitations.",
      label: "Spaces",
      title: "Spaces",
    });
  });
});
