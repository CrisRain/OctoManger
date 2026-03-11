import { useMemo, useState } from "react";
import { ChevronDown, ChevronUp, Loader2, Settings2 } from "lucide-react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { Textarea } from "@/components/ui/textarea";
import { api, extractErrorMessage } from "@/lib/api";
import { parseJSONObjectText } from "@/lib/format";
import type { JsonObject } from "@/types";
import {
  OUTLOOK_OAUTH_CALLBACK_MESSAGE,
  type OutlookOAuthCallbackMessage,
} from "./outlook-oauth-bridge";
import { isOutlookConfigReady, splitScopes, type OutlookOAuthConfig } from "./outlook-config";

interface EmailAccountCreateProps {
  config: OutlookOAuthConfig;
  onSuccess: () => void;
  onOpenSetup?: () => void;
}

const CALLBACK_TIMEOUT_MS = 3 * 60 * 1000;

function toBase64URL(input: Uint8Array): string {
  let binary = "";
  for (const item of input) {
    binary += String.fromCharCode(item);
  }
  return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/g, "");
}

function randomBase64URL(size = 32): string {
  const bytes = new Uint8Array(size);
  crypto.getRandomValues(bytes);
  return toBase64URL(bytes);
}

async function createPkcePair() {
  const verifier = randomBase64URL(48);
  const digest = await crypto.subtle.digest("SHA-256", new TextEncoder().encode(verifier));
  return {
    verifier,
    challenge: toBase64URL(new Uint8Array(digest)),
  };
}

function isOutlookConsumerAddress(address: string) {
  const [, domain = ""] = address.toLowerCase().split("@");
  return ["outlook.com", "hotmail.com", "live.com", "msn.com"].includes(domain);
}

function resolveTenant(address: string, configuredTenant: string) {
  const trimmed = configuredTenant.trim();
  if (trimmed) {
    return trimmed;
  }
  return isOutlookConsumerAddress(address) ? "consumers" : "common";
}

function buildTokenURL(tenant: string) {
  return `https://login.microsoftonline.com/${encodeURIComponent(tenant)}/oauth2/v2.0/token`;
}

function waitForOAuthCallback(popup: Window, expectedState: string): Promise<string> {
  return new Promise((resolve, reject) => {
    let done = false;

    const cleanup = () => {
      window.removeEventListener("message", handleMessage);
      window.clearTimeout(timeoutID);
      window.clearInterval(closeCheckID);
    };

    const finish = (next: () => void) => {
      if (done) {
        return;
      }
      done = true;
      cleanup();
      next();
    };

    const handleMessage = (event: MessageEvent<OutlookOAuthCallbackMessage>) => {
      if (event.origin !== window.location.origin) {
        return;
      }

      const payload = event.data;
      if (!payload || payload.type !== OUTLOOK_OAUTH_CALLBACK_MESSAGE) {
        return;
      }

      if ((payload.state ?? "") !== expectedState) {
        finish(() => reject(new Error("授权校验失败，请重新发起登录。")));
        return;
      }

      if (payload.error) {
        const detail = payload.error_description?.trim() || payload.error;
        finish(() => reject(new Error(`微软授权没有完成：${detail}`)));
        return;
      }

      if (!payload.code?.trim()) {
        finish(() => reject(new Error("授权成功了，但回调里没有拿到 code。")));
        return;
      }

      finish(() => resolve(payload.code!.trim()));
    };

    const timeoutID = window.setTimeout(() => {
      finish(() => reject(new Error("等待微软回调超时，请重新登录一次。")));
    }, CALLBACK_TIMEOUT_MS);

    const closeCheckID = window.setInterval(() => {
      if (!popup.closed) {
        return;
      }
      finish(() => reject(new Error("登录窗口在完成前被关闭了，请重新打开。")));
    }, 500);

    window.addEventListener("message", handleMessage);
  });
}

function openOAuthPopup(url: string): Window {
  const width = 520;
  const height = 760;
  const left = window.screenX + Math.max(0, (window.outerWidth - width) / 2);
  const top = window.screenY + Math.max(0, (window.outerHeight - height) / 2);
  const features = [
    `width=${Math.round(width)}`,
    `height=${Math.round(height)}`,
    `left=${Math.round(left)}`,
    `top=${Math.round(top)}`,
    "resizable=yes",
    "scrollbars=yes",
  ].join(",");

  const popup = window.open(url, "outlook-oauth", features);
  if (!popup) {
    throw new Error("登录弹窗被浏览器拦截了，请允许弹窗后再试。");
  }
  popup.focus();
  return popup;
}

