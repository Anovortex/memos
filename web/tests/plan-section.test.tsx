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
  fetchSetting: vi.fn(),
  billing: { teamsPrice: "500 BDT / month", paymentInstructions: "bKash 0170 0000000, reference: your username", periodDays: 30 },
  toastSuccess: vi.fn(),
  toastError: vi.fn(),
}));

vi.mock("react-hot-toast", () => ({ toast: { success: mocks.toastSuccess, error: mocks.toastError } }));
vi.mock("@/hooks/useCurrentUser", () => ({ default: () => ({ name: "users/alice", username: "alice" }) }));
vi.mock("@/contexts/AuthContext", () => ({ useAuth: () => ({ userPackageSetting: mocks.plan, refetchSettings: mocks.refetchSettings }) }));
vi.mock("@/contexts/InstanceContext", () => ({ useInstance: () => ({ billingSetting: mocks.billing, fetchSetting: mocks.fetchSetting }) }));
vi.mock("@/hooks/useUserQueries", () => ({ useUpdateUserSetting: () => ({ mutateAsync: mocks.update, isPending: false }) }));
vi.mock("@/utils/i18n", () => ({
  useTranslate: () => (key: string, values?: Record<string, string>) => {
    const value = values?.plan ?? values?.price ?? values?.date;
    return value ? `${key}:${value}` : key;
  },
}));

const { FREE, TEAMS, PLAN_UNSPECIFIED } = UserSetting_PackageSetting_Plan;
const planRecord = (
  plan: UserSetting_PackageSetting_Plan,
  requestedPlan = PLAN_UNSPECIFIED,
  options: { expire?: Date; reportedAt?: Date } = {},
) =>
  create(UserSetting_PackageSettingSchema, {
    plan,
    requestedPlan,
    expireTime: options.expire ? timestampFromDate(options.expire) : undefined,
    paymentReportedTime: options.reportedAt ? timestampFromDate(options.reportedAt) : undefined,
  });
const lastUpdate = () => mocks.update.mock.calls[0][0];

describe("<PlanSection>", () => {
  beforeEach(() => {
    mocks.plan = undefined;
    mocks.update.mockReset();
    mocks.update.mockResolvedValue(undefined);
    mocks.refetchSettings.mockResolvedValue(undefined);
  });

  it("step 1: shows Free with the price and lets the member ask for Teams", async () => {
    render(<PlanSection />);

    expect(mocks.fetchSetting).toHaveBeenCalled();
    expect(screen.getByText("setting.plan.price:500 BDT / month")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "setting.plan.request:setting.member.plan-teams" }));

    await waitFor(() => expect(mocks.update).toHaveBeenCalledTimes(1));
    expect(lastUpdate().setting.name).toBe("users/alice/settings/PACKAGE");
    expect(lastUpdate().updateMask).toEqual(["requested_plan"]);
    expect(lastUpdate().setting.value.value.requestedPlan).toBe(TEAMS);
    expect(mocks.toastSuccess).toHaveBeenCalledWith("setting.plan.request-sent");
  });

  it("step 2: shows how to pay and reports the payment with a reference", async () => {
    mocks.plan = planRecord(FREE, TEAMS);

    render(<PlanSection />);

    expect(screen.getByText(/bKash 0170 0000000/)).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText("setting.plan.reference"), { target: { value: " TXN-42 " } });
    fireEvent.click(screen.getByRole("button", { name: "setting.plan.paid" }));

    await waitFor(() => expect(mocks.update).toHaveBeenCalledTimes(1));
    expect(lastUpdate().updateMask).toEqual(["payment_reference"]);
    expect(lastUpdate().setting.value.value.paymentReference).toBe("TXN-42");
    expect(mocks.toastSuccess).toHaveBeenCalledWith("setting.plan.payment-reported");
  });

  it("step 3: shows the report as waiting, with a cancel that withdraws it", async () => {
    mocks.plan = planRecord(FREE, TEAMS, { reportedAt: new Date(2026, 8, 24) });

    render(<PlanSection />);

    expect(screen.getByText("setting.plan.reported-note")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "setting.plan.paid" })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "setting.plan.cancel-request" }));

    await waitFor(() => expect(mocks.update).toHaveBeenCalledTimes(1));
    expect(lastUpdate().updateMask).toEqual(["requested_plan"]);
    expect(lastUpdate().setting.value.value.requestedPlan).toBe(PLAN_UNSPECIFIED);
  });

  it("on Teams: shows the expiry and offers renewal or Free", () => {
    mocks.plan = planRecord(TEAMS, PLAN_UNSPECIFIED, { expire: new Date(2030, 0, 15) });

    render(<PlanSection />);

    expect(screen.getByText("setting.member.plan-teams")).toBeInTheDocument();
    expect(screen.getByText(/^setting\.plan\.expires:/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "setting.plan.renew" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "setting.plan.request:setting.member.plan-free" })).toBeInTheDocument();
    expect(screen.queryByText("setting.plan.steps")).not.toBeInTheDocument();
  });
});
