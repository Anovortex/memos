import { describe, expect, it } from "vitest";
import { getSignupUsernameErrorKey } from "@/utils/signup";

describe("getSignupUsernameErrorKey", () => {
  it("explains that an email address cannot be used as a username", () => {
    expect(getSignupUsernameErrorKey("ahkhan.dev@gmail.com")).toBe("auth.username-email-not-allowed");
  });

  it("accepts a valid username", () => {
    expect(getSignupUsernameErrorKey("ahkhan-dev")).toBeUndefined();
  });
});
