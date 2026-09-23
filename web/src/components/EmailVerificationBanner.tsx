import { MailWarningIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "react-hot-toast";
import { authServiceClient } from "@/connect";
import useCurrentUser from "@/hooks/useCurrentUser";
import useLoading from "@/hooks/useLoading";
import { handleError } from "@/lib/error";
import { User_Role } from "@/types/proto/api/v1/user_service_pb";
import { useTranslate } from "@/utils/i18n";

// Nudges a signed-in user whose email is not verified yet. Admins are exempt
// server-side, and accounts without an email have nothing to verify.
const EmailVerificationBanner = () => {
  const t = useTranslate();
  const currentUser = useCurrentUser();
  const [sent, setSent] = useState(false);
  const loadingState = useLoading(false);

  if (!currentUser || currentUser.role !== User_Role.USER || !currentUser.email || currentUser.emailVerified) {
    return null;
  }

  const resend = async () => {
    if (loadingState.isLoading) {
      return;
    }
    try {
      loadingState.setLoading();
      await authServiceClient.sendEmailVerification({});
      setSent(true);
      toast.success(t("auth.verification-sent"));
    } catch (error: unknown) {
      handleError(error, toast.error);
    }
    loadingState.setFinish();
  };

  return (
    <div className="static w-full border-b border-border bg-muted/70 px-4 py-2 text-sm text-muted-foreground sm:px-6">
      <div className="mx-auto flex max-w-5xl flex-col items-start gap-1 sm:flex-row sm:items-center sm:justify-center sm:gap-2">
        <MailWarningIcon className="h-4 w-4 shrink-0 text-foreground" aria-hidden="true" />
        <span>{t("auth.verify-email-banner")}</span>
        <button
          type="button"
          className="font-medium text-primary underline-offset-4 hover:underline disabled:opacity-60"
          onClick={resend}
          disabled={sent || loadingState.isLoading}
        >
          {sent ? t("auth.verification-sent") : t("auth.resend-verification")}
        </button>
      </div>
    </div>
  );
};

export default EmailVerificationBanner;
