export const getSignupUsernameErrorKey = (username: string): "auth.username-email-not-allowed" | undefined => {
  if (username.includes("@")) {
    return "auth.username-email-not-allowed";
  }
  return undefined;
};

// Mirrors the server's minimum; the server remains the trust boundary.
export const MIN_PASSWORD_LENGTH = 8;

export const getSignupPasswordErrorKey = (password: string): "auth.password-too-short" | undefined => {
  if (password.length < MIN_PASSWORD_LENGTH) {
    return "auth.password-too-short";
  }
  return undefined;
};

// Reads the email an invite link was issued for, from the unverified JWT
// payload. It only prefills the form; the server checks signature and expiry.
export const getInviteEmail = (inviteToken: string): string => {
  const payload = inviteToken.split(".")[1];
  if (!payload) {
    return "";
  }
  try {
    const base64 = payload.replace(/-/g, "+").replace(/_/g, "/");
    const padded = base64 + "=".repeat((4 - (base64.length % 4)) % 4);
    const bytes = Uint8Array.from(atob(padded), (char) => char.charCodeAt(0));
    const claims: unknown = JSON.parse(new TextDecoder().decode(bytes));
    const email = (claims as { email?: unknown })?.email;
    return typeof email === "string" ? email : "";
  } catch {
    return "";
  }
};
