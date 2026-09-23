import { create } from "@bufbuild/protobuf";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import PlanSection from "@/components/Settings/PlanSection";
import { UserSetting_PackageSetting_Plan, UserSetting_PackageSettingSchema } from "@/types/proto/api/v1/user_service_pb";

const mocks = vi.hoisted(() => ({
  plan: undefined as unknown,
  update: vi.fn(),
  refetchSettings: vi.fn(),
  toastSuccess: vi.fn(),
  toastError: vi.fn(),
}));

vi.mock("react-hot-toast", () => ({ toast: { success: mocks.toastSuccess, error: mocks.toastError } }));
vi.mock("@/hooks/useCurrentUser", () => ({ default: () => ({ name: "users/alice", username: "alice" }) }));
vi.mock("@/contexts/AuthContext", () => ({ useAuth: () => ({ userPackageSetting: mocks.plan, refetchSettings: mocks.refetchSettings }) }));
vi.mock("@/hooks/useUserQueries", () => ({ useUpdateUserSetting: () => ({ mutateAsync: mocks.update, isPending: false }) }));
vi.mock("@/utils/i18n", () => ({
  useTranslate: () => (key: string, values?: Record<string, string>) => (values?.plan ? `${key}:${values.plan}` : key),
}));

const { FREE, TEAMS, PLAN_UNSPECIFIED } = UserSetting_PackageSetting_Plan;
const planRecord = (plan: UserSetting_PackageSetting_Plan, requestedPlan = PLAN_UNSPECIFIED, expire?: Date) =>
  create(UserSetting_PackageSettingSchema, { plan, requestedPlan, expireTime: expire ? timestampFromDate(expire) : undefined });

describe("<PlanSection>", () => {
  beforeEach(() => {
    mocks.plan = undefined;
    mocks.update.mockResolvedValue(undefined);
    mocks.refetchSettings.mockResolvedValue(undefined);
  });

  it("shows Free and lets the member ask for Teams", async () => {
    render(<PlanSection />);

    expect(screen.getByText("setting.member.plan-free")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "setting.plan.request:setting.member.plan-teams" }));

    await waitFor(() => expect(mocks.update).toHaveBeenCalledTimes(1));
    const { setting, updateMask } = mocks.update.mock.calls[0][0];
    expect(setting.name).toBe("users/alice/settings/PACKAGE");
    expect(updateMask).toEqual(["requested_plan"]);
    expect(setting.value.value.requestedPlan).toBe(TEAMS);
    expect(mocks.refetchSettings).toHaveBeenCalled();
    expect(mocks.toastSuccess).toHaveBeenCalledWith("setting.plan.request-sent");
  });

  it("shows a pending request with a cancel that clears it", async () => {
    mocks.plan = planRecord(FREE, TEAMS);

    render(<PlanSection />);

    expect(screen.getByText("setting.plan.requested:setting.member.plan-teams")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "setting.plan.cancel-request" }));

    await waitFor(() => expect(mocks.update).toHaveBeenCalledTimes(1));
    expect(mocks.update.mock.calls[0][0].setting.value.value.requestedPlan).toBe(PLAN_UNSPECIFIED);
  });

  it("shows Teams with its expiry and offers Free", () => {
    mocks.plan = planRecord(TEAMS, PLAN_UNSPECIFIED, new Date(2030, 0, 15));

    render(<PlanSection />);

    expect(screen.getByText("setting.member.plan-teams")).toBeInTheDocument();
    expect(screen.getByText("setting.plan.expires")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "setting.plan.request:setting.member.plan-free" })).toBeInTheDocument();
  });
});
