import { useMemo, useState } from "react";
import { ChevronDown, ChevronUp, Loader2 } from "lucide-react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { Textarea } from "@/components/ui/textarea";
import { api, extractErrorMessage } from "@/lib/api";
import type { BatchImportGraphEmailResult } from "@/types";
import { splitScopes, type OutlookOAuthConfig } from "./outlook-config";

interface EmailAccountImportProps {
  config: OutlookOAuthConfig;
  onSuccess: () => void;
  onOpenSetup?: () => void;
}

const IMPORT_EXAMPLES = [
  {
    label: "只有邮箱和 refresh token",
    value: "user@outlook.com----refresh_token",
  },
  {
    label: "额外指定 client_id",
    value: "user@outlook.com----client_id----refresh_token",
  },
  {
    label: "附带密码和 client_id",
    value: "user@outlook.com----password----client_id----refresh_token",
  },
];

export function EmailAccountImport({ config, onSuccess, onOpenSetup }: EmailAccountImportProps) {
  const [loading, setLoading] = useState(false);
  const [showExamples, setShowExamples] = useState(false);
  const [text, setText] = useState("");
  const [inputError, setInputError] = useState<string | null>(null);
  const [result, setResult] = useState<BatchImportGraphEmailResult | null>(null);

  const lineCount = useMemo(() => text.split("\n").filter((line) => line.trim()).length, [text]);
  const helpfulDefaultsReady = Boolean(config.clientId.trim() && splitScopes(config.scope).length > 0);

  const handleImport = async () => {
    if (!text.trim()) {
      setInputError("请先粘贴至少一行账号数据。");
      toast.error("请先粘贴要导入的账号数据。");
      return;
    }

    setInputError(null);
    setLoading(true);
    setResult(null);

    try {
      const nextResult = await api.batchImportGraphEmailAccounts({
        content: text,
        default_client_id: config.clientId,
        tenant: config.tenant,
        scope: splitScopes(config.scope),
        mailbox: config.mailbox,
        graph_base_url: config.graphBaseURL,
        status: 0,
      });

      setResult(nextResult);

      if (nextResult.queued) {
        toast.success(
          `导入任务已提交：接受 ${nextResult.accepted} 条，跳过 ${nextResult.skipped} 条${
            nextResult.job_id ? `（job: ${nextResult.job_id}）` : ""
          }`,
        );
        onSuccess();
        return;
      }

      toast.error(`这次没有进入后台处理：接受 ${nextResult.accepted} 条，跳过 ${nextResult.skipped} 条。`);
    } catch (error) {
      toast.error(extractErrorMessage(error));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="grid gap-6">
      <Card className="overflow-hidden border-emerald-200/70 bg-gradient-to-br from-white via-white to-emerald-50/70 shadow-sm">
        <CardHeader className="gap-4 border-b border-emerald-100/80 bg-white/80 backdrop-blur">
          <div className="space-y-3">
            <Badge variant="outline" className="w-fit">
              适合已经拿到 refresh token 的情况
            </Badge>
            <div className="space-y-2">
              <CardTitle className="text-xl">批量导入已有 Outlook 账号</CardTitle>
              <CardDescription className="max-w-2xl leading-6">
                如果这些邮箱已经有可用的 refresh token，可以直接一行一条导入，不需要再手动走微软授权弹窗。
              </CardDescription>
            </div>
          </div>

          <div className="grid gap-3 md:grid-cols-3">
            <div className="rounded-2xl border border-border/80 bg-white/85 p-4">
              <p className="text-sm font-medium">1. 一行一个账号</p>
              <p className="mt-2 text-sm text-muted-foreground">把邮箱和 token 按支持格式粘贴进来即可。</p>
            </div>
            <div className="rounded-2xl border border-border/80 bg-white/85 p-4">
              <p className="text-sm font-medium">2. 系统自动补默认设置</p>
              <p className="mt-2 text-sm text-muted-foreground">会优先使用你保存的 client_id、tenant、scope 等默认值。</p>
            </div>
            <div className="rounded-2xl border border-border/80 bg-white/85 p-4">
              <p className="text-sm font-medium">3. 后台批量处理</p>
              <p className="mt-2 text-sm text-muted-foreground">导入任务会进入后台队列，处理完成后出现在邮箱列表里。</p>
            </div>
          </div>
        </CardHeader>

        <CardContent className="space-y-6 pt-6">
          <div
            className={`rounded-2xl border p-4 ${
              helpfulDefaultsReady ? "border-emerald-200 bg-emerald-50/70" : "border-amber-200 bg-amber-50/80"
            }`}
          >
            <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
              <div>
                <p className="text-sm font-medium">
                  {helpfulDefaultsReady ? "默认导入参数已就绪" : "推荐先补齐连接设置"}
                </p>
                <p className="mt-1 text-sm text-muted-foreground">
                  {helpfulDefaultsReady
                    ? "系统会自动带上已保存的 client_id、tenant、scope 和邮箱文件夹配置。"
                    : "如果先完成连接设置，导入时就不容易缺少 client_id 或 scope。"}
                </p>
              </div>
              {!helpfulDefaultsReady && onOpenSetup ? (
                <Button type="button" variant="outline" onClick={onOpenSetup}>
                  去完成连接设置
                </Button>
              ) : null}
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="email-import-text">账号数据（每行一个）</Label>
            <Textarea
              id="email-import-text"
              className="min-h-[240px] font-mono text-xs"
              placeholder={IMPORT_EXAMPLES.map((item) => item.value).join("\n")}
              value={text}
              onChange={(e) => {
                setText(e.target.value);
                setInputError(null);
              }}
              aria-invalid={Boolean(inputError)}
            />
            <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
              <p className={`text-xs ${inputError ? "text-destructive" : "text-muted-foreground"}`}>
                {inputError ?? `当前共识别到 ${lineCount} 行数据。`}
              </p>
              <Button type="button" variant="outline" size="sm" onClick={() => setShowExamples((prev) => !prev)}>
                {showExamples ? <ChevronUp className="mr-2 h-4 w-4" /> : <ChevronDown className="mr-2 h-4 w-4" />}
                {showExamples ? "收起格式说明" : "查看支持格式"}
              </Button>
            </div>
          </div>

          <div className="overflow-hidden rounded-2xl border border-border/80 bg-white/75">
            <button
              type="button"
              className="flex w-full items-center justify-between gap-3 px-4 py-4 text-left"
              onClick={() => setShowExamples((prev) => !prev)}
            >
              <div>
                <p className="text-sm font-medium">导入格式说明</p>
                <p className="mt-1 text-sm text-muted-foreground">
                  不确定时展开看例子。普通用户只需要照着例子一行一条粘贴即可。
                </p>
              </div>
              {showExamples ? <ChevronUp className="h-4 w-4 shrink-0" /> : <ChevronDown className="h-4 w-4 shrink-0" />}
            </button>

            {showExamples ? (
              <>
                <Separator />
                <div className="space-y-3 p-4">
                  {IMPORT_EXAMPLES.map((example) => (
                    <div key={example.label} className="rounded-2xl border border-border/80 bg-muted/20 p-4">
                      <p className="text-sm font-medium">{example.label}</p>
                      <code className="mt-2 block break-all rounded-lg bg-card px-3 py-2 text-xs">{example.value}</code>
                    </div>
                  ))}
                  <p className="text-xs text-muted-foreground">
                    系统会优先使用每一行里自带的 client_id。没有写时，才会回退到你在连接设置里保存的默认值。
                  </p>
                </div>
              </>
            ) : null}
          </div>

          <Button onClick={() => void handleImport()} disabled={loading || !text.trim()} className="w-full sm:w-auto">
            {loading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            开始批量导入
          </Button>
        </CardContent>
      </Card>

      {result ? (
        <Card>
          <CardHeader>
            <CardTitle>导入结果</CardTitle>
            <CardDescription>这里会显示本次导入接受了多少条、跳过了多少条，以及失败原因。</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="mb-4 grid gap-3 text-center sm:grid-cols-3">
              <div className="rounded-2xl bg-muted/50 p-4">
                <div className="text-2xl font-semibold">{result.total}</div>
                <div className="mt-1 text-xs text-muted-foreground">总行数</div>
              </div>
              <div className="rounded-2xl bg-emerald-50 p-4">
                <div className="text-2xl font-semibold text-emerald-600">{result.accepted}</div>
                <div className="mt-1 text-xs text-muted-foreground">已接受</div>
              </div>
              <div className="rounded-2xl bg-red-50 p-4">
                <div className="text-2xl font-semibold text-red-600">{result.skipped}</div>
                <div className="mt-1 text-xs text-muted-foreground">已跳过</div>
              </div>
            </div>

            {result.queued ? (
              <div className="mb-4 rounded-2xl border border-border/80 bg-muted/20 p-4 text-sm text-muted-foreground">
                导入任务已经进入后台队列
                {result.job_id ? `，job_id: ${result.job_id}` : ""}
                {result.task_id ? `，task_id: ${result.task_id}` : ""}。
              </div>
            ) : null}

            {result.failures.length > 0 ? (
              <div>
                <h4 className="mb-2 text-sm font-medium">失败原因</h4>
                <div className="max-h-[220px] space-y-1 overflow-y-auto rounded-2xl border bg-muted/20 p-3 font-mono text-xs">
                  {result.failures.map((failure) => (
                    <div key={`${failure.line}-${failure.address ?? ""}`} className="text-red-500">
                      [第 {failure.line} 行] {failure.address || "未知地址"}: {failure.error}
                    </div>
                  ))}
                </div>
              </div>
            ) : null}
          </CardContent>
        </Card>
      ) : null}
    </div>
  );
}
