import { create } from "@bufbuild/protobuf";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import { CheckIcon } from "lucide-react";
import { useEffect, useState } from "react";
import { toast } from "react-hot-toast";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useAuth } from "@/contexts/AuthContext";
import { useInstance } from "@/contexts/InstanceContext";
import useCurrentUser from "@/hooks/useCurrentUser";
import { useUpdateUserSetting } from "@/hooks/useUserQueries";
import { handleError } from "@/lib/error";
import { buildUserSettingName } from "@/lib/resource-names";
import { cn } from "@/lib/utils";
import { InstanceSetting_Key } from "@/types/proto/api/v1/instance_service_pb";
import { UserSetting_Key, UserSetting_PackageSetting_Plan, UserSettingSchema } from "@/types/proto/api/v1/user_service_pb";
import { useTranslate } from "@/utils/i18n";
import { isTeamsActive } from "@/utils/plan";
import SettingGroup from "./SettingGroup";
import { SettingList, SettingListItem } from "./SettingList";
import SettingSection from "./SettingSection";

const { FREE, TEAMS, PLAN_UNSPECIFIED } = UserSetting_PackageSetting_Plan;

type StepState = "done" | "current" | "pending";

const StepMark = ({ index, state }: { index: number; state: StepState }) => (
  <span
    aria-hidden="true"
    className={cn(
      "flex size-6 shrink-0 items-center justify-center rounded-full border font-mono text-xs",
      state === "done" && "border-primary bg-primary text-primary-foreground",
      state === "current" && "border-primary text-primary",
      state === "pending" && "border-border text-muted-foreground",
    )}
  >
    {state === "done" ? <CheckIcon className="size-3.5" /> : index}
  </span>
);

