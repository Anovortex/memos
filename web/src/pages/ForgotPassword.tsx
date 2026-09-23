import { MailCheckIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "react-hot-toast";
import AuthPageLayout, { AuthEmptyState, AuthLinkPrompt } from "@/components/AuthPageLayout";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { authServiceClient } from "@/connect";
import useLoading from "@/hooks/useLoading";
import { handleError } from "@/lib/error";
import { ROUTES } from "@/router/routes";
import { useTranslate } from "@/utils/i18n";

const ForgotPassword = () => {
  const t = useTranslate();
  const [email, setEmail] = useState("");
  const [sent, setSent] = useState(false);
  const loadingState = useLoading(false);

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (email.trim() === "" || loadingState.isLoading) {
      return;
    }
    try {
      loadingState.setLoading();
      await authServiceClient.requestPasswordReset({ email: email.trim() });
      setSent(true);
    } catch (error: unknown) {
      handleError(error, toast.error);
    }
    loadingState.setFinish();
  };

  return (
    <AuthPageLayout title={t("auth.forgot-password-title")} subtitle={sent ? undefined : t("auth.forgot-password-description")}>
      {sent ? (
        <AuthEmptyState
          icon={<MailCheckIcon className="h-5 w-5" />}
          title={t("auth.reset-link-sent-title")}
          description={t("auth.reset-link-sent-description")}
        />
      ) : (
        <form className="flex w-full flex-col gap-4" onSubmit={handleSubmit}>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="forgot-email">{t("common.email")}</Label>
            <Input
              id="forgot-email"
              type="email"
              placeholder={t("common.email")}
              value={email}
              autoComplete="email"
              autoCapitalize="off"
              spellCheck={false}
              readOnly={loadingState.isLoading}
              onChange={(e) => setEmail(e.target.value)}
              required
            />
          </div>
          <Button type="submit" disabled={loadingState.isLoading}>
            {t("auth.send-reset-link")}
          </Button>
        </form>
      )}
      <AuthLinkPrompt prompt="" to={ROUTES.AUTH} label={t("auth.back-to-sign-in")} />
    </AuthPageLayout>
  );
};

export default ForgotPassword;
