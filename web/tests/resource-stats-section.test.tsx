import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import ResourceStatsSection from "@/components/Settings/ResourceStatsSection";

vi.mock("@tanstack/react-query", () => ({
  useQueryClient: () => ({ invalidateQueries: vi.fn() }),
}));

vi.mock("@/hooks/useInstanceQueries", () => ({
  instanceKeys: { stats: () => ["instance", "stats"] },
  useInstanceStats: () => ({
    data: {
      database: { driver: "postgres", sizeBytes: 1024n },
      localStorageBytes: 2048n,
      userUsage: [
        {
          name: "users/alice",
          memoCount: 2,
          attachmentCount: 1,
          attachmentBytes: 5n,
        },
      ],
    },
    isLoading: false,
    isError: false,
    isFetching: false,
  }),
}));

vi.mock("@/utils/i18n", () => ({
  useTranslate: () => (key: string) => key,
}));

describe("<ResourceStatsSection>", () => {
  it("shows per-user content and attachment usage", () => {
    render(<ResourceStatsSection />);

    expect(screen.getByText("@alice")).toBeInTheDocument();
    expect(screen.getByText("2 memos")).toBeInTheDocument();
    expect(screen.getByText("1 attachment · 5 B")).toBeInTheDocument();
  });
});
