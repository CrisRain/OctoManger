import { JsonView } from "@/components/json-view";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { compactId, formatDateTime } from "@/lib/format";
import type { JobRun } from "@/types";

interface JobRunDetailsProps {
  open: boolean;
  run: JobRun | null;
  onOpenChange: (open: boolean) => void;
}

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

export function JobRunDetails({ open, run, onOpenChange }: JobRunDetailsProps) {
  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="flex w-[720px] flex-col p-0 sm:max-w-[720px]">
        <SheetHeader className="border-b p-6">
          <SheetTitle>执行日志详情</SheetTitle>
          <SheetDescription>
            {run ? `run #${compactId(run.id)} · job #${compactId(run.job_id)}` : "查看任务执行日志、结果和错误信息。"}
          </SheetDescription>
        </SheetHeader>

        <div className="flex-1 overflow-hidden p-6">
          <ScrollArea className="h-full pr-4">
            {run ? (
              <div className="space-y-6">
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <h4 className="mb-1 text-sm font-medium text-muted-foreground">运行状态</h4>
                    <Badge variant={runStatusMap[run.status]?.variant ?? "outline"}>
                      {runStatusMap[run.status]?.label ?? run.status}
                    </Badge>
                  </div>
                  <div>
                    <h4 className="mb-1 text-sm font-medium text-muted-foreground">任务</h4>
                    <div className="flex items-center gap-2">
                      <Badge variant="outline">{run.job_type_key}</Badge>
                      <Badge variant="outline">{run.job_action_key}</Badge>
                    </div>
                  </div>
                  <div>
                    <h4 className="mb-1 text-sm font-medium text-muted-foreground">Run ID</h4>
                    <p className="font-mono text-sm">{run.id}</p>
                  </div>
                  <div>
                    <h4 className="mb-1 text-sm font-medium text-muted-foreground">Job ID</h4>
                    <p className="font-mono text-sm">{run.job_id}</p>
                  </div>
                  <div>
                    <h4 className="mb-1 text-sm font-medium text-muted-foreground">Account ID</h4>
                    <p className="text-sm">{run.account_id ?? "-"}</p>
                  </div>
                  <div>
                    <h4 className="mb-1 text-sm font-medium text-muted-foreground">Worker</h4>
                    <p className="font-mono text-xs">{run.worker_id}</p>
                  </div>
                  <div>
                    <h4 className="mb-1 text-sm font-medium text-muted-foreground">开始时间</h4>
                    <p className="text-sm">{formatDateTime(run.started_at)}</p>
                  </div>
                  <div>
                    <h4 className="mb-1 text-sm font-medium text-muted-foreground">结束时间</h4>
                    <p className="text-sm">{run.ended_at ? formatDateTime(run.ended_at) : "-"}</p>
                  </div>
                  <div>
                    <h4 className="mb-1 text-sm font-medium text-muted-foreground">运行时长</h4>
                    <p className="text-sm">{formatDuration(run.started_at, run.ended_at)}</p>
                  </div>
                  <div>
                    <h4 className="mb-1 text-sm font-medium text-muted-foreground">尝试次数</h4>
                    <p className="text-sm">{run.attempt}</p>
                  </div>
                  <div>
                    <h4 className="mb-1 text-sm font-medium text-muted-foreground">错误码</h4>
                    <p className="text-sm">{run.error_code || "-"}</p>
                  </div>
                  <div>
                    <h4 className="mb-1 text-sm font-medium text-muted-foreground">错误信息</h4>
                    <p className="text-sm">{run.error_message || "-"}</p>
                  </div>
                </div>

                <div>
                  <h4 className="mb-2 text-sm font-medium text-muted-foreground">执行结果</h4>
                  <div className="rounded-md border bg-muted/50 p-2">
                    <JsonView value={run.result ?? {}} />
                  </div>
                </div>

                <div>
                  <h4 className="mb-2 text-sm font-medium text-muted-foreground">日志</h4>
                  <div className="rounded-md border bg-muted/50 p-3">
                    {run.logs?.length ? (
                      <>
                        <div className="mb-2 text-xs text-muted-foreground">共 {run.logs.length} 条</div>
                        <ScrollArea className="h-72">
                          <div className="space-y-1 font-mono text-xs">
                            {run.logs.map((line, index) => (
                              <div key={`${run.id}-${index}`} className="break-all">
                                {line}
                              </div>
                            ))}
                          </div>
                        </ScrollArea>
                      </>
                    ) : (
                      <div className="text-xs text-muted-foreground">暂无日志</div>
                    )}
                  </div>
                </div>
              </div>
            ) : (
              <div className="text-sm text-muted-foreground">未选择日志记录。</div>
            )}
          </ScrollArea>
        </div>
      </SheetContent>
    </Sheet>
  );
}
