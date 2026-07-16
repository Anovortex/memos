import { describe, expect, it } from "vitest";
import { getSignupUsernameError } from "@/utils/signup";

describe("getSignupUsernameError", () => {
  it("explains that an email address cannot be used as a username", () => {
    expect(getSignupUsernameError("ahkhan.dev@gmail.com")).toBe(
      "Use a username, not an email address. Usernames can contain letters, numbers, and hyphens.",
    );
  });

  it("accepts a valid username", () => {
    expect(getSignupUsernameError("ahkhan-dev")).toBeUndefined();
  });
});
