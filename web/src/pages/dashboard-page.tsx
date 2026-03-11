import { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { PageHeader } from "@/components/page-header";
import { Button } from "@/components/ui/button";
import { RecentJobs } from "@/pages/dashboard/components/recent-jobs";
import { StatCards } from "@/pages/dashboard/components/stat-cards";
import { JobDetails } from "./jobs/components/job-details";

export function DashboardPage() {
  const queryClient = useQueryClient();
  const [detailsOpen, setDetailsOpen] = useState(false);
  const [selectedJobId, setSelectedJobId] = useState<number | null>(null);

  const handleRefresh = () => {
    void queryClient.invalidateQueries({ queryKey: ["health"] });
    void queryClient.invalidateQueries({ queryKey: ["account-types"] });
    void queryClient.invalidateQueries({ queryKey: ["accounts"] });
    void queryClient.invalidateQueries({ queryKey: ["email-accounts"] });
    void queryClient.invalidateQueries({ queryKey: ["jobs"] });
  };

  const handleViewJob = (id: number) => {
    setSelectedJobId(id);
    setDetailsOpen(true);
  };

  return (
    <div className="space-y-4">
      <PageHeader
        title="控制台"
        description="在一个视图里同时跟踪系统健康、账号规模和任务运行情况。"
        action={
          <Button variant="outline" onClick={handleRefresh}>
            刷新总览
          </Button>
        }
      />

      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-5">
        <StatCards />
      </div>

      <RecentJobs onViewJob={handleViewJob} />

      <JobDetails
        jobId={selectedJobId}
        open={detailsOpen}
        onOpenChange={setDetailsOpen}
      />
    </div>
  );
}
