import { LinkIcon, LoaderIcon, MailCheckIcon } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useSearchParams } from "react-router-dom";
import AuthPageLayout, { AuthEmptyState, AuthLinkPrompt } from "@/components/AuthPageLayout";
import { authServiceClient } from "@/connect";
import { useAuth } from "@/contexts/AuthContext";
import { getErrorMessage } from "@/lib/error";
import { ROUTES } from "@/router/routes";
import { useTranslate } from "@/utils/i18n";

type Status = "verifying" | "verified" | "invalid";

// Consumes the token from an emailed verification link. Reachable signed in
// or out; when signed in, the auth context is refreshed so the banner clears.
const VerifyEmail = () => {
  const t = useTranslate();
  const [searchParams] = useSearchParams();
  const { currentUser, initialize } = useAuth();
  const handledRef = useRef(false);
  const [status, setStatus] = useState<Status>("verifying");
  const [detail, setDetail] = useState("");

  useEffect(() => {
    if (handledRef.current) {
      return;
    }
    handledRef.current = true;
    const user = searchParams.get("user") ?? "";
    const token = searchParams.get("token") ?? "";
    if (user === "" || token === "") {
      setStatus("invalid");
      return;
    }
    const verify = async () => {
      try {
        await authServiceClient.verifyEmail({ name: `users/${user}`, token });
        if (currentUser) {
          await initialize();
        }
        setStatus("verified");
      } catch (error: unknown) {
        setDetail(getErrorMessage(error));
        setStatus("invalid");
      }
    };
    void verify();
  }, [searchParams, currentUser, initialize]);

  const continueLink = currentUser ? (
    <AuthLinkPrompt prompt="" to={ROUTES.HOME} label={t("common.home")} />
  ) : (
    <AuthLinkPrompt prompt="" to={ROUTES.AUTH} label={t("common.sign-in")} />
  );

  return (
    <AuthPageLayout title={t("auth.verify-email-title")}>
      {status === "verifying" && (
        <div className="flex items-center justify-center gap-2 py-4 text-sm text-muted-foreground">
          <LoaderIcon className="h-4 w-4 animate-spin" />
          {t("auth.verifying-email")}
        </div>
      )}
      {status === "verified" && (
        <AuthEmptyState
          icon={<MailCheckIcon className="h-5 w-5" />}
          title={t("auth.email-verified-title")}
          description={t("auth.email-verified-description")}
        />
      )}
      {status === "invalid" && (
        <AuthEmptyState
          icon={<LinkIcon className="h-5 w-5" />}
          title={t("auth.verify-link-invalid-title")}
          description={detail || t("auth.verify-link-invalid-description")}
        />
      )}
      {status !== "verifying" && continueLink}
    </AuthPageLayout>
  );
};

export default VerifyEmail;
