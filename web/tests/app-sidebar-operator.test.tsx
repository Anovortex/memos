import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { screen, render as testingLibraryRender } from "@testing-library/react";
import { MemoryRouter, useLocation } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import AppSidebar from "@/components/AppSidebar";
import { type MemoFilter } from "@/contexts/MemoFilterContext";
import { resolveCollectionRoute } from "@/router/routes";

type FakeUser = { name: string; username: string; role: number };
const authState = vi.hoisted(() => ({
  currentUser: undefined as { name: string; username: string; role: number } | undefined,
}));
const globalEditorState = vi.hoisted(() => ({ canOpen: true, openEditor: vi.fn() }));
const filterState = vi.hoisted(() => ({ filters: [] as MemoFilter[] }));

vi.mock("@/components/MemosLogo", () => ({
  default: () => <span>Memos logo</span>,
}));
vi.mock("@/components/UserMenu", () => ({
  default: () => <button type="button">User menu</button>,
}));
vi.mock("@/components/CreateSpaceDialog", () => ({ default: () => null }));
vi.mock("@/components/StatisticsView", () => ({ default: () => <div>Calendar</div> }));
vi.mock("@/components/AppSidebar/TagsSection", () => ({ default: () => <div>Tags</div> }));
vi.mock("@/contexts/AppSidebarContext", () => ({
  useAppSidebar: () => ({
    attachmentSection: "all",
    setAttachmentSection: vi.fn(),
    inboxFilter: "all",
    setInboxFilter: vi.fn(),
    memoDetail: undefined,
    setMemoDetail: vi.fn(),
    mobileOpen: false,
    setMobileOpen: vi.fn(),
    quickFindOpen: false,
    setQuickFindOpen: vi.fn(),
    memoScope: "home",
    setMemoScope: vi.fn(),
  }),
}));
vi.mock("@/contexts/AuthContext", () => ({ useAuth: () => ({ isInitialized: true }) }));
vi.mock("@/contexts/GlobalMemoEditorContext", () => ({ useGlobalMemoEditor: () => globalEditorState }));
vi.mock("@/contexts/InstanceContext", () => ({ useInstance: () => ({ isInitialized: true }) }));
vi.mock("@/contexts/MemoFilterContext", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/contexts/MemoFilterContext")>()),
  useMemoFilterContext: () => ({ filters: filterState.filters, memoView: undefined, setMemoView: vi.fn() }),
}));
vi.mock("@/contexts/SpaceContext", () => ({
  useSpaceContext: () => {
    const { spaceName } = resolveCollectionRoute(useLocation().pathname);
    return {
      spaces: [],
      selectedSpace: undefined,
      selectSpace: vi.fn(),
      selectedSpaceName: spaceName,
      memoFilter: undefined,
      duplicateSpaceTitles: new Set<string>(),
      isLoadingSpaces: false,
      isSpacesError: false,
    };
  },
}));
vi.mock("@/hooks/useCurrentUser", () => ({ default: () => authState.currentUser }));
vi.mock("@/hooks/useFilteredMemoStats", () => ({
  useFilteredMemoStats: () => ({ statistics: { activityStats: {}, timeBasis: "create_time" }, tags: {} }),
}));
vi.mock("@/hooks/useAttachmentLibrary", () => ({
  useAttachmentLibraryStats: () => ({ stats: { media: 0, documents: 0, audio: 0, unused: 0 } }),
}));
vi.mock("@/hooks/useMediaQuery", () => ({ default: () => true }));
vi.mock("@/hooks/useUserQueries", () => ({
  userKeys: { memoViews: (parent?: string) => ["users", "memoViews", parent] },
  useMemoViews: () => ({ data: [] }),
  useNotifications: () => ({ data: [] }),
  useUser: () => ({ data: undefined }),
}));
vi.mock("@/i18n", () => ({ default: { language: "en" } }));
vi.mock("@/utils/i18n", () => ({ useTranslate: () => (key: string) => key }));

const renderAt = (path: string) =>
  testingLibraryRender(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter initialEntries={[path]}>
        <AppSidebar />
      </MemoryRouter>
    </QueryClientProvider>,
  );

const operator: FakeUser = { name: "users/noviledger", username: "noviledger", role: 2 };
const member: FakeUser = { name: "users/alice", username: "alice", role: 1 };

describe("App sidebar for the operator", () => {
  beforeEach(() => {
    authState.currentUser = undefined;
    globalEditorState.canOpen = true;
    filterState.filters = [];
  });

  it("gives the operator only the dashboard and settings", () => {
    authState.currentUser = operator;

    renderAt("/dashboard");

    expect(screen.getByRole("link", { name: "common.dashboard" })).toHaveAttribute("aria-current", "page");
    expect(screen.getByRole("link", { name: "common.settings" })).toBeInTheDocument();
    for (const name of ["common.calendar", "common.map", "common.attachments"]) {
      expect(screen.queryByRole("link", { name })).not.toBeInTheDocument();
    }
    expect(screen.queryByRole("button", { name: "common.home" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "common.search" })).not.toBeInTheDocument();
    expect(document.querySelector("[data-new-memo-trigger]")).toBeNull();
    expect(screen.getByText("Memos logo").closest("a")).toHaveAttribute("href", "/dashboard");
  });

  it("keeps the note surfaces for a member, with the brand linking home and no Space switcher", () => {
    authState.currentUser = member;

    renderAt("/");

    for (const name of ["common.calendar", "common.map", "common.attachments"]) {
      expect(screen.getByRole("link", { name })).toBeInTheDocument();
    }
    expect(screen.getByRole("button", { name: "common.home" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "common.search" })).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "common.dashboard" })).not.toBeInTheDocument();
    expect(screen.getByText("Memos logo").closest("a")).toHaveAttribute("href", "/");
  });
});
