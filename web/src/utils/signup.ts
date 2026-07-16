export const getSignupUsernameErrorKey = (username: string): "auth.username-email-not-allowed" | undefined => {
  if (username.includes("@")) {
    return "auth.username-email-not-allowed";
  }
  return undefined;
};
