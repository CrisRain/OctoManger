import { useQuery } from "@tanstack/react-query";
import { Eye } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { api } from "@/lib/api";
import { compactId, formatDateTime } from "@/lib/format";

const jobStatusMap: Record<
  number,
  { label: string; variant: "default" | "secondary" | "destructive" | "outline" }
> = {
  0: { label: "待执行", variant: "secondary" },
  1: { label: "执行中", variant: "default" },
  2: { label: "已成功", variant: "outline" },
  3: { label: "已失败", variant: "destructive" },
  4: { label: "已取消", variant: "secondary" },
};

interface RecentJobsProps {
  onViewJob: (id: number) => void;
}

export function RecentJobs({ onViewJob }: RecentJobsProps) {
  const { data, isLoading } = useQuery({
    queryKey: ["jobs", "list", { limit: 10 }],
    queryFn: () => api.listJobs({ limit: 10 }),
    refetchInterval: 30000,
  });

  const recentJobs = [...(data?.items ?? [])].sort(
    (a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime(),
  );

  return (
    <Card>
      <CardHeader>
        <CardTitle>最近任务</CardTitle>
        <CardDescription>按更新时间倒序显示最近 10 条任务，便于快速定位失败和积压。</CardDescription>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="space-y-2">
            <Skeleton className="h-10" />
            <Skeleton className="h-10" />
            <Skeleton className="h-10" />
          </div>
        ) : recentJobs.length === 0 ? (
          <div className="rounded-lg border border-dashed border-border/80 bg-muted/25 px-4 py-8 text-center text-sm text-muted-foreground">
            暂无任务记录
          </div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>任务 ID</TableHead>
                <TableHead>类型</TableHead>
                <TableHead>动作</TableHead>
                <TableHead>状态</TableHead>
                <TableHead>最近 Run</TableHead>
                <TableHead>更新时间</TableHead>
                <TableHead className="text-right">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {recentJobs.map((job) => (
                <TableRow key={job.id}>
                  <TableCell className="font-mono text-xs font-medium">{compactId(job.id)}</TableCell>
                  <TableCell>{job.type_key}</TableCell>
                  <TableCell>{job.action_key}</TableCell>
                  <TableCell>
                    <Badge variant={jobStatusMap[job.status]?.variant ?? "outline"}>
                      {jobStatusMap[job.status]?.label ?? `#${job.status}`}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {job.last_run ? (
                      <Badge
                        variant={
                          job.last_run.status === "running"
                            ? "default"
                            : job.last_run.status === "failed"
                              ? "destructive"
                              : "outline"
                        }
                      >
                        {job.last_run.status === "running"
                          ? "运行中"
                          : job.last_run.status === "failed"
                            ? "失败"
                            : "成功"}
                      </Badge>
                    ) : (
                      "-"
                    )}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatDateTime(job.updated_at)}
                  </TableCell>
                  <TableCell className="text-right">
                    <Button
                      size="icon"
                      variant="ghost"
                      className="h-8 w-8"
                      onClick={() => onViewJob(job.id)}
                    >
                      <Eye className="h-4 w-4" />
                      <span className="sr-only">查看任务详情</span>
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  );
}
