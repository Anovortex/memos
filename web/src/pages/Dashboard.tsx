import { useQueryClient } from "@tanstack/react-query";
import { MailPlusIcon, RefreshCwIcon, UsersIcon } from "lucide-react";
import { useMemo } from "react";
import { Link } from "react-router-dom";
import InviteUserDialog from "@/components/InviteUserDialog";
import MobileHeader from "@/components/MobileHeader";
import { SettingList, SettingListItem } from "@/components/Settings/SettingList";
import { Button } from "@/components/ui/button";
import { useDialog } from "@/hooks/useDialog";
import { instanceKeys, useInstanceStats } from "@/hooks/useInstanceQueries";
import useMediaQuery from "@/hooks/useMediaQuery";
import { cn } from "@/lib/utils";
import { ROUTES } from "@/router/routes";
import type { InstanceStats_UserUsage } from "@/types/proto/api/v1/instance_service_pb";
import { useTranslate } from "@/utils/i18n";

const BYTE_UNITS = ["B", "KB", "MB", "GB", "TB"];
const PENDING = "…";
const relativeTimeFormatter = new Intl.RelativeTimeFormat(undefined, { numeric: "auto" });

const formatBytes = (bytes: number): string => {
  if (bytes < 0) return "—";
  if (bytes === 0) return "0 B";
  const i = Math.min(BYTE_UNITS.length - 1, Math.floor(Math.log(bytes) / Math.log(1024)));
  return `${(bytes / 1024 ** i).toFixed(i === 0 ? 0 : 1)} ${BYTE_UNITS[i]}`;
};

const formatRelativeTime = (date: Date): string => {
  const seconds = Math.max(0, Math.floor((Date.now() - date.getTime()) / 1000));
  if (seconds < 60) return relativeTimeFormatter.format(-seconds, "second");
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return relativeTimeFormatter.format(-minutes, "minute");
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return relativeTimeFormatter.format(-hours, "hour");
  return relativeTimeFormatter.format(-Math.floor(hours / 24), "day");
};

// The server reports -1 (or nothing) when a size could not be measured.
const renderBytes = (value: bigint | number | undefined, unknown: string): string => {
  const n = value === undefined ? -1 : Number(value);
  return n < 0 ? unknown : formatBytes(n);
};

const activitySeconds = (usage: InstanceStats_UserUsage): number => Number(usage.lastActivityTime?.seconds ?? 0);

const sortByActivity = (usage: InstanceStats_UserUsage[]): InstanceStats_UserUsage[] =>
  [...usage].sort((a, b) => activitySeconds(b) - activitySeconds(a) || a.name.localeCompare(b.name));

interface StatTileProps {
  label: string;
  value: string;
  hint?: string;
  className?: string;
}

const StatTile = ({ label, value, hint, className }: StatTileProps) => (
  <div className={cn("flex min-w-0 flex-col gap-1 bg-background px-4 py-4", className)}>
    <span className="text-[11px] font-medium uppercase tracking-[0.14em] text-muted-foreground">{label}</span>
    <span className="truncate font-mono text-2xl font-semibold tabular-nums text-foreground sm:text-3xl">{value}</span>
    {hint && <span className="truncate text-xs text-muted-foreground">{hint}</span>}
  </div>
);

