import {
  ArrowLeftRightIcon,
  AstroidIcon,
  CogIcon,
  DatabaseIcon,
  HeartHandshakeIcon,
  KeyIcon,
  KeyRoundIcon,
  LibraryIcon,
  type LucideIcon,
  MailIcon,
  ReceiptIcon,
  Settings2Icon,
  SparklesIcon,
  TagsIcon,
  UserIcon,
  UsersIcon,
  WebhookIcon,
} from "lucide-react";
import { type ComponentType } from "react";
import AccessTokenSection from "@/components/Settings/AccessTokenSection";
import AISection from "@/components/Settings/AISection";
import BillingSection from "@/components/Settings/BillingSection";
import InstanceSection from "@/components/Settings/InstanceSection";
import MemberSection from "@/components/Settings/MemberSection";
import MemoExportSection from "@/components/Settings/MemoExportSection";
import MemoRelatedSettings from "@/components/Settings/MemoRelatedSettings";
import MyAccountSection from "@/components/Settings/MyAccountSection";
import NotificationSection from "@/components/Settings/NotificationSection";
import PlanSection from "@/components/Settings/PlanSection";
import PreferencesSection from "@/components/Settings/PreferencesSection";
import SpacesSection from "@/components/Settings/SpacesSection";
import SSOSection from "@/components/Settings/SSOSection";
import StorageSection from "@/components/Settings/StorageSection";
import TagsSection from "@/components/Settings/TagsSection";
import WebhookSection from "@/components/Settings/WebhookSection";
import { SPACES_ENABLED } from "@/lib/features";
import { InstanceSetting_Key } from "@/types/proto/api/v1/instance_service_pb";

export type SettingSectionKey =
  | "my-account"
  | "plan"
  | "memo-export"
  | "spaces"
  | "access-token"
  | "preference"
  | "webhook"
  | "member"
  | "billing"
  | "system"
  | "memo"
  | "storage"
  | "notification"
  | "sso"
  | "tags"
  | "ai";

// "member" sections are basic sections that only members (not the operator) get.
type SettingSectionScope = "basic" | "member" | "admin";

export interface SettingSectionDefinition {
  key: SettingSectionKey;
  scope: SettingSectionScope;
  labelKey: `setting.${SettingSectionKey}.label`;
  icon: LucideIcon;
  component: ComponentType;
  preloadSettingKeys?: InstanceSetting_Key[];
}

const ALL_SETTINGS_SECTIONS: SettingSectionDefinition[] = [
  {
    key: "my-account",
    scope: "basic",
    labelKey: "setting.my-account.label",
    icon: UserIcon,
    component: MyAccountSection,
  },
  {
    key: "plan",
    scope: "member",
    labelKey: "setting.plan.label",
    icon: SparklesIcon,
    component: PlanSection,
  },
  {
    key: "spaces",
    scope: "basic",
    labelKey: "setting.spaces.label",
    icon: AstroidIcon,
    component: SpacesSection,
  },
  {
    key: "access-token",
    scope: "basic",
    labelKey: "setting.access-token.label",
    icon: KeyRoundIcon,
    component: AccessTokenSection,
  },
  {
    key: "preference",
    scope: "basic",
    labelKey: "setting.preference.label",
    icon: CogIcon,
    component: PreferencesSection,
  },
  {
    key: "webhook",
    scope: "basic",
    labelKey: "setting.webhook.label",
    icon: WebhookIcon,
    component: WebhookSection,
  },
  {
    key: "member",
    scope: "admin",
    labelKey: "setting.member.label",
    icon: UsersIcon,
    component: MemberSection,
  },
  {
    key: "billing",
    scope: "admin",
    labelKey: "setting.billing.label",
    icon: ReceiptIcon,
    component: BillingSection,
    preloadSettingKeys: [InstanceSetting_Key.BILLING],
  },
  {
    key: "system",
    scope: "admin",
    labelKey: "setting.system.label",
    icon: Settings2Icon,
    component: InstanceSection,
  },
  {
    key: "memo",
    scope: "admin",
    labelKey: "setting.memo.label",
    icon: LibraryIcon,
    component: MemoRelatedSettings,
  },
  {
    key: "tags",
    scope: "basic",
    labelKey: "setting.tags.label",
    icon: TagsIcon,
    component: TagsSection,
  },
  {
    key: "memo-export",
    scope: "basic",
    labelKey: "setting.memo-export.label",
    icon: ArrowLeftRightIcon,
    component: MemoExportSection,
  },
  {
    key: "storage",
    scope: "admin",
    labelKey: "setting.storage.label",
    icon: DatabaseIcon,
    component: StorageSection,
    preloadSettingKeys: [InstanceSetting_Key.STORAGE],
  },
  {
    key: "notification",
    scope: "admin",
    labelKey: "setting.notification.label",
    icon: MailIcon,
    component: NotificationSection,
    preloadSettingKeys: [InstanceSetting_Key.NOTIFICATION],
  },
  {
    key: "sso",
    scope: "admin",
    labelKey: "setting.sso.label",
    icon: KeyIcon,
    component: SSOSection,
  },
  {
    key: "ai",
    scope: "admin",
    labelKey: "setting.ai.label",
    icon: HeartHandshakeIcon,
    component: AISection,
    preloadSettingKeys: [InstanceSetting_Key.AI],
  },
];

// Spaces stay hidden until the Teams gate exists (see lib/features.ts).
export const SETTINGS_SECTIONS: SettingSectionDefinition[] = ALL_SETTINGS_SECTIONS.filter(
  (section) => section.key !== "spaces" || SPACES_ENABLED,
);

export const DEFAULT_SETTING_SECTION: SettingSectionKey = "my-account";

export const isSettingSectionKey = (value: string): value is SettingSectionKey => {
  return SETTINGS_SECTIONS.some((section) => section.key === value);
};
