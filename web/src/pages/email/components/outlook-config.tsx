import { useEffect, useMemo, useState } from "react";
import { ChevronDown, ChevronUp, CircleHelp, Loader2, ShieldCheck } from "lucide-react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { Textarea } from "@/components/ui/textarea";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { api, extractErrorMessage } from "@/lib/api";

export interface OutlookOAuthConfig {
  clientId: string;
  clientSecret: string;
  tenant: string;
  redirectUri: string;
  scope: string;
  mailbox: string;
  graphBaseURL: string;
}

type TenantMode = "consumers" | "common" | "custom";

type OutlookConfigFieldErrors = Partial<Record<"clientId" | "redirectUri" | "scope", string>>;

export const DEFAULT_OUTLOOK_CONFIG: OutlookOAuthConfig = {
  clientId: "",
  clientSecret: "",
  tenant: "consumers",
  redirectUri: "http://localhost:8080/oauth/callback",
  scope: "offline_access openid profile email https://graph.microsoft.com/Mail.Read",
  mailbox: "INBOX",
  graphBaseURL: "https://graph.microsoft.com/v1.0",
};

const CONFIG_KEY = "outlook_oauth_config";

const TENANT_OPTIONS: Array<{ value: TenantMode; label: string; description: string }> = [
  {
    value: "consumers",
    label: "只接个人 Outlook 邮箱",
    description: "适用于 outlook.com、hotmail.com 等个人邮箱，推荐保留这个选项。",
  },
  {
    value: "common",
    label: "同时支持个人和工作/学校账号",
    description: "如果你还要接入企业或学校邮箱，再切换到这个选项。",
  },
  {
    value: "custom",
    label: "使用自定义 Tenant / 目录 ID",
    description: "只有你明确知道自己的 Tenant ID 时才需要填写。",
  },
];

function resolveTenantMode(tenant: string): TenantMode {
  if (tenant === "common") {
    return "common";
  }
  if (tenant === "consumers") {
    return "consumers";
  }
  return "custom";
}

export function splitScopes(raw: string): string[] {
  const normalized = raw.trim().replace(/,/g, " ");
  if (!normalized) {
    return [];
  }
  return Array.from(new Set(normalized.split(/\s+/).map((item) => item.trim()).filter(Boolean)));
}

export function getRecommendedRedirectUri(origin = typeof window !== "undefined" ? window.location.origin : "") {
  return origin ? `${origin}/oauth/callback` : DEFAULT_OUTLOOK_CONFIG.redirectUri;
}

export function getOutlookConfigChecklist(
  config: OutlookOAuthConfig,
  origin = typeof window !== "undefined" ? window.location.origin : "",
) {
  const scopes = splitScopes(config.scope);
  let redirectReady = false;
  let redirectHint = "请填写完整的回调地址。";

  if (config.redirectUri.trim()) {
    try {
      const redirectURL = new URL(config.redirectUri);
      if (!origin || redirectURL.origin === origin) {
        redirectReady = true;
        redirectHint = "已和当前站点同域，授权后可以自动完成。";
      } else {
        redirectHint = `建议改成 ${getRecommendedRedirectUri(origin)}，否则授权回调无法自动完成。`;
      }
    } catch {
      redirectHint = "地址格式不正确，请检查是否包含 http:// 或 https://。";
    }
  }

  const checks = [
    {
      key: "clientId",
      label: "应用 ID",
      done: Boolean(config.clientId.trim()),
      hint: "去微软应用注册页复制 Application (client) ID，粘贴到这里。",
    },
    {
      key: "redirectUri",
      label: "回调地址",
      done: redirectReady,
      hint: redirectHint,
    },
    {
      key: "scope",
      label: "授权权限",
      done: scopes.length > 0,
      hint: scopes.length > 0 ? `已配置 ${scopes.length} 项权限，默认值已经够大多数场景使用。` : "请至少保留 1 项权限。",
    },
  ];

  return {
    checks,
    ready: checks.every((item) => item.done),
    scopeCount: scopes.length,
  };
}

