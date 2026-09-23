import { create } from "@bufbuild/protobuf";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import { toast } from "react-hot-toast";
import { Button } from "@/components/ui/button";
import { useAuth } from "@/contexts/AuthContext";
import useCurrentUser from "@/hooks/useCurrentUser";
import { useUpdateUserSetting } from "@/hooks/useUserQueries";
import { handleError } from "@/lib/error";
import { buildUserSettingName } from "@/lib/resource-names";
import { UserSetting_Key, UserSetting_PackageSetting_Plan, UserSettingSchema } from "@/types/proto/api/v1/user_service_pb";
import { useTranslate } from "@/utils/i18n";
import { isTeamsActive } from "@/utils/plan";
import SettingGroup from "./SettingGroup";
import { SettingList, SettingListItem } from "./SettingList";
import SettingSection from "./SettingSection";

const { FREE, TEAMS, PLAN_UNSPECIFIED } = UserSetting_PackageSetting_Plan;

// A member's own plan: what they are on, until when, and a request for the
// other plan that the operator answers from Members.
const PlanSection = () => {
  const t = useTranslate();
  const currentUser = useCurrentUser();
  const { userPackageSetting: plan, refetchSettings } = useAuth();
  const { mutateAsync: updateUserSetting, isPending } = useUpdateUserSetting();

  const active = isTeamsActive(plan);
  const current = active ? TEAMS : FREE;
  const other = active ? FREE : TEAMS;
  const requested = plan?.requestedPlan ?? PLAN_UNSPECIFIED;
  const planName = (value: UserSetting_PackageSetting_Plan) =>
    t(value === TEAMS ? "setting.member.plan-teams" : "setting.member.plan-free");
  const expiry = active && plan?.expireTime ? timestampDate(plan.expireTime).toLocaleDateString() : "";

  const request = async (value: UserSetting_PackageSetting_Plan) => {
    if (!currentUser) return;
    try {
      await updateUserSetting({
        setting: create(UserSettingSchema, {
          name: buildUserSettingName(currentUser.name, UserSetting_Key.PACKAGE),
          value: { case: "packageSetting", value: { requestedPlan: value } },
        }),
        updateMask: ["requested_plan"],
      });
      await refetchSettings();
      toast.success(t(value === PLAN_UNSPECIFIED ? "setting.plan.request-cancelled" : "setting.plan.request-sent"));
    } catch (error: unknown) {
      handleError(error, toast.error, { context: "Request plan" });
    }
  };

  return (
    <SettingSection title={t("setting.plan.title")} description={t("setting.plan.description")}>
      <SettingGroup title={t("setting.plan.current")}>
        <SettingList>
          <SettingListItem
            label={planName(current)}
            description={
              active ? (expiry ? t("setting.plan.expires", { date: expiry }) : t("setting.plan.no-expiry")) : t("setting.plan.teams-perk")
            }
          />
        </SettingList>
      </SettingGroup>
      <SettingGroup title={t("setting.plan.change")} showSeparator>
        <SettingList>
          {requested !== PLAN_UNSPECIFIED ? (
            <SettingListItem
              label={t("setting.plan.requested", { plan: planName(requested) })}
              description={t("setting.plan.requested-note")}
            >
              <Button variant="outline" size="sm" disabled={isPending} onClick={() => request(PLAN_UNSPECIFIED)}>
                {t("setting.plan.cancel-request")}
              </Button>
            </SettingListItem>
          ) : (
            <SettingListItem label={t("setting.plan.request", { plan: planName(other) })} description={t("setting.plan.request-note")}>
              <Button size="sm" disabled={isPending} onClick={() => request(other)}>
                {t("setting.plan.request", { plan: planName(other) })}
              </Button>
            </SettingListItem>
          )}
        </SettingList>
      </SettingGroup>
    </SettingSection>
  );
};

export default PlanSection;