const Dashboard = () => {
  const t = useTranslate();
  const md = useMediaQuery("md");
  const queryClient = useQueryClient();
  const inviteDialog = useDialog();
  const { data, isLoading, isError, isFetching } = useInstanceStats();

  const unknown = t("dashboard.unknown");
  const usage = useMemo(() => sortByActivity(data?.userUsage ?? []), [data]);
  const totals = useMemo(
    () =>
      usage.reduce(
        (acc, item) => ({
          memos: acc.memos + item.memoCount,
          attachments: acc.attachments + item.attachmentCount,
          bytes: acc.bytes + Number(item.attachmentBytes),
        }),
        { memos: 0, attachments: 0, bytes: 0 },
      ),
    [usage],
  );
  const generatedTime = data?.generatedTime
    ? t("dashboard.last-updated", { ago: formatRelativeTime(new Date(Number(data.generatedTime.seconds) * 1000)) })
    : undefined;
  const pending = isLoading && !data;
  const count = (n: number) => (pending ? PENDING : String(n));

  return (
    <section className="@container w-full max-w-5xl min-h-full flex flex-col justify-start items-start sm:pt-3 md:pt-6 pb-8">
      {!md && <MobileHeader />}
      <div className="flex w-full flex-col gap-6 px-4 sm:px-6">
        <header className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
          <div className="min-w-0">
            <h1 className="text-2xl font-semibold text-foreground">{t("dashboard.title")}</h1>
            <p className="mt-1 text-sm text-muted-foreground">{t("dashboard.description")}</p>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            {generatedTime && <span className="text-xs text-muted-foreground">{generatedTime}</span>}
            <Button
              variant="outline"
              size="sm"
              disabled={isFetching}
              onClick={() => void queryClient.invalidateQueries({ queryKey: instanceKeys.stats() })}
            >
              <RefreshCwIcon className="size-4" />
              {t("dashboard.refresh")}
            </Button>
            <Button size="sm" onClick={inviteDialog.open}>
              <MailPlusIcon className="size-4" />
              {t("dashboard.invite")}
            </Button>
          </div>
        </header>

        {isError && <div className="text-sm text-destructive">{t("dashboard.load-error")}</div>}

        {/* Hairline grid: the border colour shows through the 1px gaps between tiles. */}
        <div className="grid grid-cols-2 gap-px overflow-hidden rounded-xl border border-border bg-border sm:grid-cols-5">
          <StatTile label={t("dashboard.members")} value={count(usage.length)} />
          <StatTile label={t("dashboard.notes")} value={count(totals.memos)} />
          <StatTile
            label={t("common.attachments")}
            value={count(totals.attachments)}
            hint={pending ? undefined : formatBytes(totals.bytes)}
          />
          <StatTile
            label={t("dashboard.database.title")}
            value={pending ? PENDING : renderBytes(data?.database?.sizeBytes, unknown)}
            hint={data?.database?.driver || undefined}
          />
          <StatTile
            label={t("dashboard.local-storage.title")}
            value={pending ? PENDING : renderBytes(data?.localStorageBytes, unknown)}
            hint={t("dashboard.local-storage.size")}
            className="col-span-2 sm:col-span-1"
          />
        </div>

        <div className="flex flex-col gap-3">
          <div className="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
            <div className="min-w-0">
              <h2 className="text-base font-semibold text-foreground">{t("dashboard.user-usage.title")}</h2>
              <p className="text-sm text-muted-foreground">{t("dashboard.user-usage.description")}</p>
            </div>
            <Link
              to={`${ROUTES.SETTING}#member`}
              className="inline-flex shrink-0 items-center gap-1 text-sm text-primary hover:underline"
              viewTransition
            >
              <UsersIcon className="size-4" />
              {t("dashboard.manage-members")}
            </Link>
          </div>
          <SettingList>
            {usage.map((item) => {
              const username = item.name.split("/").at(-1) || item.name;
              const activity = item.lastActivityTime
                ? t("dashboard.user-usage.last-activity", {
                    ago: formatRelativeTime(new Date(Number(item.lastActivityTime.seconds) * 1000)),
                  })
                : t("dashboard.user-usage.no-activity");
              const memoCount = t(item.memoCount === 1 ? "dashboard.user-usage.memo-count_one" : "dashboard.user-usage.memo-count_other", {
                count: item.memoCount,
              });
              const attachmentCount = t(
                item.attachmentCount === 1 ? "dashboard.user-usage.attachment-count_one" : "dashboard.user-usage.attachment-count_other",
                { count: item.attachmentCount, size: formatBytes(Number(item.attachmentBytes)) },
              );
              return (
                <SettingListItem
                  key={item.name}
                  label={`@${username}`}
                  description={activity}
                  controlClassName="w-full justify-end sm:w-auto"
                >
                  <div className="flex flex-col items-end gap-1 text-sm tabular-nums">
                    <span>{memoCount}</span>
                    <span className="text-muted-foreground">{attachmentCount}</span>
                  </div>
                </SettingListItem>
              );
            })}
            {pending && <div className="px-3 py-3 text-sm text-muted-foreground">{PENDING}</div>}
          </SettingList>
        </div>
      </div>
      <InviteUserDialog open={inviteDialog.isOpen} onOpenChange={inviteDialog.setOpen} />
    </section>
  );
};

export default Dashboard;
