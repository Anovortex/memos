import { timestampDate } from "@bufbuild/protobuf/wkt";
import { UserSetting_PackageSetting, UserSetting_PackageSetting_Plan } from "@/types/proto/api/v1/user_service_pb";

// Mirrors plan.Effective on the server: a Teams package counts only until its
// expiry, and nothing else counts at all.
export const isTeamsActive = (pkg: UserSetting_PackageSetting | undefined, now: Date = new Date()): boolean =>
  pkg?.plan === UserSetting_PackageSetting_Plan.TEAMS && (!pkg.expireTime || timestampDate(pkg.expireTime) > now);
