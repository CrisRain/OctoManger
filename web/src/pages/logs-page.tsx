import { useCallback, useEffect, useMemo, useState } from "react";
import { Eye, RefreshCw, Search } from "lucide-react";
import { useSearchParams } from "react-router-dom";
import { toast } from "sonner";
import { PageHeader } from "@/components/page-header";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Pagination } from "@/components/ui/pagination";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { api, extractErrorMessage } from "@/lib/api";
import { compactId, formatDateTime } from "@/lib/format";
import type { JobRun } from "@/types";
import { JobRunDetails } from "./jobs/components/job-run-details";

const runStatusMap: Record<JobRun["status"], { label: string; variant: "default" | "secondary" | "destructive" | "outline" }> = {
  running: { label: "运行中", variant: "default" },
  success: { label: "成功", variant: "outline" },
  failed: { label: "失败", variant: "destructive" }
};

function formatDuration(startedAt: string, endedAt?: string) {
  const started = new Date(startedAt).getTime();
  const ended = endedAt ? new Date(endedAt).getTime() : Date.now();
  if (Number.isNaN(started) || Number.isNaN(ended)) {
    return "-";
  }
  const seconds = Math.max(0, Math.floor((ended - started) / 1000));
  if (seconds < 60) {
    return `${seconds} 秒`;
  }
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) {
    return `${minutes} 分 ${seconds % 60} 秒`;
  }
  const hours = Math.floor(minutes / 60);
  return `${hours} 小时 ${minutes % 60} 分`;
}

