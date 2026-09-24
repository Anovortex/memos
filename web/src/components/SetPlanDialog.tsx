import { create } from "@bufbuild/protobuf";
import { timestampDate, timestampFromDate } from "@bufbuild/protobuf/wkt";
import { useEffect, useState } from "react";
import { toast } from "react-hot-toast";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { userServiceClient } from "@/connect";
import useLoading from "@/hooks/useLoading";
import { useUpdateUserSetting } from "@/hooks/useUserQueries";
import { handleError } from "@/lib/error";
import { buildUserSettingName } from "@/lib/resource-names";
import { User, UserSetting_Key, UserSetting_PackageSetting_Plan, UserSettingSchema } from "@/types/proto/api/v1/user_service_pb";
import { useTranslate } from "@/utils/i18n";

interface Props {
  user: User | undefined;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const FORM_ID = "set-plan-form";
const PLAN_OPTIONS = [
  { value: UserSetting_PackageSetting_Plan.FREE, labelKey: "setting.member.plan-free" },
  { value: UserSetting_PackageSetting_Plan.TEAMS, labelKey: "setting.member.plan-teams" },
] as const;

// The native date input speaks "YYYY-MM-DD" in the operator's local calendar;
// the expiry is stored as the end of that day.
const toDateInput = (date: Date): string =>
  `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`;

const endOfDay = (value: string): Date => {
  const [year, month, day] = value.split("-").map(Number);
  return new Date(year, month - 1, day, 23, 59, 59);
};

// Lets the operator put a member on the Free or Teams plan, with an optional lapse date.
function SetPlanDialog({ user, open, onOpenChange }: Props) {
  const t = useTranslate();
  const requestState = useLoading(false);
  const { mutateAsync: updateUserSetting } = useUpdateUserSetting();
  const [plan, setPlan] = useState<UserSetting_PackageSetting_Plan>(UserSetting_PackageSetting_Plan.FREE);
  const [expiry, setExpiry] = useState("");
  const [requested, setRequested] = useState<UserSetting_PackageSetting_Plan>(UserSetting_PackageSetting_Plan.PLAN_UNSPECIFIED);
  const settingName = user ? buildUserSettingName(user.name, UserSetting_Key.PACKAGE) : "";

  // Show the member's current package every time the dialog opens.
  useEffect(() => {
    if (!open || !settingName) {
      return;
    }
    let cancelled = false;
    userServiceClient
      .getUserSetting({ name: settingName })
      .then((setting) => {
        if (cancelled || setting.value.case !== "packageSetting") {
          return;
        }
        const current = setting.value.value;
        setRequested(current.requestedPlan);
        // Start from what the member asked for, when they asked for anything.
        const preselected =
          current.requestedPlan !== UserSetting_PackageSetting_Plan.PLAN_UNSPECIFIED ? current.requestedPlan : current.plan;
        setPlan(preselected === UserSetting_PackageSetting_Plan.TEAMS ? preselected : UserSetting_PackageSetting_Plan.FREE);
        setExpiry(current.expireTime ? toDateInput(timestampDate(current.expireTime)) : "");
      })
      .catch((error: unknown) => handleError(error, toast.error, { context: "Load package" }));
    return () => {
      cancelled = true;
    };
  }, [open, settingName]);

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (!settingName || requestState.isLoading) {
      return;
    }
    try {
      requestState.setLoading();
      await updateUserSetting({
        setting: create(UserSettingSchema, {
          name: settingName,
          value: {
            case: "packageSetting",
            value: { plan, expireTime: expiry ? timestampFromDate(endOfDay(expiry)) : undefined },
          },
        }),
        updateMask: ["plan", "expire_time"],
      });
      requestState.setFinish();
      toast.success(t("setting.member.plan-updated"));
      onOpenChange(false);
    } catch (error: unknown) {
      handleError(error, toast.error, {
        context: "Set package",
        onError: () => requestState.setError(),
      });
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>{t("setting.member.plan-title", { username: user?.username ?? "" })}</DialogTitle>
          <DialogDescription>{t("setting.member.plan-description")}</DialogDescription>
        </DialogHeader>
        <form id={FORM_ID} className="grid gap-4" onSubmit={handleSubmit}>
          {requested !== UserSetting_PackageSetting_Plan.PLAN_UNSPECIFIED && (
            <p className="rounded-md bg-muted/60 px-3 py-2 text-sm text-foreground">
              {t("setting.member.plan-requested", {
                plan: t(PLAN_OPTIONS.find((option) => option.value === requested)?.labelKey ?? "setting.member.plan-free"),
              })}
            </p>
          )}
          <div className="grid gap-2">
            <Label>{t("setting.member.plan-choice")}</Label>
            <RadioGroup
              value={String(plan)}
              onValueChange={(value) => setPlan(Number(value) as UserSetting_PackageSetting_Plan)}
              className="flex flex-row gap-4"
            >
              {PLAN_OPTIONS.map((option) => (
                <div key={option.value} className="flex items-center space-x-2">
                  <RadioGroupItem value={String(option.value)} id={`package-plan-${option.value}`} />
                  <Label htmlFor={`package-plan-${option.value}`}>{t(option.labelKey)}</Label>
                </div>
              ))}
            </RadioGroup>
          </div>
          <div className="grid gap-2">
            <Label htmlFor="package-expiry">{t("setting.member.plan-expires")}</Label>
            <Input id="package-expiry" type="date" value={expiry} onChange={(e) => setExpiry(e.target.value)} />
            <p className="text-xs text-muted-foreground">{t("setting.member.plan-no-expiry")}</p>
          </div>
        </form>
        <DialogFooter>
          <Button variant="ghost" disabled={requestState.isLoading} onClick={() => onOpenChange(false)}>
            {t("common.cancel")}
          </Button>
          <Button type="submit" form={FORM_ID} disabled={requestState.isLoading}>
            {t("common.confirm")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export default SetPlanDialog;
