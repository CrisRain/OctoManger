import { useQuery } from "@tanstack/react-query";
import { ExternalLink, RefreshCw } from "lucide-react";
import { Link } from "react-router-dom";
import { JsonView } from "@/components/json-view";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { api } from "@/lib/api";
import { compactId, formatDateTime } from "@/lib/format";
import type { JobRun } from "@/types";

interface JobDetailsProps {
  jobId: number | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const statusMap: Record<
  number,
  { label: string; variant: "default" | "secondary" | "destructive" | "outline" }
> = {
  0: { label: "待执行", variant: "secondary" },
  1: { label: "执行中", variant: "default" },
  2: { label: "已成功", variant: "outline" },
  3: { label: "已失败", variant: "destructive" },
  4: { label: "已取消", variant: "secondary" },
};

const runStatusMap: Record<
  JobRun["status"],
  { label: string; variant: "default" | "secondary" | "destructive" | "outline" }
> = {
  running: { label: "运行中", variant: "default" },
  success: { label: "成功", variant: "outline" },
  failed: { label: "失败", variant: "destructive" },
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

export function JobDetails({ jobId, open, onOpenChange }: JobDetailsProps) {
  const jobQuery = useQuery({
    queryKey: ["jobs", jobId],
    queryFn: () => api.getJob(jobId!),
    enabled: !!jobId && open,
    refetchInterval: 4000,
  });

  const runsQuery = useQuery({
    queryKey: ["jobs", jobId, "runs", { limit: 5 }],
    queryFn: () => api.listJobRuns({ job_id: jobId!, limit: 5, offset: 0 }),
    enabled: !!jobId && open,
    refetchInterval: 4000,
  });

  const job = jobQuery.data;
  const history = runsQuery.data?.items ?? [];
  const historyTotal = runsQuery.data?.total ?? 0;
  const isLoading = jobQuery.isLoading || runsQuery.isLoading;

  const isLive =
    job?.status === 0 || job?.status === 1 || job?.last_run?.status === "running";

  const handleRefresh = () => {
    void jobQuery.refetch();
    void runsQuery.refetch();
  };

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="flex w-[720px] flex-col p-0 sm:max-w-[720px]">
        <SheetHeader className="border-b p-6">
          <div className="flex items-start justify-between gap-3">
            <div className="space-y-1">
              <SheetTitle>任务详情</SheetTitle>
              <SheetDescription>
                查看任务 {jobId} 的实时状态、最近执行和日志入口。
              </SheetDescription>
            </div>
            <div className="flex items-center gap-2">
              {isLive ? (
                <Badge variant="default">实时刷新中</Badge>
              ) : (
                <Badge variant="outline">静态结果</Badge>
              )}
              <Button
                variant="outline"
                size="icon"
                onClick={handleRefresh}
                disabled={isLoading || !jobId}
              >
                <RefreshCw className={`h-4 w-4 ${isLoading ? "animate-spin" : ""}`} />
              </Button>
              {jobId ? (
                <Button asChild variant="outline">
                  <Link
                    to={`/logs?job_id=${jobId}`}
                    onClick={() => onOpenChange(false)}
                  >
                    <ExternalLink className="mr-2 h-4 w-4" />
                    完整日志
                  </Link>
                </Button>
              ) : null}
            </div>
          </div>
        </SheetHeader>

        <div className="flex-1 overflow-hidden p-6">
          <ScrollArea className="h-full pr-4">
            {isLoading && !job ? (
              <div className="text-sm text-muted-foreground">加载中...</div>
            ) : job ? (
              <div className="space-y-6">
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <h4 className="mb-1 text-sm font-medium text-muted-foreground">ID</h4>
                    <p className="font-mono text-sm">{job.id}</p>
                  </div>
                  <div>
                    <h4 className="mb-1 text-sm font-medium text-muted-foreground">
                      类型 / 动作
                    </h4>
                    <div className="flex items-center gap-2">
                      <Badge variant="outline">{job.type_key}</Badge>
                      <span className="text-muted-foreground">/</span>
                      <Badge variant="outline">{job.action_key}</Badge>
                    </div>
                  </div>
                  <div>
                    <h4 className="mb-1 text-sm font-medium text-muted-foreground">
                      状态
                    </h4>
                    <Badge variant={statusMap[job.status]?.variant ?? "outline"}>
                      {statusMap[job.status]?.label ?? `#${job.status}`}
                    </Badge>
                  </div>
                  <div>
                    <h4 className="mb-1 text-sm font-medium text-muted-foreground">
                      创建时间
                    </h4>
                    <p className="text-sm">{formatDateTime(job.created_at)}</p>
                  </div>
                  <div>
                    <h4 className="mb-1 text-sm font-medium text-muted-foreground">
                      更新时间
                    </h4>
                    <p className="text-sm">{formatDateTime(job.updated_at)}</p>
                  </div>
                  <div>
                    <h4 className="mb-1 text-sm font-medium text-muted-foreground">
                      执行历史
                    </h4>
                    <p className="text-sm">{historyTotal} 次 run</p>
                  </div>
                </div>

                <div>
                  <h4 className="mb-2 text-sm font-medium text-muted-foreground">
                    选择器
                  </h4>
                  <div className="rounded-md border bg-muted/50 p-2">
                    <JsonView value={job.selector} />
                  </div>
                </div>

                <div>
                  <h4 className="mb-2 text-sm font-medium text-muted-foreground">
                    参数
                  </h4>
                  <div className="rounded-md border bg-muted/50 p-2">
                    <JsonView value={job.params} />
                  </div>
                </div>

                {job.last_run ? (
                  <>
                    <div>
                      <h4 className="mb-2 text-sm font-medium text-muted-foreground">
                        最近一次执行
                      </h4>
                      <div className="grid grid-cols-2 gap-4 rounded-md border bg-muted/50 p-3 text-sm">
                        <div>
                          <h5 className="mb-1 text-xs font-medium text-muted-foreground">
                            执行状态
                          </h5>
                          <Badge
                            variant={
                              runStatusMap[job.last_run.status]?.variant ?? "outline"
                            }
                          >
                            {runStatusMap[job.last_run.status]?.label ??
                              job.last_run.status}
                          </Badge>
                        </div>
                        <div>
                          <h5 className="mb-1 text-xs font-medium text-muted-foreground">
                            工作节点
                          </h5>
                          <p className="font-mono text-xs">{job.last_run.worker_id}</p>
                        </div>
                        <div>
                          <h5 className="mb-1 text-xs font-medium text-muted-foreground">
                            尝试次数
                          </h5>
                          <p>{job.last_run.attempt}</p>
                        </div>
                        <div>
                          <h5 className="mb-1 text-xs font-medium text-muted-foreground">
                            运行时长
                          </h5>
                          <p>
                            {formatDuration(
                              job.last_run.started_at,
                              job.last_run.ended_at
                            )}
                          </p>
                        </div>
                        <div>
                          <h5 className="mb-1 text-xs font-medium text-muted-foreground">
                            开始时间
                          </h5>
                          <p>{formatDateTime(job.last_run.started_at)}</p>
                        </div>
                        <div>
                          <h5 className="mb-1 text-xs font-medium text-muted-foreground">
                            结束时间
                          </h5>
                          <p>
                            {job.last_run.ended_at
                              ? formatDateTime(job.last_run.ended_at)
                              : "-"}
                          </p>
                        </div>
                        <div>
                          <h5 className="mb-1 text-xs font-medium text-muted-foreground">
                            错误码
                          </h5>
                          <p>{job.last_run.error_code || "-"}</p>
                        </div>
                        <div>
                          <h5 className="mb-1 text-xs font-medium text-muted-foreground">
                            错误信息
                          </h5>
                          <p>{job.last_run.error_message || "-"}</p>
                        </div>
                      </div>
                    </div>

                    <div>
                      <h4 className="mb-2 text-sm font-medium text-muted-foreground">
                        最近执行结果
                      </h4>
                      <div className="rounded-md border bg-muted/50 p-2">
                        <JsonView value={job.last_run.result ?? {}} />
                      </div>
                    </div>

                    <div>
                      <h4 className="mb-2 text-sm font-medium text-muted-foreground">
                        最近执行日志
                      </h4>
                      <div className="rounded-md border bg-muted/50 p-3">
                        {job.last_run.logs?.length ? (
                          <>
                            <div className="mb-2 text-xs text-muted-foreground">
                              共 {job.last_run.logs.length} 条
                            </div>
                            <ScrollArea className="h-48">
                              <div className="space-y-1 font-mono text-xs">
                                {job.last_run.logs.map((line, index) => (
                                  <div
                                    key={`${job.last_run?.id}-${index}`}
                                    className="break-all"
                                  >
                                    {line}
                                  </div>
                                ))}
                              </div>
                            </ScrollArea>
                          </>
                        ) : (
                          <div className="text-xs text-muted-foreground">
                            暂无日志
                          </div>
                        )}
                      </div>
                    </div>
                  </>
                ) : null}

                <div>
                  <div className="mb-2 flex items-center justify-between gap-3">
                    <h4 className="text-sm font-medium text-muted-foreground">
                      最近 5 次执行
                    </h4>
                    {jobId ? (
                      <Link
                        to={`/logs?job_id=${jobId}`}
                        className="text-xs text-primary underline-offset-4 hover:underline"
                        onClick={() => onOpenChange(false)}
                      >
                        打开完整日志页
                      </Link>
                    ) : null}
                  </div>
                  <div className="space-y-2 rounded-md border bg-muted/30 p-3">
                    {history.length ? (
                      history.map((run) => (
                        <div
                          key={run.id}
                          className="rounded-md border bg-background p-3"
                        >
                          <div className="flex flex-wrap items-center justify-between gap-3">
                            <div className="space-y-1">
                              <div className="flex items-center gap-2">
                                <span className="font-mono text-xs">
                                  run #{compactId(run.id)}
                                </span>
                                <Badge
                                  variant={
                                    runStatusMap[run.status]?.variant ?? "outline"
                                  }
                                >
                                  {runStatusMap[run.status]?.label ?? run.status}
                                </Badge>
                              </div>
                              <div className="text-xs text-muted-foreground">
                                worker {run.worker_id} · 尝试 {run.attempt}
                                {run.account_id ? ` · 账号 ${run.account_id}` : ""}
                              </div>
                            </div>
                            <div className="text-right text-xs text-muted-foreground">
                              <div>{formatDateTime(run.started_at)}</div>
                              <div>
                                {formatDuration(run.started_at, run.ended_at)}
                              </div>
                            </div>
                          </div>
                        </div>
                      ))
                    ) : (
                      <div className="text-xs text-muted-foreground">
                        暂无 run 记录
                      </div>
                    )}
                  </div>
                </div>
              </div>
            ) : (
              <div className="text-sm text-muted-foreground">任务未找到。</div>
            )}
          </ScrollArea>
        </div>
      </SheetContent>
    </Sheet>
  );
}