export function LogsPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const [items, setItems] = useState<JobRun[]>([]);
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const [pageSize, setPageSize] = useState(20);
  const [loading, setLoading] = useState(true);
  const [selectedRun, setSelectedRun] = useState<JobRun | null>(null);
  const [detailsOpen, setDetailsOpen] = useState(false);

  const [jobIdInput, setJobIdInput] = useState(searchParams.get("job_id") ?? "");
  const [typeKeyInput, setTypeKeyInput] = useState(searchParams.get("type_key") ?? "");
  const [outcomeInput, setOutcomeInput] = useState(searchParams.get("outcome") ?? "all");

  const jobIdFilter = searchParams.get("job_id") ?? "";
  const typeKeyFilter = searchParams.get("type_key") ?? "";
  const outcomeFilter = searchParams.get("outcome") ?? "";

  useEffect(() => {
    setJobIdInput(jobIdFilter);
    setTypeKeyInput(typeKeyFilter);
    setOutcomeInput(outcomeFilter || "all");
  }, [jobIdFilter, outcomeFilter, typeKeyFilter]);

  const load = useCallback(async (options?: { silent?: boolean }) => {
    const parsedJobID = Number(jobIdFilter);

    try {
      if (!options?.silent) {
        setLoading(true);
      }
      const response = await api.listJobRuns({
        limit: pageSize,
        offset,
        job_id: jobIdFilter && Number.isFinite(parsedJobID) ? parsedJobID : undefined,
        type_key: typeKeyFilter || undefined,
        outcome: outcomeFilter === "success" || outcomeFilter === "failed" ? outcomeFilter : undefined
      });
      setItems(response.items);
      setTotal(response.total);
    } catch (error) {
      if (!options?.silent) {
        toast.error(extractErrorMessage(error));
      }
    } finally {
      if (!options?.silent) {
        setLoading(false);
      }
    }
  }, [jobIdFilter, offset, outcomeFilter, pageSize, typeKeyFilter]);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    const timer = window.setInterval(() => {
      if (document.visibilityState !== "visible") {
        return;
      }
      void load({ silent: true });
    }, 5000);

    return () => window.clearInterval(timer);
  }, [load]);

  const handleApplyFilters = () => {
    if (jobIdInput && !/^\d+$/.test(jobIdInput.trim())) {
      toast.error("job_id 必须是数字");
      return;
    }

    setOffset(0);
    const next = new URLSearchParams();
    if (jobIdInput.trim()) {
      next.set("job_id", jobIdInput.trim());
    }
    if (typeKeyInput.trim()) {
      next.set("type_key", typeKeyInput.trim());
    }
    if (outcomeInput !== "all") {
      next.set("outcome", outcomeInput);
    }
    setSearchParams(next);
  };

  const handleLimitChange = (newLimit: number) => {
    setOffset(0);
    setPageSize(newLimit);
  };

  const metrics = useMemo(() => {
    const success = items.filter((item) => item.status === "success").length;
    const failed = items.filter((item) => item.status === "failed").length;
    const running = items.filter((item) => item.status === "running").length;
    const workers = new Set(items.map((item) => item.worker_id)).size;
    return [
      { label: "当前筛选总数", value: total, hint: "服务端分页后的总记录数" },
      { label: "本页运行中", value: running, hint: "当前页仍在执行的 run" },
      { label: "本页成功", value: success, hint: "当前页成功 run 数量" },
      { label: "本页失败", value: failed, hint: "当前页失败 run 数量" },
      { label: "活跃节点", value: workers, hint: "当前页涉及的 worker 数量" }
    ];
  }, [items, total]);

  const openRunDetails = (run: JobRun) => {
    setSelectedRun(run);
    setDetailsOpen(true);
  };

  useEffect(() => {
    if (!selectedRun) {
      return;
    }
    const next = items.find((item) => item.id === selectedRun.id);
    if (next) {
      setSelectedRun(next);
    }
  }, [items, selectedRun]);

  return (
    <div className="space-y-4">
      <PageHeader title="日志" description="集中查看任务 run 历史、执行结果和原始日志。">
        <div className="flex flex-wrap items-center gap-2">
          <Badge variant="outline">自动刷新 5 秒</Badge>
          <Button variant="outline" size="icon" onClick={() => void load()} disabled={loading}>
            <RefreshCw className={`h-4 w-4 ${loading ? "animate-spin" : ""}`} />
          </Button>
        </div>
      </PageHeader>

      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        {metrics.map((metric) => (
          <Card key={metric.label}>
            <CardHeader className="pb-3">
              <CardDescription>{metric.label}</CardDescription>
              <CardTitle>{metric.value}</CardTitle>
            </CardHeader>
            <CardContent className="text-xs text-muted-foreground">{metric.hint}</CardContent>
          </Card>
        ))}
      </div>

      <Card>
        <CardHeader>
          <CardTitle>筛选条件</CardTitle>
          <CardDescription>支持按任务、类型和执行结果快速定位日志。</CardDescription>
        </CardHeader>
        <CardContent className="grid gap-3 md:grid-cols-[180px_minmax(0,1fr)_180px_auto]">
          <Input
            value={jobIdInput}
            onChange={(event) => setJobIdInput(event.target.value)}
            placeholder="job_id"
          />
          <Input
            value={typeKeyInput}
            onChange={(event) => setTypeKeyInput(event.target.value)}
            placeholder="type_key"
          />
          <Select value={outcomeInput} onValueChange={setOutcomeInput}>
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部结果</SelectItem>
              <SelectItem value="success">仅成功</SelectItem>
              <SelectItem value="failed">仅失败</SelectItem>
            </SelectContent>
          </Select>
          <Button onClick={handleApplyFilters}>
            <Search className="mr-2 h-4 w-4" />
            应用筛选
          </Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>执行日志列表</CardTitle>
          <CardDescription>
            共 {total} 条
            {jobIdFilter ? `，当前限定 job #${jobIdFilter}` : ""}
            {typeKeyFilter ? `，类型 ${typeKeyFilter}` : ""}
          </CardDescription>
        </CardHeader>
        <CardContent>
          {items.length === 0 && !loading ? (
            <div className="rounded-lg border border-dashed border-border/80 bg-muted/25 px-4 py-8 text-center text-sm text-muted-foreground">
              当前筛选条件下暂无日志
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Run</TableHead>
                  <TableHead>Job</TableHead>
                  <TableHead>类型 / 动作</TableHead>
                  <TableHead>结果</TableHead>
                  <TableHead>账号</TableHead>
                  <TableHead>耗时</TableHead>
                  <TableHead>开始时间</TableHead>
                  <TableHead>Worker</TableHead>
                  <TableHead className="text-right">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {items.map((item) => (
                  <TableRow key={item.id}>
                    <TableCell className="font-mono text-xs">{compactId(item.id)}</TableCell>
                    <TableCell className="font-mono text-xs">{compactId(item.job_id)}</TableCell>
                    <TableCell>
                      <div className="space-y-1">
                        <div>{item.job_type_key}</div>
                        <div className="text-xs text-muted-foreground">{item.job_action_key}</div>
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant={runStatusMap[item.status]?.variant ?? "outline"}>
                        {runStatusMap[item.status]?.label ?? item.status}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-xs">{item.account_id ?? "-"}</TableCell>
                    <TableCell className="text-xs text-muted-foreground">{formatDuration(item.started_at, item.ended_at)}</TableCell>
                    <TableCell className="text-xs text-muted-foreground">{formatDateTime(item.started_at)}</TableCell>
                    <TableCell className="font-mono text-xs text-muted-foreground">{item.worker_id}</TableCell>
                    <TableCell className="text-right">
                      <Button size="icon" variant="ghost" onClick={() => openRunDetails(item)} title="查看日志详情">
                        <Eye className="h-4 w-4" />
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
          <Pagination total={total} limit={pageSize} offset={offset} onPageChange={setOffset} onLimitChange={handleLimitChange} />
        </CardContent>
      </Card>

      <JobRunDetails open={detailsOpen} run={selectedRun} onOpenChange={setDetailsOpen} />
    </div>
  );
}
