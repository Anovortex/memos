import { timestampDate } from "@bufbuild/protobuf/wkt";
import { Badge } from "@/components/ui/badge";
import { useUserPlan } from "@/hooks/useUserQueries";
import { UserSetting_PackageSetting_Plan } from "@/types/proto/api/v1/user_service_pb";
import { useTranslate } from "@/utils/i18n";
import { isTeamsActive } from "@/utils/plan";

interface Props {
  userName: string;
}

const planLabelKey = (plan: UserSetting_PackageSetting_Plan) =>
  plan === UserSetting_PackageSetting_Plan.TEAMS ? "setting.member.plan-teams" : "setting.member.plan-free";

// The operator's view of one member's plan: what they are on, until when, and
// what they asked for.
const PlanBadge = ({ userName }: Props) => {
  const t = useTranslate();
  const { data: plan } = useUserPlan(userName);
  const active = isTeamsActive(plan);
  const expiry = active && plan?.expireTime ? timestampDate(plan.expireTime).toLocaleDateString() : "";
  const requested = plan?.requestedPlan ?? UserSetting_PackageSetting_Plan.PLAN_UNSPECIFIED;

  return (
    <>
      <Badge variant={active ? "default" : "secondary"} className="rounded-full px-2.5 py-0.5">
        {t(active ? "setting.member.plan-teams" : "setting.member.plan-free")}
        {expiry && <span className="ms-1 opacity-75">· {expiry}</span>}
      </Badge>
      {requested !== UserSetting_PackageSetting_Plan.PLAN_UNSPECIFIED && (
        <Badge variant="outline" className="rounded-full border-primary/50 px-2.5 py-0.5 text-primary">
          {t("setting.member.plan-requested", { plan: t(planLabelKey(requested)) })}
        </Badge>
      )}
    </>
  );
};

export default PlanBadge;