export function EmailAccountCreate({ config, onSuccess, onOpenSetup }: EmailAccountCreateProps) {
  const [loading, setLoading] = useState(false);
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [form, setForm] = useState({
    address: "",
    status: "0",
    loginHint: "",
  });
  const [fieldErrors, setFieldErrors] = useState<{ address?: string }>({});

  const [graphConfigDialogOpen, setGraphConfigDialogOpen] = useState(false);
  const [graphConfigText, setGraphConfigText] = useState("{}");
  const [graphConfigDraft, setGraphConfigDraft] = useState("{}");

  const oauthReady = isOutlookConfigReady(config);
  const scopeCount = splitScopes(config.scope).length;

  const graphOverrideCount = useMemo(() => {
    try {
      return Object.keys(parseJSONObjectText(graphConfigText, "graph_config")).length;
    } catch {
      return 0;
    }
  }, [graphConfigText]);

  const openGraphConfigEditor = () => {
    setGraphConfigDraft(graphConfigText);
    setGraphConfigDialogOpen(true);
  };

  const saveGraphConfigEditor = () => {
    try {
      const parsed = parseJSONObjectText(graphConfigDraft, "graph_config");
      setGraphConfigText(JSON.stringify(parsed, null, 2));
      setGraphConfigDialogOpen(false);
      toast.success("高级补充设置已更新。");
    } catch (error) {
      toast.error(extractErrorMessage(error));
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const address = form.address.trim().toLowerCase();
    const scopes = splitScopes(config.scope);
    const clientId = config.clientId.trim();
    const clientSecret = config.clientSecret.trim();
    const redirectURI = config.redirectUri.trim();
    const tenant = resolveTenant(address, config.tenant);
    const loginHint = form.loginHint.trim();
    const mailbox = config.mailbox.trim() || "INBOX";
    const graphBaseURL = config.graphBaseURL.trim() || "https://graph.microsoft.com/v1.0";

    const nextErrors: { address?: string } = {};

    if (!address || !address.includes("@")) {
      nextErrors.address = "请输入完整邮箱地址，例如 name@outlook.com。";
    }

    setFieldErrors(nextErrors);
    if (Object.keys(nextErrors).length > 0) {
      toast.error("先把邮箱地址补正确，再继续。");
      return;
    }

    if (!oauthReady || !clientId || !redirectURI || scopes.length === 0) {
      toast.error("连接设置还没准备好，请先完成“连接设置”里的必填项。");
      onOpenSetup?.();
      return;
    }

    let redirectOrigin = "";
    try {
      redirectOrigin = new URL(redirectURI).origin;
    } catch {
      toast.error("连接设置里的回调地址格式不正确，请先去修正。");
      onOpenSetup?.();
      return;
    }
    if (redirectOrigin !== window.location.origin) {
      toast.error("回调地址必须和当前页面同域，才能自动完成授权。请先回到连接设置修正。");
      onOpenSetup?.();
      return;
    }

    let graphConfigOverrides: JsonObject;
    try {
      graphConfigOverrides = parseJSONObjectText(graphConfigText, "graph_config");
    } catch (error) {
      toast.error(extractErrorMessage(error));
      return;
    }

    setLoading(true);
    let popup: Window | null = null;
    try {
      const state = randomBase64URL(16);
      const pkce = await createPkcePair();

      const authorize = await api.buildOutlookAuthorizeURL({
        client_id: clientId,
        tenant,
        redirect_uri: redirectURI,
        scope: scopes,
        state,
        login_hint: loginHint || undefined,
        code_challenge: pkce.challenge,
        code_challenge_method: "S256",
      });

      popup = openOAuthPopup(authorize.authorize_url);
      const expectedState = authorize.state?.trim() || state;
      const authCode = await waitForOAuthCallback(popup, expectedState);

      const token = await api.exchangeOutlookCode({
        client_id: clientId,
        tenant,
        redirect_uri: redirectURI,
        code: authCode,
        scope: scopes,
        code_verifier: pkce.verifier,
        ...(clientSecret ? { client_secret: clientSecret } : {}),
      });

      const refreshToken = token.refresh_token?.trim();
      if (!refreshToken) {
        throw new Error("已经拿到授权结果，但没有拿到 refresh_token。请确认连接设置里保留了 offline_access 权限。");
      }

      const remoteScopes = splitScopes(token.scope ?? "");
      const resolvedScopes = remoteScopes.length > 0 ? remoteScopes : scopes;

      const graphConfig = {
        auth_method: "graph_oauth2",
        username: address,
        client_id: clientId,
        refresh_token: refreshToken,
        tenant,
        scope: resolvedScopes,
        token_url: token.token_url?.trim() || buildTokenURL(tenant),
        graph_base_url: graphBaseURL,
        mailbox,
        ...(clientSecret ? { client_secret: clientSecret } : {}),
        ...(token.access_token?.trim() ? { access_token: token.access_token.trim() } : {}),
        ...(token.expires_at?.trim() ? { token_expires_at: token.expires_at.trim() } : {}),
      };

      await api.createEmailAccount({
        address,
        provider: "outlook",
        status: Number(form.status),
        graph_config: {
          ...graphConfig,
          ...graphConfigOverrides,
        },
      });

      toast.success(`邮箱 ${address} 已接入完成。`);
      setForm({
        address: "",
        status: "0",
        loginHint: "",
      });
      setFieldErrors({});
      onSuccess();
    } catch (error) {
      toast.error(extractErrorMessage(error));
    } finally {
      if (popup && !popup.closed) {
        popup.close();
      }
      setLoading(false);
    }
  };

  return (
    <>
      <Card className="overflow-hidden border-sky-200/70 bg-gradient-to-br from-white via-white to-sky-50/80 shadow-sm">
        <CardHeader className="gap-4 border-b border-sky-100/80 bg-white/80 backdrop-blur">
          <div className="space-y-3">
            <Badge variant={oauthReady ? "outline" : "secondary"} className="w-fit">
              {oauthReady ? "已连接，可直接添加" : "还差一步：先完成连接设置"}
            </Badge>
            <div className="space-y-2">
              <CardTitle className="text-xl">添加 1 个 Outlook 邮箱</CardTitle>
              <CardDescription className="max-w-2xl leading-6">
                输入邮箱后点击主按钮，系统会弹出微软登录窗口。你完成登录后，这个邮箱就会自动接入，不需要自己处理 OAuth 细节。
              </CardDescription>
            </div>
          </div>

          <div className="grid gap-3 md:grid-cols-3">
            <div className="rounded-2xl border border-border/80 bg-white/85 p-4">
              <p className="text-sm font-medium">1. 输入邮箱</p>
              <p className="mt-2 text-sm text-muted-foreground">只填普通邮箱地址即可，例如 `name@outlook.com`。</p>
            </div>
            <div className="rounded-2xl border border-border/80 bg-white/85 p-4">
              <p className="text-sm font-medium">2. 登录微软</p>
              <p className="mt-2 text-sm text-muted-foreground">系统会自动打开登录弹窗，你只管按微软页面提示操作。</p>
            </div>
            <div className="rounded-2xl border border-border/80 bg-white/85 p-4">
              <p className="text-sm font-medium">3. 自动接入完成</p>
              <p className="mt-2 text-sm text-muted-foreground">成功后会自动写入账号，无需手动复制 token 或 JSON。</p>
            </div>
          </div>
        </CardHeader>

        <CardContent className="space-y-6 pt-6">
          {!oauthReady ? (
            <div className="rounded-2xl border border-amber-200 bg-amber-50/80 p-4">
              <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                <div>
                  <p className="text-sm font-medium">当前还不能直接添加邮箱</p>
                  <p className="mt-1 text-sm text-muted-foreground">
                    先去填好 Outlook 连接设置。完成后，这里会自动使用你保存的默认参数。
                  </p>
                </div>
                {onOpenSetup ? (
                  <Button type="button" variant="outline" onClick={onOpenSetup}>
                    去完成连接设置
                  </Button>
                ) : null}
              </div>
            </div>
          ) : (
            <div className="rounded-2xl border border-emerald-200 bg-emerald-50/70 p-4">
              <p className="text-sm font-medium">将使用已保存的推荐设置</p>
              <p className="mt-1 text-sm text-muted-foreground">
                已启用 {scopeCount} 项授权权限，默认邮箱文件夹为 {config.mailbox.trim() || "INBOX"}。
              </p>
            </div>
          )}

          <form className="space-y-6" onSubmit={handleSubmit}>
            <div className="grid gap-4 md:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="email-address">要接入的邮箱地址</Label>
                <Input
                  id="email-address"
                  value={form.address}
                  onChange={(e) => {
                    setForm((prev) => ({ ...prev, address: e.target.value }));
                    setFieldErrors((prev) => ({ ...prev, address: undefined }));
                  }}
                  placeholder="例如 name@outlook.com"
                  required
                  aria-invalid={Boolean(fieldErrors.address)}
                />
                <p className={`text-xs ${fieldErrors.address ? "text-destructive" : "text-muted-foreground"}`}>
                  {fieldErrors.address ?? "输入你准备登录授权的那个邮箱地址。"}
                </p>
              </div>

              <div className="space-y-2">
                <Label htmlFor="email-status">添加后默认状态</Label>
                <Select
                  value={form.status}
                  onValueChange={(value) => setForm((prev) => ({ ...prev, status: value }))}
                >
                  <SelectTrigger id="email-status">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="0">先保存，稍后再确认可用性</SelectItem>
                    <SelectItem value="1">直接标记为可用</SelectItem>
                  </SelectContent>
                </Select>
                <p className="text-xs text-muted-foreground">不确定时保留默认选项即可。</p>
              </div>
            </div>

            <div className="overflow-hidden rounded-2xl border border-border/80 bg-white/75">
              <button
                type="button"
                className="flex w-full items-center justify-between gap-3 px-4 py-4 text-left"
                onClick={() => setShowAdvanced((prev) => !prev)}
              >
                <div>
                  <p className="text-sm font-medium">高级选项 / 更多设置</p>
                  <p className="mt-1 text-sm text-muted-foreground">
                    不确定就别展开。这里主要给想预填登录邮箱或覆盖底层参数的高级用户使用。
                  </p>
                </div>
                {showAdvanced ? <ChevronUp className="h-4 w-4 shrink-0" /> : <ChevronDown className="h-4 w-4 shrink-0" />}
              </button>

              {showAdvanced ? (
                <>
                  <Separator />
                  <div className="space-y-4 p-4">
                    <div className="space-y-2">
                      <Label htmlFor="oauth-login-hint">登录页预填邮箱（可选）</Label>
                      <Input
                        id="oauth-login-hint"
                        value={form.loginHint}
                        onChange={(e) => setForm((prev) => ({ ...prev, loginHint: e.target.value }))}
                        placeholder="例如 name@outlook.com"
                      />
                      <p className="text-xs text-muted-foreground">
                        填写后，微软登录页会优先帮你带上这个邮箱。留空也完全没问题。
                      </p>
                    </div>

                    <div className="rounded-2xl border border-dashed border-border/80 bg-muted/20 p-4">
                      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                        <div>
                          <p className="text-sm font-medium">高级补充设置</p>
                          <p className="mt-1 text-sm text-muted-foreground">
                            只有在你要手动覆盖系统生成的底层参数时才需要填写。默认留空即可。
                          </p>
                        </div>
                        <Button type="button" variant="outline" onClick={openGraphConfigEditor} disabled={loading}>
                          <Settings2 className="mr-2 h-4 w-4" />
                          {graphOverrideCount > 0 ? `已设置 ${graphOverrideCount} 项` : "打开 JSON 设置"}
                        </Button>
                      </div>
                    </div>
                  </div>
                </>
              ) : null}
            </div>

            <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
              <div className="flex flex-wrap items-center gap-2 text-sm text-muted-foreground">
                <span>默认使用已保存的 Outlook 连接设置。</span>
                {graphOverrideCount > 0 ? <Badge variant="outline">附加了 {graphOverrideCount} 项高级覆写</Badge> : null}
              </div>
              <div className="flex flex-col gap-2 sm:flex-row">
                <Button type="button" variant="outline" onClick={openGraphConfigEditor} disabled={loading}>
                  <Settings2 className="mr-2 h-4 w-4" />
                  高级补充设置
                </Button>
                <Button type="submit" disabled={loading || !oauthReady}>
                  {loading ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
                  开始授权并添加
                </Button>
              </div>
            </div>
          </form>
        </CardContent>
      </Card>

      <Dialog open={graphConfigDialogOpen} onOpenChange={setGraphConfigDialogOpen}>
        <DialogContent className="max-w-3xl">
          <DialogHeader>
            <DialogTitle>高级补充设置（JSON）</DialogTitle>
            <DialogDescription>
              只有你要覆盖系统自动生成的底层参数时才需要填写。保持为
              {" "}
              <code className="rounded bg-muted px-1 py-0.5 text-xs">{`{}`}</code>
              {" "}就表示完全使用系统默认值。
            </DialogDescription>
          </DialogHeader>

          <Textarea
            className="min-h-[320px] font-mono text-xs"
            value={graphConfigDraft}
            onChange={(e) => setGraphConfigDraft(e.target.value)}
            placeholder={`{\n  "mailbox": "Archive"\n}`}
          />

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setGraphConfigDialogOpen(false)}>
              取消
            </Button>
            <Button type="button" onClick={saveGraphConfigEditor}>
              保存补充设置
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
