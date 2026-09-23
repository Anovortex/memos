import { create } from "@bufbuild/protobuf";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import { describe, expect, it } from "vitest";
import { UserSetting_PackageSetting_Plan, UserSetting_PackageSettingSchema } from "@/types/proto/api/v1/user_service_pb";
import { isTeamsActive } from "@/utils/plan";

const now = new Date("2026-09-23T12:00:00Z");
const teams = (expire?: Date) =>
  create(UserSetting_PackageSettingSchema, {
    plan: UserSetting_PackageSetting_Plan.TEAMS,
    expireTime: expire ? timestampFromDate(expire) : undefined,
  });

describe("isTeamsActive", () => {
  it("is false without a package or on Free", () => {
    expect(isTeamsActive(undefined, now)).toBe(false);
    expect(isTeamsActive(create(UserSetting_PackageSettingSchema, { plan: UserSetting_PackageSetting_Plan.FREE }), now)).toBe(false);
  });

  it("is true for Teams without an expiry or before it", () => {
    expect(isTeamsActive(teams(), now)).toBe(true);
    expect(isTeamsActive(teams(new Date("2026-09-24T12:00:00Z")), now)).toBe(true);
  });

  it("is false once the expiry has passed", () => {
    expect(isTeamsActive(teams(now), now)).toBe(false);
    expect(isTeamsActive(teams(new Date("2026-09-22T12:00:00Z")), now)).toBe(false);
  });
});
