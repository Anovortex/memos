import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useTranslate } from "@/utils/i18n";

interface Props {
  idPrefix: string;
  username: string;
  password: string;
  // When onEmailChange is provided an email field renders between username and password.
  email?: string;
  passwordAutoComplete: "current-password" | "new-password";
  readOnly?: boolean;
  onUsernameChange: (username: string) => void;
  onEmailChange?: (email: string) => void;
  onPasswordChange: (password: string) => void;
}

// Username + password field pair shared by the sign-in and sign-up forms.
const CredentialFields = ({
  idPrefix,
  username,
  email,
  password,
  passwordAutoComplete,
  readOnly,
  onUsernameChange,
  onEmailChange,
  onPasswordChange,
}: Props) => {
  const t = useTranslate();

  return (
    <>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor={`${idPrefix}-username`}>{t("common.username")}</Label>
        <Input
          id={`${idPrefix}-username`}
          type="text"
          readOnly={readOnly}
          placeholder={t("common.username")}
          value={username}
          autoComplete="username"
          autoCapitalize="off"
          spellCheck={false}
          onChange={(e) => onUsernameChange(e.target.value)}
          required
        />
      </div>
      {onEmailChange && (
        <div className="flex flex-col gap-1.5">
          <Label htmlFor={`${idPrefix}-email`}>{t("common.email")}</Label>
          <Input
            id={`${idPrefix}-email`}
            type="email"
            readOnly={readOnly}
            placeholder={t("common.email")}
            value={email ?? ""}
            autoComplete="email"
            autoCapitalize="off"
            spellCheck={false}
            onChange={(e) => onEmailChange(e.target.value)}
            required
          />
        </div>
      )}
      <div className="flex flex-col gap-1.5">
        <Label htmlFor={`${idPrefix}-password`}>{t("common.password")}</Label>
        <Input
          id={`${idPrefix}-password`}
          type="password"
          readOnly={readOnly}
          placeholder={t("common.password")}
          value={password}
          autoComplete={passwordAutoComplete}
          autoCapitalize="off"
          spellCheck={false}
          onChange={(e) => onPasswordChange(e.target.value)}
          required
        />
      </div>
    </>
  );
};

export default CredentialFields;
