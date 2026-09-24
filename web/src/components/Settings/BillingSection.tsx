import { create } from "@bufbuild/protobuf";
import { isEqual } from "lodash-es";
import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { useInstance } from "@/contexts/InstanceContext";
import {
  InstanceSetting_BillingSetting,
  InstanceSetting_BillingSettingSchema,
  InstanceSetting_Key,
  InstanceSettingSchema,
} from "@/types/proto/api/v1/instance_service_pb";
import { useTranslate } from "@/utils/i18n";
import SettingGroup from "./SettingGroup";
import SettingRow from "./SettingRow";
import SettingSection from "./SettingSection";
import useInstanceSettingUpdater, { buildInstanceSettingName } from "./useInstanceSettingUpdater";

// What members see when they ask for Teams. Billing is manual for now; a
// payment provider later replaces the pay step, not this setting.
const BillingSection = () => {
  const t = useTranslate();
  const saveInstanceSetting = useInstanceSettingUpdater();
  const { billingSetting: original } = useInstance();
  const [setting, setSetting] = useState<InstanceSetting_BillingSetting>(original);

  useEffect(() => {
    setSetting(original);
  }, [original]);

  const update = (partial: Partial<InstanceSetting_BillingSetting>) =>
    setSetting(create(InstanceSetting_BillingSettingSchema, { ...setting, ...partial }));

  const handleSave = async () => {
    await saveInstanceSetting({
      key: InstanceSetting_Key.BILLING,
      setting: create(InstanceSettingSchema, {
        name: buildInstanceSettingName(InstanceSetting_Key.BILLING),
        value: { case: "billingSetting", value: setting },
      }),
      errorContext: "Save billing setting",
    });
  };

  return (
    <SettingSection
      title={t("setting.billing.title")}
      description={t("setting.billing.description")}
      actions={
        <Button size="sm" disabled={isEqual(setting, original)} onClick={handleSave}>
          {t("common.save")}
        </Button>
      }
    >
      <SettingGroup>
        <SettingRow label={t("setting.billing.price")} description={t("setting.billing.price-hint")}>
          <Input
            value={setting.teamsPrice}
            onChange={(e) => update({ teamsPrice: e.target.value })}
            aria-label={t("setting.billing.price")}
          />
        </SettingRow>
        <SettingRow label={t("setting.billing.period")} description={t("setting.billing.period-hint")}>
          <Input
            type="number"
            min={1}
            max={3650}
            className="w-24"
            value={setting.periodDays || ""}
            onChange={(e) => update({ periodDays: Number.parseInt(e.target.value, 10) || 0 })}
            aria-label={t("setting.billing.period")}
          />
        </SettingRow>
        <SettingRow label={t("setting.billing.instructions")} description={t("setting.billing.instructions-hint")} vertical>
          <Textarea
            rows={5}
            value={setting.paymentInstructions}
            onChange={(e) => update({ paymentInstructions: e.target.value })}
            aria-label={t("setting.billing.instructions")}
          />
        </SettingRow>
      </SettingGroup>
    </SettingSection>
  );
};

export default BillingSection;