export function isOutlookConfigReady(
  config: OutlookOAuthConfig,
  origin = typeof window !== "undefined" ? window.location.origin : "",
) {
  return getOutlookConfigChecklist(config, origin).ready;
}

function validateOutlookConfigDraft(
  draft: OutlookOAuthConfig,
  origin = typeof window !== "undefined" ? window.location.origin : "",
): OutlookConfigFieldErrors {
  const errors: OutlookConfigFieldErrors = {};

  if (!draft.clientId.trim()) {
    errors.clientId = "请粘贴微软后台里的 Application (client) ID。";
  }

  if (!draft.redirectUri.trim()) {
    errors.redirectUri = `请填写回调地址，推荐直接使用 ${getRecommendedRedirectUri(origin)}。`;
  } else {
    try {
      const redirectURL = new URL(draft.redirectUri);
      if (origin && redirectURL.origin !== origin) {
        errors.redirectUri = `为了自动完成授权，回调地址必须和当前页面同域。建议改成 ${getRecommendedRedirectUri(origin)}。`;
      }
    } catch {
      errors.redirectUri = "回调地址格式不正确，请检查是否包含 http:// 或 https://。";
    }
  }

  if (splitScopes(draft.scope).length === 0) {
    errors.scope = "至少保留 1 项授权权限。默认值已经足够，大多数人不需要改这里。";
  }

  return errors;
}

export function useOutlookConfig() {
  const [config, setConfig] = useState<OutlookOAuthConfig>(DEFAULT_OUTLOOK_CONFIG);
  const [configLoading, setConfigLoading] = useState(true);

  useEffect(() => {
    api
      .getConfig(CONFIG_KEY)
      .then((res) => {
        if (res.value && typeof res.value === "object") {
          setConfig({ ...DEFAULT_OUTLOOK_CONFIG, ...(res.value as Partial<OutlookOAuthConfig>) });
        }
      })
      .catch(() => {
        // Keep sensible defaults when the config is missing.
      })
      .finally(() => setConfigLoading(false));
  }, []);

  const saveConfig = async (next: OutlookOAuthConfig): Promise<void> => {
    await api.setConfig(CONFIG_KEY, next);
    setConfig(next);
  };

  return { config, configLoading, saveConfig };
}

function LabelWithTip({
  htmlFor,
  label,
  tip,
}: {
  htmlFor: string;
  label: string;
  tip?: string;
}) {
  return (
    <div className="flex items-center gap-1.5">
      <Label htmlFor={htmlFor}>{label}</Label>
      {tip ? (
        <TooltipProvider delayDuration={120}>
          <Tooltip>
            <TooltipTrigger asChild>
              <button
                type="button"
                className="inline-flex h-5 w-5 items-center justify-center rounded-full text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
                aria-label={`${label} 说明`}
              >
                <CircleHelp className="h-3.5 w-3.5" />
              </button>
            </TooltipTrigger>
            <TooltipContent>{tip}</TooltipContent>
          </Tooltip>
        </TooltipProvider>
      ) : null}
    </div>
  );
}

interface OutlookConfigPanelProps {
  config: OutlookOAuthConfig;
  configLoading: boolean;
  onSave: (config: OutlookOAuthConfig) => Promise<void>;
}

