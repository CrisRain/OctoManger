import { useCallback, useEffect, useMemo, useState } from "react";
import { CircleStop, Clock3, Eye, Plus, RefreshCw, RotateCcw, ScrollText, Trash2 } from "lucide-react";
import { Link } from "react-router-dom";
import { toast } from "sonner";
import { PageHeader } from "@/components/page-header";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Pagination } from "@/components/ui/pagination";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { api, extractErrorMessage } from "@/lib/api";
import { compactId, formatDateTime } from "@/lib/format";
import type { Job, JobSummary } from "@/types";
import { JobCreate } from "./jobs/components/job-create";
import { JobDetails } from "./jobs/components/job-details";

const statusMap: Record<number, { label: string; variant: "default" | "secondary" | "destructive" | "outline" }> = {
  0: { label: "待执行", variant: "secondary" },
  1: { label: "执行中", variant: "default" },
  2: { label: "已成功", variant: "outline" },
  3: { label: "已失败", variant: "destructive" },
  4: { label: "已取消", variant: "secondary" }
};

function formatElapsed(raw: string): string {
  const timestamp = new Date(raw).getTime();
  if (Number.isNaN(timestamp)) {
    return "-";
  }
  const diff = Math.max(0, Date.now() - timestamp);
  const seconds = Math.floor(diff / 1000);
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

function metricTone(value: number, inverse = false) {
  if (value === 0) {
    return "text-foreground";
  }
  if (inverse) {
    return "text-emerald-600";
  }
  return "text-amber-600";
}

export function JobsPage() {
  const [items, setItems] = useState<Job[]>([]);
  const [summary, setSummary] = useState<JobSummary | null>(null);
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const [pageSize, setPageSize] = useState(10);
  const [loading, setLoading] = useState(true);
  const [lastRefreshedAt, setLastRefreshedAt] = useState<string>("");

  const [createOpen, setCreateOpen] = useState(false);
  const [detailsOpen, setDetailsOpen] = useState(false);
  const [selectedJobId, setSelectedJobId] = useState<number | null>(null);

  const load = useCallback(async (options?: { silent?: boolean }) => {
    const silent = options?.silent ?? false;

    try {
      if (!silent) {
        setLoading(true);
      }
      const [jobs, nextSummary] = await Promise.all([
        api.listJobs({ limit: pageSize, offset }),
        api.getJobSummary()
      ]);
      setItems(jobs.items);
      setTotal(jobs.total);
      setSummary(nextSummary);
      setLastRefreshedAt(new Date().toISOString());
    } catch (error) {
      if (!silent) {
        toast.error(extractErrorMessage(error));
      }
    } finally {
      if (!silent) {
        setLoading(false);
      }
    }
  }, [offset, pageSize]);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    const timer = window.setInterval(() => {
      if (document.visibilityState !== "visible") {
        return;
      }
      void load({ silent: true });
    }, 4000);

    return () => window.clearInterval(timer);
  }, [load]);

  const handleLimitChange = (newLimit: number) => {
    setOffset(0);
    setPageSize(newLimit);
  };

  const cancelJob = async (id: number) => {
    try {
      await api.cancelJob(id);
      toast.success("任务已取消");
      await load();
    } catch (error) {
      toast.error(extractErrorMessage(error));
    }
  };

  const retryJob = async (id: number) => {
    try {
      await api.retryJob(id);
      toast.success("已创建重试任务");
      await load();
    } catch (error) {
      toast.error(extractErrorMessage(error));
    }
  };

  const deleteJob = async (id: number) => {
    if (!confirm("确定要删除此任务记录吗？")) return;
    try {
      await api.deleteJob(id);
      toast.success("任务已删除");
      await load();
    } catch (error) {
      toast.error(extractErrorMessage(error));
    }
  };

  const handleViewDetails = (id: number) => {
    setSelectedJobId(id);
    setDetailsOpen(true);
  };

  const activeJobs = summary?.active ?? 0;
  const queueMetrics = useMemo(() => ([
    { label: "运行中", value: summary?.running ?? 0, hint: "Worker 正在处理的任务" },
    { label: "排队中", value: summary?.queued ?? 0, hint: "等待分发或执行" },
    { label: "失败", value: summary?.failed ?? 0, hint: "需要重试或排查" },
    { label: "已完成", value: summary?.done ?? 0, hint: "累计成功执行" }
  ]), [summary]);

  return (
    <div className="space-y-4">
      <PageHeader title="任务" description="实时观察队列积压、运行中任务和执行日志。">
        <div className="flex flex-wrap items-center gap-2">
          <Badge variant={activeJobs > 0 ? "default" : "outline"}>
            {activeJobs > 0 ? `实时轮询中 · ${activeJobs} 个活跃任务` : "实时轮询中 · 队列空闲"}
          </Badge>
          <Button variant="outline" size="icon" onClick={() => void load()} disabled={loading}>
            <RefreshCw className={`h-4 w-4 ${loading ? "animate-spin" : ""}`} />
          </Button>
          <Button asChild variant="outline">
            <Link to="/logs">
              <ScrollText className="mr-2 h-4 w-4" />
              查看日志页
            </Link>
          </Button>
          <Button onClick={() => setCreateOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            创建任务
          </Button>
        </div>
      </PageHeader>

      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        {queueMetrics.map((metric) => (
          <Card key={metric.label}>
            <CardHeader className="pb-3">
              <CardDescription>{metric.label}</CardDescription>
              <CardTitle className={metric.label === "已完成" ? metricTone(metric.value, true) : metricTone(metric.value)}>
                {metric.value}
              </CardTitle>
            </CardHeader>
            <CardContent className="text-xs text-muted-foreground">
              {metric.hint}
            </CardContent>
          </Card>
        ))}
      </div>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <div className="space-y-1">
            <CardTitle>任务队列</CardTitle>
            <CardDescription>
              共 {total} 条
              {lastRefreshedAt ? `，上次刷新 ${formatDateTime(lastRefreshedAt)}` : ""}
            </CardDescription>
          </div>
          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            <Clock3 className="h-4 w-4" />
            <span>自动刷新间隔 4 秒</span>
          </div>
        </CardHeader>
        <CardContent>
          {items.length === 0 && !loading ? (
            <div className="rounded-lg border border-dashed border-border/80 bg-muted/25 px-4 py-8 text-center text-sm text-muted-foreground">
              暂无任务
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>ID</TableHead>
                  <TableHead>类型</TableHead>
                  <TableHead>动作</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead>运行态</TableHead>
                  <TableHead>最近 Run</TableHead>
                  <TableHead>更新时间</TableHead>
                  <TableHead className="text-right">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {items.map((item) => {
                  const canCancel = item.status === 0 || item.status === 1;
                  const canRetry = item.status === 3 || item.status === 4;
                  const isRunning = item.status === 1;
                  const isQueued = item.status === 0;
                  const liveRun = item.last_run;

                  return (
                    <TableRow key={item.id}>
                      <TableCell className="font-medium font-mono text-xs">{compactId(item.id)}</TableCell>
                      <TableCell>{item.type_key}</TableCell>
                      <TableCell>{item.action_key}</TableCell>
                      <TableCell>
                        <div className="space-y-1">
                          <Badge variant={statusMap[item.status]?.variant ?? "outline"}>
                            {statusMap[item.status]?.label ?? `#${item.status}`}
                          </Badge>
                          {(isRunning || isQueued) ? (
                            <div className="text-xs text-muted-foreground">
                              {isRunning ? "状态由 worker 实时回写" : "等待 worker 接单"}
                            </div>
                          ) : null}
                        </div>
                      </TableCell>
                      <TableCell className="text-xs text-muted-foreground">
                        {isRunning ? `已运行 ${formatElapsed(item.updated_at)}` : isQueued ? `已排队 ${formatElapsed(item.created_at)}` : "-"}
                      </TableCell>
                      <TableCell className="text-xs text-muted-foreground">
                        {liveRun ? (
                          <div className="space-y-1">
                            <Badge variant={liveRun.status === "running" ? "default" : liveRun.status === "failed" ? "destructive" : "outline"}>
                              {liveRun.status === "running" ? "实时运行中" : liveRun.status === "failed" ? "最近失败" : "最近成功"}
                            </Badge>
                            <div>日志 {liveRun.logs?.length ?? 0} 条</div>
                          </div>
                        ) : (
                          "-"
                        )}
                      </TableCell>
                      <TableCell className="text-muted-foreground text-xs">{formatDateTime(item.updated_at)}</TableCell>
                      <TableCell className="text-right">
                        <div className="flex items-center justify-end gap-2">
                          <Button
                            size="icon"
                            variant="ghost"
                            onClick={() => handleViewDetails(item.id)}
                            title="查看详情"
                          >
                            <Eye className="h-4 w-4" />
                          </Button>
                          <Button asChild size="icon" variant="ghost" title="查看日志">
                            <Link to={`/logs?job_id=${item.id}`}>
                              <ScrollText className="h-4 w-4 text-sky-600" />
                            </Link>
                          </Button>
                          {canCancel && (
                            <Button
                              size="icon"
                              variant="ghost"
                              onClick={() => void cancelJob(item.id)}
                              title="取消任务"
                            >
                              <CircleStop className="h-4 w-4 text-orange-500" />
                            </Button>
                          )}
                          {canRetry && (
                            <Button
                              size="icon"
                              variant="ghost"
                              onClick={() => void retryJob(item.id)}
                              title="重试任务"
                            >
                              <RotateCcw className="h-4 w-4 text-blue-500" />
                            </Button>
                          )}
                          <Button
                            size="icon"
                            variant="ghost"
                            onClick={() => void deleteJob(item.id)}
                            title="删除任务"
                          >
                            <Trash2 className="h-4 w-4 text-destructive" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          )}
          <Pagination total={total} limit={pageSize} offset={offset} onPageChange={setOffset} onLimitChange={handleLimitChange} />
        </CardContent>
      </Card>

      <JobCreate
        open={createOpen}
        onOpenChange={setCreateOpen}
        onSuccess={load}
      />

      <JobDetails
        jobId={selectedJobId}
        open={detailsOpen}
        onOpenChange={setDetailsOpen}
      />
    </div>
  );
}
