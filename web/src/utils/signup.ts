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
