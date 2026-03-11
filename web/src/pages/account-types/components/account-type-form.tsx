import { useEffect, useState } from "react";
import { Loader2, Plus, X } from "lucide-react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Textarea } from "@/components/ui/textarea";
import { api, extractErrorMessage } from "@/lib/api";
import { parseJSONObjectText } from "@/lib/format";
import type { AccountType } from "@/types";

interface AccountTypeFormProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  initialData?: AccountType | null;
  onSuccess: () => void;
}

export function AccountTypeForm({
  open,
  onOpenChange,
  initialData,
  onSuccess,
}: AccountTypeFormProps) {
  const [loading, setLoading] = useState(false);
  const [form, setForm] = useState({ key: "", name: "", category: "generic", schema: "{}" });
  const [actions, setActions] = useState<string[]>(["REGISTER", "VERIFY"]);
  const [newAction, setNewAction] = useState("");
  const [entryFile, setEntryFile] = useState("");
  const [showAdvanced, setShowAdvanced] = useState(false);

  useEffect(() => {
    if (!open) return;
    if (initialData) {
      setForm({
        key: initialData.key,
        name: initialData.name,
        category: initialData.category,
        schema: JSON.stringify(initialData.schema ?? {}, null, 2),
      });
      const caps = initialData.capabilities as Record<string, unknown>;
      const rawActions = Array.isArray(caps?.actions)
        ? (caps.actions as Array<Record<string, unknown>>)
            .map((a) => String(a.key ?? "").trim().toUpperCase())
            .filter(Boolean)
        : [];
      setActions(rawActions.length > 0 ? rawActions : ["REGISTER", "VERIFY"]);
      const sc = initialData.script_config as Record<string, unknown> | null | undefined;
      const octoMod = sc?.octoModule as Record<string, unknown> | undefined;
      setEntryFile(String(sc?.entry ?? octoMod?.entry ?? "").trim());
    } else {
      setForm({ key: "", name: "", category: "generic", schema: "{}" });
      setActions(["REGISTER", "VERIFY"]);
      setEntryFile("");
    }
    setNewAction("");
    setShowAdvanced(false);
  }, [open, initialData]);

  const addAction = () => {
    const key = newAction.trim().toUpperCase().replace(/[^A-Z0-9_]/g, "_");
    if (!key) return;
    if (!actions.includes(key)) setActions([...actions, key]);
    setNewAction("");
  };

  const removeAction = (key: string) => setActions(actions.filter((a) => a !== key));

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      setLoading(true);
      const capabilities = {
        actions: actions.filter((k) => k.trim()).map((k) => ({ key: k.trim() })),
      };
      const script_config = entryFile.trim() ? { entry: entryFile.trim() } : null;
      const payload = {
        name: form.name.trim(),
        category: form.category,
        schema: parseJSONObjectText(form.schema, "schema"),
        capabilities,
        script_config,
      };
      if (initialData) {
        await api.patchAccountType(initialData.key, payload);
        toast.success("账号类型已更新");
      } else {
        await api.createAccountType({ key: form.key.trim(), ...payload });
        toast.success("账号类型已创建");
      }
      onSuccess();
      onOpenChange(false);
    } catch (error) {
      toast.error(extractErrorMessage(error));
    } finally {
      setLoading(false);
    }
  };

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="w-[600px] sm:max-w-[600px] flex flex-col p-0">
        <SheetHeader className="p-6 border-b">
          <SheetTitle>{initialData ? "编辑类型" : "创建类型"}</SheetTitle>
          <SheetDescription>
            {initialData
              ? `编辑 ${initialData.key} 的配置。`
              : "填写基础字段后即可在后端自动生成对应 OctoModule 脚本入口。"}
          </SheetDescription>
        </SheetHeader>
        <div className="flex-1 overflow-hidden p-6">
          <ScrollArea className="h-full pr-4">
            <form id="account-type-form" onSubmit={handleSubmit} className="space-y-5">

              <div className="space-y-2">
                <Label htmlFor="type-key">键名</Label>
                <Input
                  id="type-key"
                  value={form.key}
                  onChange={(e) => setForm({ ...form, key: e.target.value })}
                  placeholder="generic_x"
                  disabled={!!initialData}
                  required
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="type-name">名称</Label>
                <Input
                  id="type-name"
                  value={form.name}
                  onChange={(e) => setForm({ ...form, name: e.target.value })}
                  placeholder="Generic X"
                  required
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="type-category">分类</Label>
                <Select value={form.category} onValueChange={(v) => setForm({ ...form, category: v })}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="generic">generic</SelectItem>
                    <SelectItem value="email">email</SelectItem>
                    <SelectItem value="system">system</SelectItem>
                  </SelectContent>
                </Select>
                <p className="text-xs text-muted-foreground">仅 generic 类型会自动创建 OctoModule 脚本入口。</p>
              </div>

              <div className="space-y-2">
                <Label>支持的动作</Label>
                <div className="flex flex-wrap gap-1.5 min-h-[36px] rounded-md border border-input bg-muted/30 px-2 py-1.5">
                  {actions.length === 0 && (
                    <span className="text-xs text-muted-foreground self-center">暂无动作</span>
                  )}
                  {actions.map((key) => (
                    <Badge key={key} variant="secondary" className="font-mono gap-1 pr-1">
                      {key}
                      <button
                        type="button"
                        onClick={() => removeAction(key)}
                        className="rounded-sm opacity-60 hover:opacity-100 focus:outline-none"
                      >
                        <X className="h-3 w-3" />
                      </button>
                    </Badge>
                  ))}
                </div>
                <div className="flex gap-2">
                  <Input
                    className="font-mono text-xs h-8"
                    placeholder="新动作名，例如 LOGIN"
                    value={newAction}
                    onChange={(e) => setNewAction(e.target.value.toUpperCase())}
                    onKeyDown={(e) => {
                      if (e.key === "Enter") { e.preventDefault(); addAction(); }
                    }}
                  />
                  <Button type="button" size="sm" variant="outline" onClick={addAction} className="h-8 shrink-0">
                    <Plus className="h-3.5 w-3.5 mr-1" />
                    添加
                  </Button>
                </div>
                <p className="text-xs text-muted-foreground">模块脚本需为每个动作实现对应处理函数。</p>
              </div>

              <div className="space-y-2">
                <Label htmlFor="type-entry">模块入口脚本</Label>
                <Input
                  id="type-entry"
                  className="font-mono text-sm"
                  value={entryFile}
                  onChange={(e) => setEntryFile(e.target.value)}
                  placeholder="留空则使用默认路径 {key}/main.py"
                />
                <p className="text-xs text-muted-foreground">相对于模块目录的路径，例如 social/discord.py。</p>
              </div>

              <div>
                <button
                  type="button"
                  className="text-xs text-muted-foreground hover:text-foreground transition-colors"
                  onClick={() => setShowAdvanced(!showAdvanced)}
                >
                  {showAdvanced ? "▲ 收起高级选项" : "▼ 展开高级选项"}
                </button>
                {showAdvanced && (
                  <div className="mt-3 space-y-2">
                    <Label htmlFor="type-schema">Schema（JSON 对象）</Label>
                    <Textarea
                      id="type-schema"
                      className="font-mono text-xs min-h-[100px]"
                      value={form.schema}
                      onChange={(e) => setForm({ ...form, schema: e.target.value })}
                    />
                    <p className="text-xs text-muted-foreground">描述账号 spec 字段的 JSON Schema，通常保持 {"{}"} 即可。</p>
                  </div>
                )}
              </div>

            </form>
          </ScrollArea>
        </div>
        <SheetFooter className="p-6 border-t">
          <Button type="submit" form="account-type-form" disabled={loading}>
            {loading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            {initialData ? "保存修改" : "立即创建"}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
