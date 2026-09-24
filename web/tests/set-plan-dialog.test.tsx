import { create } from "@bufbuild/protobuf";
import { timestampDate, timestampFromDate } from "@bufbuild/protobuf/wkt";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import SetPlanDialog from "@/components/SetPlanDialog";
import { type User, UserSetting_PackageSetting_Plan, UserSettingSchema } from "@/types/proto/api/v1/user_service_pb";

const mocks = vi.hoisted(() => ({
  getUserSetting: vi.fn(),
  update: vi.fn(),
  toastSuccess: vi.fn(),
  toastError: vi.fn(),
}));

vi.mock("react-hot-toast", () => ({ toast: { success: mocks.toastSuccess, error: mocks.toastError } }));
vi.mock("@/connect", () => ({ userServiceClient: { getUserSetting: mocks.getUserSetting } }));
vi.mock("@/hooks/useUserQueries", () => ({ useUpdateUserSetting: () => ({ mutateAsync: mocks.update }) }));
vi.mock("@/utils/i18n", () => ({ useTranslate: () => (key: string) => key }));

const alice = { name: "users/alice", username: "alice" } as User;
const packageName = "users/alice/settings/PACKAGE";

const packageSetting = (plan: UserSetting_PackageSetting_Plan, expire?: Date, requestedPlan?: UserSetting_PackageSetting_Plan) =>
  create(UserSettingSchema, {
    name: packageName,
    value: { case: "packageSetting", value: { plan, expireTime: expire ? timestampFromDate(expire) : undefined, requestedPlan } },
  });

describe("<SetPlanDialog>", () => {
  beforeEach(() => {
    mocks.update.mockResolvedValue(undefined);
  });

  it("shows the member's current package", async () => {
    mocks.getUserSetting.mockResolvedValue(packageSetting(UserSetting_PackageSetting_Plan.TEAMS, new Date(2027, 0, 1, 23, 59, 59)));

    render(<SetPlanDialog user={alice} open onOpenChange={vi.fn()} />);

    expect(mocks.getUserSetting).toHaveBeenCalledWith({ name: packageName });
    await waitFor(() => expect(screen.getByLabelText("setting.member.plan-expires")).toHaveValue("2027-01-01"));
    expect(screen.getByRole("radio", { name: "setting.member.plan-teams" })).toHaveAttribute("aria-checked", "true");
  });

  it("pre-selects the plan the member asked for and says so", async () => {
    mocks.getUserSetting.mockResolvedValue(
      packageSetting(UserSetting_PackageSetting_Plan.FREE, undefined, UserSetting_PackageSetting_Plan.TEAMS),
    );

    render(<SetPlanDialog user={alice} open onOpenChange={vi.fn()} />);

    await screen.findByText("setting.member.plan-requested");
    expect(screen.getByRole("radio", { name: "setting.member.plan-teams" })).toHaveAttribute("aria-checked", "true");
  });

  it("saves Teams with the chosen day as the end of that day", async () => {
    mocks.getUserSetting.mockResolvedValue(packageSetting(UserSetting_PackageSetting_Plan.FREE));
    const onOpenChange = vi.fn();

    render(<SetPlanDialog user={alice} open onOpenChange={onOpenChange} />);
    await waitFor(() => expect(mocks.getUserSetting).toHaveBeenCalled());
    fireEvent.click(screen.getByRole("radio", { name: "setting.member.plan-teams" }));
    fireEvent.change(screen.getByLabelText("setting.member.plan-expires"), { target: { value: "2027-06-30" } });
    fireEvent.click(screen.getByRole("button", { name: "common.confirm" }));

    await waitFor(() => expect(mocks.update).toHaveBeenCalledTimes(1));
    const { setting, updateMask } = mocks.update.mock.calls[0][0];
    expect(setting.name).toBe(packageName);
    expect(updateMask).toEqual(["plan", "expire_time"]);
    expect(setting.value.case).toBe("packageSetting");
    expect(setting.value.value.plan).toBe(UserSetting_PackageSetting_Plan.TEAMS);
    expect(timestampDate(setting.value.value.expireTime)).toEqual(new Date(2027, 5, 30, 23, 59, 59));
    expect(mocks.toastSuccess).toHaveBeenCalledWith("setting.member.plan-updated");
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });
});
