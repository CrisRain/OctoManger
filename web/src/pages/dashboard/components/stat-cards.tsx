import { BellRing, Cable, MailCheck, Shapes, Workflow } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { StatCard } from "@/components/stat-card";
import { Skeleton } from "@/components/ui/skeleton";
import { api, fetchHealth } from "@/lib/api";
import { formatDateTime } from "@/lib/format";

export function StatCards() {
  const healthQuery = useQuery({
    queryKey: ["health"],
    queryFn: fetchHealth,
    refetchInterval: 30000,
  });

  const accountTypesQuery = useQuery({
    queryKey: ["account-types"],
    queryFn: api.listAccountTypes,
  });

  const accountsQuery = useQuery({
    queryKey: ["accounts", "count"],
    queryFn: () => api.listAccounts({ limit: 1 }),
  });

  const emailsQuery = useQuery({
    queryKey: ["email-accounts", "count"],
    queryFn: () => api.listEmailAccounts({ limit: 1 }),
  });

  const jobsSummaryQuery = useQuery({
    queryKey: ["jobs", "summary"],
    queryFn: api.getJobSummary,
  });

  const isLoading =
    healthQuery.isLoading ||
    accountTypesQuery.isLoading ||
    accountsQuery.isLoading ||
    emailsQuery.isLoading ||
    jobsSummaryQuery.isLoading;

  if (isLoading) {
    return (
      <>
        {Array.from({ length: 5 }).map((_, idx) => (
          <Skeleton key={idx} className="h-[132px] rounded-xl" />
        ))}
      </>
    );
  }

  return (
    <>
      <StatCard
        label="账号类型"
        value={accountTypesQuery.data?.length ?? 0}
        hint="可注册的账号类型总数"
        icon={<Shapes className="h-5 w-5" />}
      />
      <StatCard
        label="账号数"
        value={accountsQuery.data?.total ?? 0}
        hint="当前资产账号池规模"
        icon={<Cable className="h-5 w-5" />}
      />
      <StatCard
        label="邮箱账号"
        value={emailsQuery.data?.total ?? 0}
        hint="已接入的邮箱账号数量"
        icon={<MailCheck className="h-5 w-5" />}
      />
      <StatCard
        label="任务数"
        value={jobsSummaryQuery.data?.total ?? 0}
        hint={`失败任务 ${jobsSummaryQuery.data?.failed ?? 0}`}
        icon={<Workflow className="h-5 w-5" />}
      />
      <StatCard
        label="健康状态"
        value={healthQuery.data?.status.toUpperCase() ?? "UNKNOWN"}
        hint={healthQuery.data ? formatDateTime(healthQuery.data.time) : "-"}
        icon={<BellRing className="h-5 w-5" />}
      />
    </>
  );
}
