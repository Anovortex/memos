import { describe, expect, it } from "vitest";
import { isSettingSectionKey, SETTINGS_SECTIONS } from "@/components/Settings/settingSections";

// Spaces stay hidden until the Teams gate exists (lib/features.ts).
describe("settings sections", () => {
  it("does not list Spaces while the feature is off", () => {
    expect(SETTINGS_SECTIONS.some((section) => section.key === "spaces")).toBe(false);
    expect(isSettingSectionKey("spaces")).toBe(false);
    expect(isSettingSectionKey("member")).toBe(true);
  });
});