// A member's own plan and the three manual steps to Teams: ask, pay the way
// Noviledger described, and wait for confirmation. A payment provider later
// replaces step 2 only.
const PlanSection = () => {
  const t = useTranslate();
  const currentUser = useCurrentUser();
  const { userPackageSetting: plan, refetchSettings } = useAuth();
  const { billingSetting, fetchSetting } = useInstance();
  const { mutateAsync: updateUserSetting, isPending } = useUpdateUserSetting();
  const [reference, setReference] = useState("");

  useEffect(() => {
    void fetchSetting(InstanceSetting_Key.BILLING);
  }, [fetchSetting]);

  const active = isTeamsActive(plan);
  const requested = plan?.requestedPlan ?? PLAN_UNSPECIFIED;
  const reportedAt = plan?.paymentReportedTime ? timestampDate(plan.paymentReportedTime).toLocaleDateString() : "";
  const expiry = active && plan?.expireTime ? timestampDate(plan.expireTime).toLocaleDateString() : "";
  const planName = (value: UserSetting_PackageSetting_Plan) =>
    t(value === TEAMS ? "setting.member.plan-teams" : "setting.member.plan-free");

  const save = async (value: Record<string, unknown>, mask: string[], successKey: string) => {
    if (!currentUser) return;
    try {
      await updateUserSetting({
        setting: create(UserSettingSchema, {
          name: buildUserSettingName(currentUser.name, UserSetting_Key.PACKAGE),
          value: { case: "packageSetting", value },
        }),
        updateMask: mask,
      });
      await refetchSettings();
      toast.success(t(successKey as Parameters<typeof t>[0]));
    } catch (error: unknown) {
      handleError(error, toast.error, { context: "Update plan" });
    }
  };
  const request = (value: UserSetting_PackageSetting_Plan) =>
    save({ requestedPlan: value }, ["requested_plan"], "setting.plan.request-sent");
  const cancel = () => save({ requestedPlan: PLAN_UNSPECIFIED }, ["requested_plan"], "setting.plan.request-cancelled");
  const reportPayment = () => save({ paymentReference: reference.trim() }, ["payment_reference"], "setting.plan.payment-reported");

  // Which of the three steps the member is on.
  const teamsRequested = requested === TEAMS;
  const step: 1 | 2 | 3 = !teamsRequested ? 1 : reportedAt ? 3 : 2;
  const stateOf = (index: 1 | 2 | 3): StepState => (index < step ? "done" : index === step ? "current" : "pending");
  const cancelButton = (
    <Button variant="outline" size="sm" disabled={isPending} onClick={cancel}>
      {t("setting.plan.cancel-request")}
    </Button>
  );

  return (
    <SettingSection title={t("setting.plan.title")} description={t("setting.plan.description")}>
      <SettingGroup title={t("setting.plan.current")}>
        <SettingList>
          <SettingListItem
            label={planName(active ? TEAMS : FREE)}
            description={
              active ? (expiry ? t("setting.plan.expires", { date: expiry }) : t("setting.plan.no-expiry")) : t("setting.plan.teams-perk")
            }
          >
            {active && !teamsRequested && requested === PLAN_UNSPECIFIED && (
              <div className="flex flex-wrap items-center gap-2">
                <Button size="sm" disabled={isPending} onClick={() => request(TEAMS)}>
                  {t("setting.plan.renew")}
                </Button>
                <Button variant="outline" size="sm" disabled={isPending} onClick={() => request(FREE)}>
                  {t("setting.plan.request", { plan: planName(FREE) })}
                </Button>
              </div>
            )}
          </SettingListItem>
          {requested === FREE && (
            <SettingListItem label={t("setting.plan.requested", { plan: planName(FREE) })} description={t("setting.plan.downgrade-note")}>
              {cancelButton}
            </SettingListItem>
          )}
        </SettingList>
      </SettingGroup>

      {(!active || teamsRequested) && (
        <SettingGroup title={t("setting.plan.steps")} showSeparator>
          <SettingList>
            <SettingListItem
              icon={<StepMark index={1} state={stateOf(1)} />}
              label={t("setting.plan.step-request")}
              description={
                billingSetting.teamsPrice ? t("setting.plan.price", { price: billingSetting.teamsPrice }) : t("setting.plan.request-note")
              }
            >
              {step === 1 && (
                <Button size="sm" disabled={isPending} onClick={() => request(TEAMS)}>
                  {t("setting.plan.request", { plan: planName(TEAMS) })}
                </Button>
              )}
            </SettingListItem>
            <SettingListItem
              icon={<StepMark index={2} state={stateOf(2)} />}
              label={t("setting.plan.step-pay")}
              description={step === 2 ? undefined : step === 3 ? t("setting.plan.reported", { date: reportedAt }) : undefined}
              vertical={step === 2}
            >
              {step === 2 && (
                <div className="flex w-full flex-col gap-3">
                  <p className="whitespace-pre-wrap rounded-md bg-muted/60 px-3 py-2 text-sm text-foreground">
                    {billingSetting.paymentInstructions || t("setting.plan.no-instructions")}
                  </p>
                  <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
                    <Input
                      value={reference}
                      onChange={(e) => setReference(e.target.value)}
                      placeholder={t("setting.plan.reference")}
                      aria-label={t("setting.plan.reference")}
                      maxLength={200}
                    />
                    <Button size="sm" disabled={isPending} onClick={reportPayment} className="shrink-0">
                      {t("setting.plan.paid")}
                    </Button>
                    {cancelButton}
                  </div>
                  <p className="text-xs text-muted-foreground">{t("setting.plan.reference-hint")}</p>
                </div>
              )}
            </SettingListItem>
            <SettingListItem
              icon={<StepMark index={3} state={stateOf(3)} />}
              label={t("setting.plan.step-confirm")}
              description={step === 3 ? t("setting.plan.reported-note") : undefined}
            >
              {step === 3 && cancelButton}
            </SettingListItem>
          </SettingList>
        </SettingGroup>
      )}
    </SettingSection>
  );
};

export default PlanSection;
