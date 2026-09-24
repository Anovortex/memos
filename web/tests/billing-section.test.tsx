import { create } from "@bufbuild/protobuf";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import BillingSection from "@/components/Settings/BillingSection";
import { InstanceSetting_BillingSettingSchema } from "@/types/proto/api/v1/instance_service_pb";

const mocks = vi.hoisted(() => ({ save: vi.fn(), billing: undefined as unknown }));

// One stable object: the section resets its form whenever the setting object changes.
vi.mock("@/contexts/InstanceContext", () => ({
  useInstance: () => {
    mocks.billing ??= create(InstanceSetting_BillingSettingSchema, { teamsPrice: "", paymentInstructions: "", periodDays: 30 });
    return { billingSetting: mocks.billing };
  },
}));
vi.mock("@/components/Settings/useInstanceSettingUpdater", () => ({
  default: () => mocks.save,
  buildInstanceSettingName: (key: number) => `instance/settings/${key === 8 ? "BILLING" : key}`,
}));
vi.mock("@/utils/i18n", () => ({ useTranslate: () => (key: string) => key }));

describe("<BillingSection>", () => {
  it("saves the price, instructions and period as the billing setting", async () => {
    mocks.save.mockResolvedValue(true);
    render(<BillingSection />);

    const saveButton = screen.getByRole("button", { name: "common.save" });
    expect(saveButton).toBeDisabled();
    fireEvent.change(screen.getByLabelText("setting.billing.price"), { target: { value: "500 BDT / month" } });
    fireEvent.change(screen.getByLabelText("setting.billing.period"), { target: { value: "31" } });
    fireEvent.change(screen.getByLabelText("setting.billing.instructions"), { target: { value: "bKash 0170" } });
    fireEvent.click(saveButton);

    await waitFor(() => expect(mocks.save).toHaveBeenCalledTimes(1));
    const { key, setting } = mocks.save.mock.calls[0][0];
    expect(key).toBe(8);
    expect(setting.name).toBe("instance/settings/BILLING");
    expect(setting.value.case).toBe("billingSetting");
    expect(setting.value.value.teamsPrice).toBe("500 BDT / month");
    expect(setting.value.value.periodDays).toBe(31);
    expect(setting.value.value.paymentInstructions).toBe("bKash 0170");
  });
});
