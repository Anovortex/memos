import { KeyRoundIcon, LinkIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "react-hot-toast";
import { useSearchParams } from "react-router-dom";
import AuthPageLayout, { AuthEmptyState, AuthLinkPrompt } from "@/components/AuthPageLayout";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { authServiceClient } from "@/connect";
import useLoading from "@/hooks/useLoading";
import { handleError } from "@/lib/error";
import { ROUTES } from "@/router/routes";
import { useTranslate } from "@/utils/i18n";
import { getSignupPasswordErrorKey } from "@/utils/signup";

const ResetPassword = () => {
  const t = useTranslate();
  const [searchParams] = useSearchParams();
  const user = searchParams.get("user") ?? "";
  const token = searchParams.get("token") ?? "";
  const [password, setPassword] = useState("");
  const [repeat, setRepeat] = useState("");
  const [done, setDone] = useState(false);
  const loadingState = useLoading(false);

  const backToSignIn = <AuthLinkPrompt prompt="" to={ROUTES.AUTH} label={t("auth.back-to-sign-in")} />;

  if (user === "" || token === "") {
    return (
      <AuthPageLayout title={t("auth.reset-password-title")}>
        <AuthEmptyState
          icon={<LinkIcon className="h-5 w-5" />}
          title={t("auth.verify-link-invalid-title")}
          description={t("auth.verify-link-invalid-description")}
        />
        {backToSignIn}
      </AuthPageLayout>
    );
  }

  if (done) {
    return (
      <AuthPageLayout title={t("auth.reset-password-title")}>
        <AuthEmptyState
          icon={<KeyRoundIcon className="h-5 w-5" />}
          title={t("auth.password-updated-title")}
          description={t("auth.password-updated-description")}
        />
        {backToSignIn}
      </AuthPageLayout>
    );
  }

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (loadingState.isLoading) {
      return;
    }
    const passwordErrorKey = getSignupPasswordErrorKey(password);
    if (passwordErrorKey) {
      toast.error(t(passwordErrorKey));
      return;
    }
    if (password !== repeat) {
      toast.error(t("auth.passwords-do-not-match"));
      return;
    }
    try {
      loadingState.setLoading();
      await authServiceClient.resetPassword({ name: `users/${user}`, token, newPassword: password });
      setDone(true);
    } catch (error: unknown) {
      handleError(error, toast.error);
    }
    loadingState.setFinish();
  };

  return (
    <AuthPageLayout title={t("auth.reset-password-title")} subtitle={t("auth.reset-password-description")}>
      <form className="flex w-full flex-col gap-4" onSubmit={handleSubmit}>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="reset-password">{t("auth.new-password")}</Label>
          <Input
            id="reset-password"
            type="password"
            placeholder={t("auth.new-password")}
            value={password}
            autoComplete="new-password"
            readOnly={loadingState.isLoading}
            onChange={(e) => setPassword(e.target.value)}
            required
          />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="reset-password-repeat">{t("auth.repeat-new-password")}</Label>
          <Input
            id="reset-password-repeat"
            type="password"
            placeholder={t("auth.repeat-new-password")}
            value={repeat}
            autoComplete="new-password"
            readOnly={loadingState.isLoading}
            onChange={(e) => setRepeat(e.target.value)}
            required
          />
        </div>
        <Button type="submit" disabled={loadingState.isLoading}>
          {t("auth.reset-password")}
        </Button>
      </form>
      {backToSignIn}
    </AuthPageLayout>
  );
};

export default ResetPassword;
