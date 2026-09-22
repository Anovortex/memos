import { Code, ConnectError } from "@connectrpc/connect";
import { describe, expect, it } from "vitest";
import { getErrorMessage } from "@/lib/error";

describe("getErrorMessage", () => {
  it("returns a ConnectError's raw message without the [code] prefix", () => {
    const error = new ConnectError("username is already taken", Code.AlreadyExists);
    expect(error.message).toBe("[already_exists] username is already taken");
    expect(getErrorMessage(error)).toBe("username is already taken");
  });

  it("keeps plain Error messages and the fallback", () => {
    expect(getErrorMessage(new Error("boom"))).toBe("boom");
    expect(getErrorMessage(undefined, "fallback")).toBe("fallback");
  });
});
