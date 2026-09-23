import { timestampDate } from "@bufbuild/protobuf/wkt";
import copy from "copy-to-clipboard";
import { useEffect, useState } from "react";
import { toast } from "react-hot-toast";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { userServiceClient } from "@/connect";
import useCurrentUser from "@/hooks/useCurrentUser";
import useLoading from "@/hooks/useLoading";
import { handleError } from "@/lib/error";
import { UserInvite } from "@/types/proto/api/v1/user_service_pb";
import { useTranslate } from "@/utils/i18n";
import { isSuperUser } from "@/utils/user";

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const FORM_ID = "invite-user-form";

// Issues an invite link for one email address and shows it for copying. The
// link grants the inviter's own role, so the copy says which account it makes.
function InviteUserDialog({ open, onOpenChange }: Props) {
  const t = useTranslate();
  const operator = isSuperUser(useCurrentUser());
  const requestState = useLoading(false);
  const [email, setEmail] = useState("");
  const [invite, setInvite] = useState<UserInvite | undefined>();

  // Start from a blank form every time the dialog opens.
  useEffect(() => {
    if (!open) {
      setEmail("");
      setInvite(undefined);
    }
  }, [open]);

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const address = email.trim();
    if (address === "" || requestState.isLoading) {
      return;
    }
    try {
      requestState.setLoading();
      setInvite(await userServiceClient.createUserInvite({ email: address }));
      requestState.setFinish();
    } catch (error: unknown) {
      handleError(error, toast.error, {
        context: "Invite user",
        onError: () => requestState.setError(),
      });
    }
  };

  const handleCopy = () => {
    if (!invite) {
      return;
    }
    copy(invite.link);
    toast.success(t("message.copied"));
  };

  const expiresOn = invite?.expireTime ? timestampDate(invite.expireTime).toLocaleDateString() : "";

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>{operator ? t("setting.member.invite-operator-title") : t("setting.member.invite-title")}</DialogTitle>
          <DialogDescription>
            {operator ? t("setting.member.invite-operator-description") : t("setting.member.invite-description")}
          </DialogDescription>
        </DialogHeader>
        {invite ? (
          <div className="flex flex-col gap-3">
            <div className="grid gap-2">
              <Label htmlFor="invite-link">{t("setting.member.invite-link")}</Label>
              <div className="flex items-center gap-2">
                <Input id="invite-link" type="text" readOnly value={invite.link} onFocus={(e) => e.currentTarget.select()} />
                <Button type="button" variant="outline" onClick={handleCopy}>
                  {t("common.copy")}
                </Button>
              </div>
            </div>
            <p className="text-sm text-muted-foreground">
              {invite.emailSent ? t("setting.member.invite-sent", { email: invite.email }) : t("setting.member.invite-not-sent")}
            </p>
            <p className="text-xs text-muted-foreground">{t("setting.member.invite-expires", { date: expiresOn })}</p>
          </div>
        ) : (
          <form id={FORM_ID} className="grid gap-2" onSubmit={handleSubmit}>
            <Label htmlFor="invite-email">{t("common.email")}</Label>
            <Input
              id="invite-email"
              type="email"
              placeholder={t("common.email")}
              value={email}
              autoComplete="off"
              autoCapitalize="off"
              spellCheck={false}
              onChange={(e) => setEmail(e.target.value)}
              required
            />
          </form>
        )}
        <DialogFooter>
          <Button variant="ghost" disabled={requestState.isLoading} onClick={() => onOpenChange(false)}>
            {invite ? t("common.close") : t("common.cancel")}
          </Button>
          {!invite && (
            <Button type="submit" form={FORM_ID} disabled={requestState.isLoading}>
              {t("setting.member.invite")}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export default InviteUserDialog;
