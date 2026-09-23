import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import Dashboard from "@/pages/Dashboard";

vi.mock("@tanstack/react-query", () => ({
  useQueryClient: () => ({ invalidateQueries: vi.fn() }),
}));

vi.mock("@/hooks/useMediaQuery", () => ({
  default: () => true,
}));

vi.mock("@/components/InviteUserDialog", () => ({
  default: () => null,
}));

vi.mock("@/hooks/useInstanceQueries", () => ({
  instanceKeys: { stats: () => ["instance", "stats"] },
  useInstanceStats: () => ({
    data: {
      database: { driver: "postgres", sizeBytes: 1024n },
      localStorageBytes: 2048n,
      userUsage: [
        { name: "users/noviledger", memoCount: 0, attachmentCount: 0, attachmentBytes: 0n },
        { name: "users/alice", memoCount: 2, attachmentCount: 1, attachmentBytes: 5n, lastActivityTime: { seconds: 100n, nanos: 0 } },
        { name: "users/bob", memoCount: 3, attachmentCount: 2, attachmentBytes: 7n, lastActivityTime: { seconds: 200n, nanos: 0 } },
      ],
    },
    isLoading: false,
    isError: false,
    isFetching: false,
  }),
}));

vi.mock("@/utils/i18n", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/utils/i18n")>()),
  useTranslate: () => (key: string, values?: Record<string, string | number>) =>
    ({
      "dashboard.user-usage.memo-count_other": `${values?.count} memos`,
      "dashboard.user-usage.attachment-count_one": `${values?.count} attachment · ${values?.size}`,
      "dashboard.user-usage.attachment-count_other": `${values?.count} attachments · ${values?.size}`,
    })[key] || key,
}));

const tile = (label: string) => screen.getByText(label).closest("div") as HTMLElement;

describe("<Dashboard>", () => {
  it("sums instance usage into the overview tiles", () => {
    render(
      <MemoryRouter>
        <Dashboard />
      </MemoryRouter>,
    );

    expect(tile("dashboard.members")).toHaveTextContent("3");
    expect(tile("dashboard.notes")).toHaveTextContent("5");
    expect(tile("common.attachments")).toHaveTextContent("3");
    expect(tile("common.attachments")).toHaveTextContent("12 B");
    expect(tile("dashboard.database.title")).toHaveTextContent("1.0 KB");
    expect(tile("dashboard.database.title")).toHaveTextContent("postgres");
    expect(tile("dashboard.local-storage.title")).toHaveTextContent("2.0 KB");
  });

  it("lists members with the most recent activity first", () => {
    render(
      <MemoryRouter>
        <Dashboard />
      </MemoryRouter>,
    );

    expect(screen.getAllByText(/^@/).map((element) => element.textContent)).toEqual(["@bob", "@alice", "@noviledger"]);
    expect(screen.getByText("2 memos")).toBeInTheDocument();
    expect(screen.getByText("1 attachment · 5 B")).toBeInTheDocument();
    expect(screen.getByText("dashboard.manage-members").closest("a")).toHaveAttribute("href", "/setting#member");
  });
});
