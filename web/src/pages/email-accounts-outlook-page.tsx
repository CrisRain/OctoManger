import { useCallback, useEffect, useMemo, useState } from "react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { PageHeader } from "@/components/page-header";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { api, extractErrorMessage } from "@/lib/api";
import type { EmailAccount } from "@/types";
import { EmailAccountBatchRegister } from "./email/components/email-account-batch";
import { EmailAccountCreate } from "./email/components/email-account-create";
import { EmailAccountDetails } from "./email/components/email-account-details";
import { EmailAccountEdit } from "./email/components/email-account-edit";
import { EmailAccountImport } from "./email/components/email-account-import";
import { EmailAccountList } from "./email/components/email-account-list";
import {
  OutlookConfigPanel,
  isOutlookConfigReady,
  splitScopes,
  useOutlookConfig,
} from "./email/components/outlook-config";

type OutlookTab = "list" | "create" | "import" | "batch" | "config";

export function EmailAccountsOutlookPage() {
  const [items, setItems] = useState<EmailAccount[]>([]);
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const [pageSize, setPageSize] = useState(10);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState<OutlookTab>("list");
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set());
  const [batchLoading, setBatchLoading] = useState(false);

  const [detailsOpen, setDetailsOpen] = useState(false);
  const [selectedAccountID, setSelectedAccountID] = useState<number | null>(null);

  const [editOpen, setEditOpen] = useState(false);
  const [editAccount, setEditAccount] = useState<typeof items[number] | null>(null);

  const { config, configLoading, saveConfig } = useOutlookConfig();
  const configReady = isOutlookConfigReady(config);
  const scopeCount = splitScopes(config.scope).length;

  const load = useCallback(async () => {
    try {
      setLoading(true);
      const res = await api.listEmailAccounts({ limit: pageSize, offset });
      setItems(res.items);
      setTotal(res.total);
      setSelectedIds(new Set());
    } catch (error) {
      toast.error(extractErrorMessage(error));
    } finally {
      setLoading(false);
    }
  }, [offset, pageSize]);

  const openTab = (tab: OutlookTab) => setActiveTab(tab);

  const handlePageSizeChange = (newSize: number) => {
    setOffset(0);
    setPageSize(newSize);
  };

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    if (!loading && !configLoading && total === 0 && !configReady && activeTab === "list") {
      setActiveTab("config");
    }
  }, [activeTab, configLoading, configReady, loading, total]);

  const handleVerify = async (id: number) => {
    try {
      await api.verifyEmailAccount(id);
      toast.success("这个邮箱已标记为可用。");
      await load();
    } catch (error) {
      toast.error(extractErrorMessage(error));
    }
  };

  const handleDelete = async (id: number) => {
    if (!confirm("确定要删除这个邮箱吗？")) {
      return;
    }
    try {
      await api.deleteEmailAccount(id);
      toast.success("邮箱已删除。");
      await load();
    } catch (error) {
      toast.error(extractErrorMessage(error));
    }
  };

  const handleViewDetails = (id: number) => {
    setSelectedAccountID(id);
    setDetailsOpen(true);
  };

  const handleEdit = (id: number) => {
    const account = items.find((a) => a.id === id) ?? null;
    setEditAccount(account);
    setEditOpen(true);
  };

  const handleToggleSelect = (id: number) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  const handleToggleSelectAll = () => {
    if (selectedIds.size === items.length) {
      setSelectedIds(new Set());
      return;
    }
    setSelectedIds(new Set(items.map((item) => item.id)));
  };

  const handleBatchDelete = async () => {
    if (!confirm(`确定要删除选中的 ${selectedIds.size} 个邮箱吗？`)) {
      return;
    }
    setBatchLoading(true);
    try {
      const result = await api.batchDeleteEmailAccounts([...selectedIds]);
      if (result.queued) {
        toast.success(
          `删除任务已提交${result.job_id ? `（job: ${result.job_id}）` : ""}${result.task_id ? `（task: ${result.task_id}）` : ""}`,
        );
      } else if (result.failed > 0) {
        toast.warning(`已删除 ${result.success} 个，另有 ${result.failed} 个失败。`);
      } else {
        toast.success(`已删除 ${result.success} 个邮箱。`);
      }
      await load();
    } catch (error) {
      toast.error(extractErrorMessage(error));
    } finally {
      setBatchLoading(false);
    }
  };

  const handleBatchVerify = async () => {
    setBatchLoading(true);
    try {
      const result = await api.batchVerifyEmailAccounts([...selectedIds]);
      if (result.queued) {
        toast.success(
          `验证任务已提交${result.job_id ? `（job: ${result.job_id}）` : ""}${result.task_id ? `（task: ${result.task_id}）` : ""}`,
        );
      } else if (result.failed > 0) {
        toast.warning(`已标记 ${result.success} 个为可用，另有 ${result.failed} 个失败。`);
      } else {
        toast.success(`已标记 ${result.success} 个邮箱为可用。`);
      }
      await load();
    } catch (error) {
      toast.error(extractErrorMessage(error));
    } finally {
      setBatchLoading(false);
    }
  };

  const heroSteps = useMemo(
    () => [
      {
        tab: "config" as const,
        title: "先连接 Outlook",
        description: configReady
          ? "应用 ID、回调地址和授权权限都已就绪。"
          : "先填好 Outlook 连接设置，后面的添加和批量操作才能一步完成。",
        badge: configReady ? "已完成" : "第 1 步",
      },
      {
        tab: "create" as const,
        title: "添加第一个邮箱",
        description:
          total > 0 ? `当前已经接入 ${total} 个邮箱，还可以继续添加。` : "建议先手动接入 1 个邮箱，确认流程没问题。",
        badge: total > 0 ? "已开始使用" : "第 2 步",
      },
      {
        tab: "batch" as const,
        title: "批量导入或批量新建",
        description: "当你确认单个流程跑通后，再来这里处理大批量账号，效率更高。",
        badge: "第 3 步",
      },
    ],
    [configReady, total],
  );

  const primaryAction = configReady ? (total > 0 ? "create" : "create") : "config";

  return (
    <div className="space-y-5">
      <PageHeader
        title="Outlook 邮箱接入"
        description="把原本复杂的连接流程拆成 3 步：先连上 Outlook、再添加邮箱、最后批量处理。系统已经为大多数场景准备了默认推荐值。"
        action={
          <Button onClick={() => openTab(primaryAction)}>
            {configReady ? "添加第一个邮箱" : "先完成连接设置"}
          </Button>
        }
      />

      <Card className="overflow-hidden border-amber-200/70 bg-gradient-to-br from-amber-50 via-white to-sky-50 shadow-sm">
        <CardContent className="grid gap-6 p-6 lg:grid-cols-[1.35fr,0.95fr]">
          <div className="space-y-4">
            <Badge variant={configReady ? "outline" : "secondary"} className="w-fit">
              {configReady ? "当前已可直接接入邮箱" : "推荐从第 1 步开始"}
            </Badge>
            <div className="space-y-2">
              <h3 className="text-3xl font-semibold tracking-tight">跟着 3 步完成 Outlook 接入</h3>
              <p className="max-w-2xl text-sm leading-6 text-muted-foreground">
                系统已经准备了默认返回地址、默认权限和默认邮箱文件夹。普通使用场景通常不需要理解微软底层参数。
              </p>
            </div>

            <div className="flex flex-wrap gap-3">
              <Button size="lg" onClick={() => openTab(primaryAction)}>
                {configReady ? "现在去添加邮箱" : "先完成连接设置"}
              </Button>
              <Button size="lg" variant="outline" onClick={() => openTab("list")}>
                查看已接入邮箱
              </Button>
            </div>

            <div className="flex flex-wrap gap-2">
              <Badge variant="outline">{configReady ? "连接已就绪" : "连接未完成"}</Badge>
              <Badge variant="outline">默认权限 {scopeCount} 项</Badge>
              <Badge variant="outline">{total > 0 ? `已接入 ${total} 个邮箱` : "还没有接入邮箱"}</Badge>
            </div>
          </div>

          <div className="grid gap-3">
            {heroSteps.map((step) => (
              <button
                key={step.tab}
                type="button"
                onClick={() => openTab(step.tab)}
                className="rounded-2xl border border-white/90 bg-white/85 px-4 py-4 text-left shadow-sm transition-transform hover:-translate-y-0.5"
              >
                <div className="flex items-center justify-between gap-3">
                  <p className="text-sm font-medium">{step.title}</p>
                  <Badge variant="outline">{step.badge}</Badge>
                </div>
                <p className="mt-2 text-sm leading-6 text-muted-foreground">{step.description}</p>
              </button>
            ))}
          </div>
        </CardContent>
      </Card>

      <Tabs value={activeTab} onValueChange={(value) => setActiveTab(value as OutlookTab)} className="space-y-4">
        <TabsList className="h-auto w-full flex-wrap justify-start gap-2 bg-transparent p-0">
          <TabsTrigger value="list" className="rounded-full border border-border bg-card px-4 py-2 data-[state=active]:border-foreground">
            已接入邮箱
          </TabsTrigger>
          <TabsTrigger value="create" className="rounded-full border border-border bg-card px-4 py-2 data-[state=active]:border-foreground">
            添加一个
          </TabsTrigger>
          <TabsTrigger value="import" className="rounded-full border border-border bg-card px-4 py-2 data-[state=active]:border-foreground">
            批量导入
          </TabsTrigger>
          <TabsTrigger value="batch" className="rounded-full border border-border bg-card px-4 py-2 data-[state=active]:border-foreground">
            批量新建
          </TabsTrigger>
          <TabsTrigger value="config" className="rounded-full border border-border bg-card px-4 py-2 data-[state=active]:border-foreground">
            连接设置
          </TabsTrigger>
        </TabsList>

        <TabsContent value="list" className="space-y-4">
          <EmailAccountList
            items={items}
            total={total}
            offset={offset}
            pageSize={pageSize}
            loading={loading}
            selectedIds={selectedIds}
            batchLoading={batchLoading}
            onVerify={handleVerify}
            onDelete={handleDelete}
            onEdit={handleEdit}
            onViewDetails={handleViewDetails}
            onPageChange={setOffset}
            onPageSizeChange={handlePageSizeChange}
            onToggleSelect={handleToggleSelect}
            onToggleSelectAll={handleToggleSelectAll}
            onBatchDelete={() => void handleBatchDelete()}
            onBatchVerify={() => void handleBatchVerify()}
            onRefresh={() => void load()}
          />
        </TabsContent>

        <TabsContent value="create">
          <EmailAccountCreate
            config={config}
            onOpenSetup={() => openTab("config")}
            onSuccess={() => {
              void load();
              setActiveTab("list");
            }}
          />
        </TabsContent>

        <TabsContent value="import">
          <EmailAccountImport
            config={config}
            onOpenSetup={() => openTab("config")}
            onSuccess={() => {
              void load();
              setActiveTab("list");
            }}
          />
        </TabsContent>

        <TabsContent value="batch">
          <EmailAccountBatchRegister
            config={config}
            onOpenSetup={() => openTab("config")}
            onSuccess={() => {
              void load();
              setActiveTab("list");
            }}
          />
        </TabsContent>

        <TabsContent value="config">
          <OutlookConfigPanel config={config} configLoading={configLoading} onSave={saveConfig} />
        </TabsContent>
      </Tabs>

      <EmailAccountDetails accountId={selectedAccountID} open={detailsOpen} onOpenChange={setDetailsOpen} />

      <EmailAccountEdit account={editAccount} open={editOpen} onOpenChange={setEditOpen} onSuccess={() => void load()} />
    </div>
  );
}
