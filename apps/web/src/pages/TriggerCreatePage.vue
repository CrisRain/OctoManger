<script setup lang="ts">
import { ref } from "vue";
import { useRouter } from "vue-router";
import { useJobDefinitions } from "@/composables/useJobs";
import { useMessage } from "@/composables";
import { useCreateTrigger } from "@/composables/useTriggers";
import { PageHeader } from "@/components/index";
import { to } from "@/router/registry";

const router = useRouter();
const message = useMessage();
const { data: definitions } = useJobDefinitions();
const create = useCreateTrigger();

const key = ref("");
const name = ref("");
const jobDefinitionId = ref(0);
const mode = ref("async");
const lastToken = ref("");
const copied = ref(false);

async function copyToken() {
  try {
    await navigator.clipboard.writeText(lastToken.value);
    copied.value = true;
    setTimeout(() => { copied.value = false; }, 2000);
  } catch { /* ignore */ }
}

async function handleCreate() {
  try {
    const result = await create.execute({
      key: key.value.trim(),
      name: name.value.trim(),
      job_definition_id: jobDefinitionId.value,
      mode: mode.value,
      default_input: {},
      enabled: true,
    });
    if (result && typeof result === "object" && "delivery_token" in result) {
      lastToken.value = result.delivery_token as string;
    }
    copied.value = false;
  } catch (e) {
    message.error(e instanceof Error ? e.message : "创建失败");
  }
}
</script>

<template>
  <div class="page-shell">
    <PageHeader
      title="创建触发器"
      subtitle="创建一个新的 Webhook 触发器"
      icon-bg="linear-gradient(135deg, rgba(234,179,8,0.12), rgba(202,138,4,0.12))"
      icon-color="var(--icon-yellow)"
      :back-to="to.triggers.list()"
      back-label="返回触发器列表"
    >
      <template #icon><icon-thunderbolt /></template>
    </PageHeader>

    <ui-card class="min-w-0">
      <ui-form layout="vertical">
        <ui-form-item label="键名">
          <ui-input v-model="key" placeholder="github-webhook" />
        </ui-form-item>
        <ui-form-item label="名称">
          <ui-input v-model="name" placeholder="GitHub Webhook" />
        </ui-form-item>
        <ui-form-item label="绑定任务定义">
          <ui-select v-model="jobDefinitionId" placeholder="选择任务定义">
            <ui-option :value="0">选择任务定义</ui-option>
            <ui-option v-for="item in definitions" :key="item.id" :value="item.id">{{ item.name }}</ui-option>
          </ui-select>
        </ui-form-item>
        <ui-form-item label="执行模式">
          <ui-select v-model="mode">
            <ui-option value="async">async（异步）</ui-option>
            <ui-option value="sync">sync（同步等待）</ui-option>
          </ui-select>
        </ui-form-item>

        <!-- Token display -->
        <div v-if="lastToken" class="mt-6 rounded-xl border p-5 border-slate-200 bg-slate-50 shadow-sm">
          <div class="mb-3 flex items-start justify-between gap-3">
            <div class="flex items-center gap-2 text-sm font-semibold text-emerald-800">
              <span class="inline-block flex-shrink-0 rounded-full bg-slate-400  h-2 w-2 [@media(prefers-reduced-motion:no-preference)]:[&.online]:bg-emerald-500 [@media(prefers-reduced-motion:no-preference)]:[&.online]:animate-[pulse-dot_2s_ease-in-out_infinite] [&.offline]:bg-red-500 [&.neutral]:bg-slate-400 online" />
              <span>创建成功！请保存您的 Delivery Token</span>
            </div>
            <ui-button size="mini" type="text" @click="copyToken">
              <template #icon><icon-copy /></template>
              {{ copied ? "已复制" : "复制" }}
            </ui-button>
          </div>
          <code class="block w-full overflow-auto rounded-xl border px-4 py-3 text-[13px] text-slate-900 border-slate-200 bg-white/[72%]">{{ lastToken }}</code>
          <p class="mt-3 text-xs text-emerald-700">注意：Token 只会显示一次，请妥善保管。</p>
        </div>

        <div class="mt-4 flex items-center justify-end gap-2 border-t border-slate-100 pt-4">
          <ui-button @click="router.push(to.triggers.list())">{{ lastToken ? "返回触发器列表" : "取消" }}</ui-button>
          <ui-button
            v-if="!lastToken"
            type="primary"
            :disabled="!key.trim() || !jobDefinitionId"
            :loading="create.loading.value"
            @click="handleCreate"
          >
            {{ create.loading.value ? "创建中…" : "创建触发器" }}
          </ui-button>
        </div>
      </ui-form>
    </ui-card>
  </div>
</template>
