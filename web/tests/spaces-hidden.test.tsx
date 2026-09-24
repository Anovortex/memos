import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import { SpaceRoute } from "@/router/SpaceRoute";

vi.mock("@/contexts/SpaceContext", () => ({
  useSpaceContext: () => ({ selectedSpaceName: "spaces/abc", isSpaceReady: true, spaceError: undefined, retrySpace: vi.fn() }),
}));
vi.mock("@/pages/NotFound", () => ({ default: () => <div>not found</div> }));
vi.mock("@/utils/i18n", () => ({ useTranslate: () => (key: string) => key }));

// Spaces stay hidden until the Teams gate exists (lib/features.ts is off by default).
describe("Spaces hidden", () => {
  it("answers a Space URL with not found even when the Space would resolve", () => {
    render(
      <MemoryRouter initialEntries={["/spaces/abc"]}>
        <SpaceRoute />
      </MemoryRouter>,
    );

    expect(screen.getByText("not found")).toBeInTheDocument();
  });
});
