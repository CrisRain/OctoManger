import { useMemo, useState } from "react";
import { ChevronDown, ChevronUp, Loader2 } from "lucide-react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { Switch } from "@/components/ui/switch";
import { api, extractErrorMessage } from "@/lib/api";
import type { BatchRegisterEmailResult } from "@/types";
import { isOutlookConfigReady, splitScopes, type OutlookOAuthConfig } from "./outlook-config";

interface EmailAccountBatchRegisterProps {
  config: OutlookOAuthConfig;
  onSuccess: () => void;
  onOpenSetup?: () => void;
}

const MONTH_OPTIONS = [
  { value: "January", label: "1 月" },
  { value: "February", label: "2 月" },
  { value: "March", label: "3 月" },
  { value: "April", label: "4 月" },
  { value: "May", label: "5 月" },
  { value: "June", label: "6 月" },
  { value: "July", label: "7 月" },
  { value: "August", label: "8 月" },
  { value: "September", label: "9 月" },
  { value: "October", label: "10 月" },
  { value: "November", label: "11 月" },
  { value: "December", label: "12 月" },
];

export function EmailAccountBatchRegister({
  config,
  onSuccess,
  onOpenSetup,
}: EmailAccountBatchRegisterProps) {
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<BatchRegisterEmailResult | null>(null);
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [fieldErrors, setFieldErrors] = useState<{
    count?: string;
    prefix?: string;
    password?: string;
  }>({});
  const [form, setForm] = useState({
    count: 1,
    prefix: "octo",
    domain: "outlook.com",
    password: "",
    startIndex: 1,
    status: "0",
    firstName: "Octo",
    lastName: "Manager",
    birthMonth: "January",
    birthDay: 1,
    birthYear: 1991,
    challengeTimeout: 300,
    pageTimeout: 90,
    oauthTimeout: 300,
    apiTimeout: 60,
    showBrowser: true,
  });

  const scopes = splitScopes(config.scope);
  const oauthReady = isOutlookConfigReady(config) && Boolean(config.clientId.trim() && scopes.length > 0);
  const maxBirthYear = new Date().getFullYear() - 1;
  const addressPreview = `${form.prefix || "prefix"}${form.startIndex}@${form.domain}`;
  const selectedMonth = MONTH_OPTIONS.find((item) => item.value === form.birthMonth)?.label ?? form.birthMonth;

  const summaryBadges = useMemo(
    () => [
      `${form.count} 个账号`,
      `起始地址 ${addressPreview}`,
      form.showBrowser ? "显示浏览器" : "无头模式",
    ],
    [addressPreview, form.count, form.showBrowser],
  );

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    const nextErrors: typeof fieldErrors = {};
    if (!Number.isFinite(form.count) || form.count < 1 || form.count > 20) {
      nextErrors.count = "一次建议创建 1 到 20 个账号。";
    }
    if (!form.prefix.trim()) {
      nextErrors.prefix = "请填写邮箱前缀，例如 octo。";
    }
    if (!form.password.trim()) {
      nextErrors.password = "请填写统一密码，系统会用它创建本批账号。";
    }

    setFieldErrors(nextErrors);
    if (Object.keys(nextErrors).length > 0) {
      toast.error("先把页面里的必填项补齐，再开始批量创建。");
      return;
    }

    if (!oauthReady) {
      toast.error("连接设置还没准备好，请先到“连接设置”里完成 Outlook 配置。");
      onOpenSetup?.();
      return;
    }

    try {
      setLoading(true);
      setResult(null);
      const res = await api.batchRegisterEmailAccounts({
        provider: "outlook",
        count: Number(form.count),
        prefix: form.prefix.trim(),
        domain: form.domain,
        start_index: Number(form.startIndex),
        status: Number(form.status),
        options: {
          password: form.password,
          first_name: form.firstName.trim() || "Octo",
          last_name: form.lastName.trim() || "Manager",
          birth_month: form.birthMonth,
          birth_day: Number(form.birthDay),
          birth_year: Number(form.birthYear),
          headless: !form.showBrowser,
          challenge_timeout_seconds: Number(form.challengeTimeout),
          page_timeout_seconds: Number(form.pageTimeout),
          oauth_timeout_seconds: Number(form.oauthTimeout),
          api_timeout_seconds: Number(form.apiTimeout),
          oauth: {
            client_id: config.clientId.trim(),
            tenant: config.tenant.trim() || "consumers",
            redirect_uri: config.redirectUri.trim(),
            scope: scopes,
            mailbox: config.mailbox.trim() || "INBOX",
            graph_base_url: config.graphBaseURL.trim() || "https://graph.microsoft.com/v1.0",
            ...(config.clientSecret.trim() ? { client_secret: config.clientSecret.trim() } : {}),
          },
        },
      });
      setResult(res);
      if (res.queued) {
        toast.success(
          `批量创建任务已提交${res.job_id ? `（job: ${res.job_id}）` : ""}${res.task_id ? `（task: ${res.task_id}）` : ""}`,
        );
      } else {
        toast.success(`本次创建完成：成功 ${res.created} 个，失败 ${res.failed} 个。`);
      }
      onSuccess();
    } catch (error) {
      toast.error(extractErrorMessage(error));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="grid gap-6">
      <Card className="overflow-hidden border-violet-200/70 bg-gradient-to-br from-white via-white to-slate-50 shadow-sm">
        <CardHeader className="gap-4 border-b border-border/70 bg-white/80 backdrop-blur">
          <div className="space-y-3">
            <Badge variant={oauthReady ? "outline" : "secondary"} className="w-fit">
              {oauthReady ? "批量创建条件已满足" : "先完成连接设置后再使用"}
            </Badge>
            <div className="space-y-2">
              <CardTitle className="text-xl">批量新建 Outlook 邮箱</CardTitle>
              <CardDescription className="max-w-2xl leading-6">
                适合你要一次连续创建多条全新邮箱时使用。系统会打开微软真实注册流程，并在成功后自动把账号和授权信息保存进来。
              </CardDescription>
            </div>
          </div>

          <div className="grid gap-3 md:grid-cols-3">
            <div className="rounded-2xl border border-border/80 bg-white/85 p-4">
              <p className="text-sm font-medium">1. 生成地址</p>
              <p className="mt-2 text-sm text-muted-foreground">例如从 {addressPreview} 开始，自动顺序创建。</p>
            </div>
            <div className="rounded-2xl border border-border/80 bg-white/85 p-4">
              <p className="text-sm font-medium">2. 按微软页面完成验证</p>
              <p className="mt-2 text-sm text-muted-foreground">推荐显示浏览器窗口，方便你手动通过人机验证。</p>
            </div>
            <div className="rounded-2xl border border-border/80 bg-white/85 p-4">
              <p className="text-sm font-medium">3. 自动保存结果</p>
              <p className="mt-2 text-sm text-muted-foreground">成功后会自动写入系统，不需要再手动补 token。</p>
            </div>
          </div>
        </CardHeader>

        <CardContent className="space-y-6 pt-6">
          <div
            className={`rounded-2xl border p-4 ${
              oauthReady ? "border-emerald-200 bg-emerald-50/70" : "border-amber-200 bg-amber-50/80"
            }`}
          >
            <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
              <div>
                <p className="text-sm font-medium">
                  {oauthReady ? "当前可以直接批量创建" : "当前还不能开始批量创建"}
                </p>
                <p className="mt-1 text-sm text-muted-foreground">
                  {oauthReady
                    ? `会自动使用你已保存的 Outlook 连接配置，当前共启用 ${scopes.length} 项授权权限。`
                    : "缺少可用的 Outlook 连接配置。先保存应用 ID 和回调地址，再回来操作。"}
                </p>
              </div>
              {!oauthReady && onOpenSetup ? (
                <Button type="button" variant="outline" onClick={onOpenSetup}>
                  去完成连接设置
                </Button>
              ) : null}
            </div>
          </div>

          <form onSubmit={handleSubmit} className="space-y-6">
            <div className="grid gap-4 md:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="batch-count">这次要创建多少个</Label>
                <Input
                  id="batch-count"
                  type="number"
                  min={1}
                  max={20}
                  value={form.count}
                  onChange={(e) => {
                    setForm((prev) => ({ ...prev, count: Number(e.target.value) }));
                    setFieldErrors((prev) => ({ ...prev, count: undefined }));
                  }}
                  required
                  aria-invalid={Boolean(fieldErrors.count)}
                />
                <p className={`text-xs ${fieldErrors.count ? "text-destructive" : "text-muted-foreground"}`}>
                  {fieldErrors.count ?? "建议先从 1 到 3 个开始试跑，确认流程没问题后再放大数量。"}
                </p>
              </div>

              <div className="space-y-2">
                <Label htmlFor="batch-password">统一密码</Label>
                <Input
                  id="batch-password"
                  type="password"
                  value={form.password}
                  onChange={(e) => {
                    setForm((prev) => ({ ...prev, password: e.target.value }));
                    setFieldErrors((prev) => ({ ...prev, password: undefined }));
                  }}
                  placeholder="例如 Aa!Batch2026"
                  required
                  aria-invalid={Boolean(fieldErrors.password)}
                />
                <p className={`text-xs ${fieldErrors.password ? "text-destructive" : "text-muted-foreground"}`}>
                  {fieldErrors.password ?? "这批邮箱会共用同一个密码，请使用符合微软要求的强密码。"}
                </p>
              </div>

              <div className="space-y-2">
                <Label htmlFor="batch-prefix">邮箱前缀</Label>
                <Input
                  id="batch-prefix"
                  value={form.prefix}
                  onChange={(e) => {
                    setForm((prev) => ({ ...prev, prefix: e.target.value }));
                    setFieldErrors((prev) => ({ ...prev, prefix: undefined }));
                  }}
                  placeholder="例如 octo"
                  aria-invalid={Boolean(fieldErrors.prefix)}
                />
                <p className={`text-xs ${fieldErrors.prefix ? "text-destructive" : "text-muted-foreground"}`}>
                  {fieldErrors.prefix ?? "系统会自动在后面补序号，例如 octo1、octo2。"}
                </p>
              </div>

              <div className="space-y-2">
                <Label htmlFor="batch-domain">邮箱域名</Label>
                <Select value={form.domain} onValueChange={(value) => setForm((prev) => ({ ...prev, domain: value }))}>
                  <SelectTrigger id="batch-domain">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="outlook.com">outlook.com</SelectItem>
                    <SelectItem value="hotmail.com">hotmail.com</SelectItem>
                    <SelectItem value="live.com">live.com</SelectItem>
                    <SelectItem value="msn.com">msn.com</SelectItem>
                  </SelectContent>
                </Select>
                <p className="text-xs text-muted-foreground">示例地址：{addressPreview}</p>
              </div>
            </div>

            <div className="rounded-2xl border border-dashed border-border/80 bg-muted/20 p-4">
              <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                <div>
                  <p className="text-sm font-medium">推荐保持浏览器可见</p>
                  <p className="mt-1 text-sm text-muted-foreground">
                    微软注册经常会出现验证码或人工确认。显示浏览器更稳定，也更方便你及时处理异常。
                  </p>
                </div>
                <div className="flex items-center gap-3">
                  <span className="text-sm text-muted-foreground">{form.showBrowser ? "已开启" : "已关闭"}</span>
                  <Switch
                    checked={form.showBrowser}
                    onCheckedChange={(checked) => setForm((prev) => ({ ...prev, showBrowser: checked }))}
                  />
                </div>
              </div>
            </div>

            <div className="overflow-hidden rounded-2xl border border-border/80 bg-white/75">
              <button
                type="button"
                className="flex w-full items-center justify-between gap-3 px-4 py-4 text-left"
                onClick={() => setShowAdvanced((prev) => !prev)}
              >
                <div>
                  <p className="text-sm font-medium">高级设置 / 更多参数</p>
                  <p className="mt-1 text-sm text-muted-foreground">
                    默认值已经适合大多数情况。只有你想调整起始序号、默认状态、资料信息或超时设置时再展开。
                  </p>
                </div>
                {showAdvanced ? <ChevronUp className="h-4 w-4 shrink-0" /> : <ChevronDown className="h-4 w-4 shrink-0" />}
              </button>

              {showAdvanced ? (
                <>
                  <Separator />
                  <div className="space-y-5 p-4">
                    <div className="grid gap-4 md:grid-cols-2">
                      <div className="space-y-2">
                        <Label htmlFor="batch-start-index">从第几个序号开始</Label>
                        <Input
                          id="batch-start-index"
                          type="number"
                          min={1}
                          value={form.startIndex}
                          onChange={(e) => setForm((prev) => ({ ...prev, startIndex: Number(e.target.value) }))}
                        />
                        <p className="text-xs text-muted-foreground">当前预览：{addressPreview}</p>
                      </div>

                      <div className="space-y-2">
                        <Label htmlFor="batch-status">创建后默认状态</Label>
                        <Select
                          value={form.status}
                          onValueChange={(value) => setForm((prev) => ({ ...prev, status: value }))}
                        >
                          <SelectTrigger id="batch-status">
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value="0">先保存，稍后再确认可用性</SelectItem>
                            <SelectItem value="1">直接标记为可用</SelectItem>
                          </SelectContent>
                        </Select>
                      </div>
                    </div>

                    <div className="grid gap-4 md:grid-cols-2">
                      <div className="space-y-2">
                        <Label htmlFor="batch-first-name">名字</Label>
                        <Input
                          id="batch-first-name"
                          value={form.firstName}
                          onChange={(e) => setForm((prev) => ({ ...prev, firstName: e.target.value }))}
                          placeholder="例如 Octo"
                        />
                      </div>

                      <div className="space-y-2">
                        <Label htmlFor="batch-last-name">姓氏</Label>
                        <Input
                          id="batch-last-name"
                          value={form.lastName}
                          onChange={(e) => setForm((prev) => ({ ...prev, lastName: e.target.value }))}
                          placeholder="例如 Manager"
                        />
                      </div>

                      <div className="space-y-2">
                        <Label htmlFor="batch-birth-month">出生月份</Label>
                        <Select
                          value={form.birthMonth}
                          onValueChange={(value) => setForm((prev) => ({ ...prev, birthMonth: value }))}
                        >
                          <SelectTrigger id="batch-birth-month">
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            {MONTH_OPTIONS.map((month) => (
                              <SelectItem key={month.value} value={month.value}>
                                {month.label}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                        <p className="text-xs text-muted-foreground">当前选择：{selectedMonth}</p>
                      </div>

                      <div className="grid gap-4 sm:grid-cols-2">
                        <div className="space-y-2">
                          <Label htmlFor="batch-birth-day">出生日</Label>
                          <Input
                            id="batch-birth-day"
                            type="number"
                            min={1}
                            max={31}
                            value={form.birthDay}
                            onChange={(e) => setForm((prev) => ({ ...prev, birthDay: Number(e.target.value) }))}
                          />
                        </div>

                        <div className="space-y-2">
                          <Label htmlFor="batch-birth-year">出生年</Label>
                          <Input
                            id="batch-birth-year"
                            type="number"
                            min={1900}
                            max={maxBirthYear}
                            value={form.birthYear}
                            onChange={(e) => setForm((prev) => ({ ...prev, birthYear: Number(e.target.value) }))}
                          />
                        </div>
                      </div>
                    </div>

                    <div className="space-y-2">
                      <Label htmlFor="batch-challenge-timeout">等待验证码/人工确认的最长时间（秒）</Label>
                      <Input
                        id="batch-challenge-timeout"
                        type="number"
                        min={30}
                        max={1800}
                        value={form.challengeTimeout}
                        onChange={(e) => setForm((prev) => ({ ...prev, challengeTimeout: Number(e.target.value) }))}
                      />
                      <p className="text-xs text-muted-foreground">
                        默认 300 秒。只有网络慢或人工验证经常超时时，再考虑调大。
                      </p>
                    </div>

                    <div className="grid gap-4 md:grid-cols-3">
                      <div className="space-y-2">
                        <Label htmlFor="batch-page-timeout">页面步骤超时（秒）</Label>
                        <Input
                          id="batch-page-timeout"
                          type="number"
                          min={30}
                          max={600}
                          value={form.pageTimeout}
                          onChange={(e) => setForm((prev) => ({ ...prev, pageTimeout: Number(e.target.value) }))}
                        />
                        <p className="text-xs text-muted-foreground">默认 90 秒，用于注册页加载和下拉框选择。</p>
                      </div>

                      <div className="space-y-2">
                        <Label htmlFor="batch-oauth-timeout">OAuth 流程超时（秒）</Label>
                        <Input
                          id="batch-oauth-timeout"
                          type="number"
                          min={60}
                          max={900}
                          value={form.oauthTimeout}
                          onChange={(e) => setForm((prev) => ({ ...prev, oauthTimeout: Number(e.target.value) }))}
                        />
                        <p className="text-xs text-muted-foreground">默认 300 秒，授权页慢或需要多次确认时可调大。</p>
                      </div>

                      <div className="space-y-2">
                        <Label htmlFor="batch-api-timeout">Token/API 超时（秒）</Label>
                        <Input
                          id="batch-api-timeout"
                          type="number"
                          min={15}
                          max={300}
                          value={form.apiTimeout}
                          onChange={(e) => setForm((prev) => ({ ...prev, apiTimeout: Number(e.target.value) }))}
                        />
                        <p className="text-xs text-muted-foreground">默认 60 秒，用于 token 交换和模块内 HTTP 请求。</p>
                      </div>
                    </div>
                  </div>
                </>
              ) : null}
            </div>

            <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
              <div className="flex flex-wrap gap-2">
                {summaryBadges.map((item) => (
                  <Badge key={item} variant="outline">
                    {item}
                  </Badge>
                ))}
              </div>
              <Button type="submit" className="w-full sm:w-auto" disabled={loading || !oauthReady}>
                {loading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                开始批量创建
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>

      {result ? (
        <Card>
          <CardHeader>
            <CardTitle>本次执行结果</CardTitle>
            <CardDescription>这里会展示创建结果，方便你快速判断这次是否顺利完成。</CardDescription>
          </CardHeader>
          <CardContent>
            {result.queued ? (
              <div className="rounded-2xl border border-border/80 bg-muted/30 p-4 text-sm text-muted-foreground">
                任务已经进入后台队列
                {result.job_id ? `，job_id: ${result.job_id}` : ""}
                {result.task_id ? `，task_id: ${result.task_id}` : ""}。
              </div>
            ) : (
              <div className="space-y-4">
                <div className="grid gap-3 text-center sm:grid-cols-4">
                  <div className="rounded-2xl bg-muted/50 p-4">
                    <div className="text-2xl font-semibold">{result.requested}</div>
                    <div className="mt-1 text-xs text-muted-foreground">计划创建</div>
                  </div>
                  <div className="rounded-2xl bg-muted/50 p-4">
                    <div className="text-2xl font-semibold">{result.generated}</div>
                    <div className="mt-1 text-xs text-muted-foreground">已生成地址</div>
                  </div>
                  <div className="rounded-2xl bg-emerald-50 p-4">
                    <div className="text-2xl font-semibold text-emerald-600">{result.created}</div>
                    <div className="mt-1 text-xs text-muted-foreground">成功接入</div>
                  </div>
                  <div className="rounded-2xl bg-red-50 p-4">
                    <div className="text-2xl font-semibold text-red-600">{result.failed}</div>
                    <div className="mt-1 text-xs text-muted-foreground">失败</div>
                  </div>
                </div>

                {result.failures.length > 0 ? (
                  <div>
                    <h4 className="mb-2 text-sm font-medium">失败原因</h4>
                    <div className="max-h-[220px] space-y-1 overflow-y-auto rounded-2xl border bg-muted/20 p-3 font-mono text-xs">
                      {result.failures.map((fail, index) => (
                        <div key={`${fail.index}-${index}`} className="text-red-500">
                          [{fail.index}] {fail.address || "未知地址"}: {fail.message}
                        </div>
                      ))}
                    </div>
                  </div>
                ) : null}
              </div>
            )}
          </CardContent>
        </Card>
      ) : null}
    </div>
  );
}