export function OutlookConfigPanel({ config, configLoading, onSave }: OutlookConfigPanelProps) {
  const [draft, setDraft] = useState<OutlookOAuthConfig>(config);
  const [saving, setSaving] = useState(false);
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [tenantMode, setTenantMode] = useState<TenantMode>(resolveTenantMode(config.tenant));
  const [customTenant, setCustomTenant] = useState(resolveTenantMode(config.tenant) === "custom" ? config.tenant : "");
  const [errors, setErrors] = useState<OutlookConfigFieldErrors>({});

  useEffect(() => {
    setDraft(config);
    const nextMode = resolveTenantMode(config.tenant);
    setTenantMode(nextMode);
    setCustomTenant(nextMode === "custom" ? config.tenant : "");
    setErrors({});
  }, [config]);

  const configStatus = useMemo(() => getOutlookConfigChecklist(draft), [draft]);
  const currentOrigin = typeof window !== "undefined" ? window.location.origin : "";
  const recommendedRedirectUri = getRecommendedRedirectUri(currentOrigin);
  const completedChecks = configStatus.checks.filter((item) => item.done).length;
  const hasUnsavedChanges = JSON.stringify(draft) !== JSON.stringify(config);
  const selectedTenantOption = TENANT_OPTIONS.find((item) => item.value === tenantMode);

  const updateDraft = <K extends keyof OutlookOAuthConfig>(key: K, value: OutlookOAuthConfig[K]) => {
    setDraft((prev) => ({ ...prev, [key]: value }));
    setErrors((prev) => ({ ...prev, [key]: undefined }));
  };

  const handleTenantModeChange = (value: string) => {
    const nextMode = value as TenantMode;
    setTenantMode(nextMode);
    if (nextMode === "custom") {
      updateDraft("tenant", customTenant);
      return;
    }
    updateDraft("tenant", nextMode);
  };

  const handleCustomTenantChange = (value: string) => {
    setCustomTenant(value);
    updateDraft("tenant", value);
  };

  const handleSave = async () => {
    const nextErrors = validateOutlookConfigDraft(draft, currentOrigin);
    setErrors(nextErrors);
    if (Object.keys(nextErrors).length > 0) {
      toast.error("还有必填项没完成，先把页面里的红色提示处理掉。");
      return;
    }

    setSaving(true);
    try {
      await onSave(draft);
      toast.success("连接设置已保存，现在可以去添加邮箱了。");
    } catch (error) {
      toast.error(extractErrorMessage(error));
    } finally {
      setSaving(false);
    }
  };

  const handleReset = () => {
    setDraft(DEFAULT_OUTLOOK_CONFIG);
    setTenantMode(resolveTenantMode(DEFAULT_OUTLOOK_CONFIG.tenant));
    setCustomTenant("");
    setErrors({});
  };

  return (
    <Card className="overflow-hidden border-amber-200/70 bg-gradient-to-br from-white via-white to-amber-50/80 shadow-sm">
      <CardHeader className="gap-4 border-b border-amber-100/80 bg-white/80 backdrop-blur">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div className="space-y-3">
            <Badge variant={configStatus.ready ? "default" : "secondary"} className="w-fit">
              <ShieldCheck className="mr-1.5 h-3.5 w-3.5" />
              {configStatus.ready ? "连接设置已就绪" : "先完成这一步"}
            </Badge>
            <div className="space-y-2">
              <CardTitle className="text-xl">先把 Outlook 连接打通</CardTitle>
              <CardDescription className="max-w-2xl leading-6">
                大多数人只需要填写 2 项: 「应用 ID」和「回调地址」。其余参数系统已经填好了推荐值，不确定时保持默认即可。
              </CardDescription>
            </div>
          </div>

          <div className="rounded-2xl border border-amber-100 bg-white/90 p-4 shadow-sm lg:min-w-[280px]">
            <div className="text-xs font-medium uppercase tracking-[0.22em] text-muted-foreground">完成度</div>
            <div className="mt-2 text-3xl font-semibold">{completedChecks}/3</div>
            <p className="mt-1 text-sm text-muted-foreground">
              {configStatus.ready ? "已经可以正常发起 Outlook 授权。" : "补齐下面的必填项后，就能开始接入邮箱。"}
            </p>
          </div>
        </div>

        <div className="grid gap-3 md:grid-cols-3">
          {configStatus.checks.map((item) => (
            <div
              key={item.key}
              className={`rounded-2xl border px-4 py-3 ${
                item.done ? "border-emerald-200 bg-emerald-50/70" : "border-border/80 bg-white/85"
              }`}
            >
              <div className="flex items-center justify-between gap-3">
                <p className="text-sm font-medium">{item.label}</p>
                <Badge variant={item.done ? "outline" : "secondary"}>{item.done ? "已完成" : "待处理"}</Badge>
              </div>
              <p className="mt-2 text-xs leading-5 text-muted-foreground">{item.hint}</p>
            </div>
          ))}
        </div>
      </CardHeader>

      <CardContent className="space-y-6 pt-6">
        <div className="grid gap-4 md:grid-cols-2">
          <div className="space-y-2">
            <LabelWithTip
              htmlFor="cfg-client-id"
              label="应用 ID"
              tip="微软应用后台里的 Application (client) ID。把那串长字符串完整贴过来即可。"
            />
            <Input
              id="cfg-client-id"
              value={draft.clientId}
              onChange={(e) => updateDraft("clientId", e.target.value)}
              placeholder="例如 00000000-1111-2222-3333-444444444444"
              aria-invalid={Boolean(errors.clientId)}
            />
            <p className={`text-xs ${errors.clientId ? "text-destructive" : "text-muted-foreground"}`}>
              {errors.clientId ?? "这是 Outlook 授权最关键的一项，没有它就无法打开微软登录页。"}
            </p>
          </div>

          <div className="space-y-2">
            <LabelWithTip
              htmlFor="cfg-redirect-uri"
              label="授权完成后返回的地址"
              tip="微软登录成功后会跳回这个地址，页面才能自动接住授权结果。通常直接用当前站点的默认地址即可。"
            />
            <Input
              id="cfg-redirect-uri"
              value={draft.redirectUri}
              onChange={(e) => updateDraft("redirectUri", e.target.value)}
              placeholder={recommendedRedirectUri}
              aria-invalid={Boolean(errors.redirectUri)}
            />
            <div className="flex flex-wrap items-center gap-2 text-xs">
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => updateDraft("redirectUri", recommendedRedirectUri)}
                disabled={saving || configLoading}
              >
                一键填入当前站点地址
              </Button>
              <span className={errors.redirectUri ? "text-destructive" : "text-muted-foreground"}>
                {errors.redirectUri ?? `推荐使用 ${recommendedRedirectUri}`}
              </span>
            </div>
          </div>
        </div>

        <div className="rounded-2xl border border-dashed border-amber-200 bg-amber-50/60 p-4">
          <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <p className="text-sm font-medium">系统已自动填好推荐值</p>
              <p className="mt-1 text-sm text-muted-foreground">
                默认邮箱文件夹: <span className="font-medium text-foreground">{draft.mailbox || "INBOX"}</span>
                {" · "}
                默认授权权限: <span className="font-medium text-foreground">{configStatus.scopeCount} 项</span>
              </p>
            </div>
            <Badge variant="outline">不确定就保持默认</Badge>
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
                只有在你明确知道自己在改什么时才需要展开，例如自定义 Tenant、权限范围或接口地址。
              </p>
            </div>
            {showAdvanced ? <ChevronUp className="h-4 w-4 shrink-0" /> : <ChevronDown className="h-4 w-4 shrink-0" />}
          </button>

          {showAdvanced ? (
            <>
              <Separator />
              <div className="space-y-5 p-4">
                <div className="space-y-2">
                  <LabelWithTip
                    htmlFor="cfg-tenant-mode"
                    label="这套连接要支持哪类账号"
                    tip="个人 Outlook 邮箱一般保持“只接个人 Outlook 邮箱”即可。企业或学校账号才需要改。"
                  />
                  <Select value={tenantMode} onValueChange={handleTenantModeChange}>
                    <SelectTrigger id="cfg-tenant-mode">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {TENANT_OPTIONS.map((option) => (
                        <SelectItem key={option.value} value={option.value}>
                          {option.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <p className="text-xs text-muted-foreground">{selectedTenantOption?.description}</p>
                </div>

                {tenantMode === "custom" ? (
                  <div className="space-y-2">
                    <LabelWithTip
                      htmlFor="cfg-tenant"
                      label="自定义 Tenant / 目录 ID"
                      tip="只有你已经拿到明确的 Tenant ID 时才填写。留空时，系统会继续使用默认行为。"
                    />
                    <Input
                      id="cfg-tenant"
                      value={customTenant}
                      onChange={(e) => handleCustomTenantChange(e.target.value)}
                      placeholder="例如 11111111-2222-3333-4444-555555555555"
                    />
                  </div>
                ) : null}

                <div className="grid gap-4 md:grid-cols-2">
                  <div className="space-y-2">
                    <LabelWithTip
                      htmlFor="cfg-client-secret"
                      label="应用密钥（可选）"
                      tip="大多数个人邮箱场景可以留空。只有你的微软应用要求机密客户端时才需要填写。"
                    />
                    <Input
                      id="cfg-client-secret"
                      type="password"
                      value={draft.clientSecret}
                      onChange={(e) => updateDraft("clientSecret", e.target.value)}
                      placeholder="只有机密客户端才需要填写"
                    />
                    <p className="text-xs text-muted-foreground">批量导入不会发送这个值，通常手动授权时才会用到。</p>
                  </div>

                  <div className="space-y-2">
                    <LabelWithTip
                      htmlFor="cfg-mailbox"
                      label="默认读取哪个邮箱文件夹"
                      tip="绝大多数场景保持 INBOX 即可，也就是“收件箱”。"
                    />
                    <Input
                      id="cfg-mailbox"
                      value={draft.mailbox}
                      onChange={(e) => updateDraft("mailbox", e.target.value)}
                      placeholder="INBOX"
                    />
                    <p className="text-xs text-muted-foreground">不确定时不要改，系统会从这里读取邮件。</p>
                  </div>
                </div>

                <div className="space-y-2">
                  <LabelWithTip
                    htmlFor="cfg-scope"
                    label="授权权限列表"
                    tip="这是微软 OAuth 的 scope。默认值已经覆盖常见收信场景，只有你知道要新增权限时才修改。"
                  />
                  <Textarea
                    id="cfg-scope"
                    className="min-h-[110px] font-mono text-xs"
                    value={draft.scope}
                    onChange={(e) => updateDraft("scope", e.target.value)}
                    placeholder={DEFAULT_OUTLOOK_CONFIG.scope}
                    aria-invalid={Boolean(errors.scope)}
                  />
                  <p className={`text-xs ${errors.scope ? "text-destructive" : "text-muted-foreground"}`}>
                    {errors.scope ?? `已识别 ${configStatus.scopeCount} 项权限，默认值通常已经够用。`}
                  </p>
                </div>

                <div className="space-y-2">
                  <LabelWithTip
                    htmlFor="cfg-graph-base-url"
                    label="微软接口地址"
                    tip="默认就是 Microsoft Graph 的正式地址。除非你在调试特殊环境，否则不要修改。"
                  />
                  <Input
                    id="cfg-graph-base-url"
                    value={draft.graphBaseURL}
                    onChange={(e) => updateDraft("graphBaseURL", e.target.value)}
                    placeholder={DEFAULT_OUTLOOK_CONFIG.graphBaseURL}
                  />
                  <p className="text-xs text-muted-foreground">保持默认即可，改错会导致后续收信和读信失败。</p>
                </div>
              </div>
            </>
          ) : null}
        </div>

        <div className="flex flex-col gap-3 border-t pt-4 sm:flex-row sm:items-center sm:justify-between">
          <p className="text-sm text-muted-foreground">
            {hasUnsavedChanges ? "你有未保存的修改。" : "当前显示的是已保存的设置。"}
          </p>
          <div className="flex flex-wrap gap-2">
            <Button type="button" variant="outline" onClick={handleReset} disabled={saving || configLoading}>
              恢复推荐值
            </Button>
            <Button type="button" onClick={() => void handleSave()} disabled={saving || configLoading}>
              {saving ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
              保存并继续
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
