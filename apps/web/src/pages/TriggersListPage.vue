<script setup lang="ts">
import { ref } from "vue";
import { useRouter } from "vue-router";
import { Message } from "@/lib/feedback";
import { useTriggers, useDeleteTrigger, useFireTrigger } from "@/composables/useTriggers";
import { PageHeader } from "@/components/index";
import DataTable from "@/components/DataTable.vue";
import { to } from "@/router/registry";

const router = useRouter();
const { data: items, loading, refresh } = useTriggers();
const deleteTrigger = useDeleteTrigger();
const fireTrigger = useFireTrigger();

const payloadText = ref("{}");
const resultText = ref("");

function safePayload(): Record<string, unknown> {
  try {
    const p = JSON.parse(payloadText.value);
    if (p && typeof p === "object" && !Array.isArray(p)) return p as Record<string, unknown>;
    return {};
  } catch { return {}; }
}

async function handleFire(id: number) {
  try {
    const result = await fireTrigger.execute(id, safePayload());
    resultText.value = JSON.stringify(result, null, 2);
    Message.success("触发成功");
  } catch (e) {
    Message.error(e instanceof Error ? e.message : "触发失败");
  }
}

async function handleDelete(id: number) {
  try {
    await deleteTrigger.execute(id);
    Message.success("已删除");
    await refresh();
  } catch (e) {
    Message.error(e instanceof Error ? e.message : "删除失败");
  }
}
</script>

<template>
  <div class="page-container list-page triggers-list-page">
    <PageHeader
      title="触发器"
      subtitle="通过 HTTP 调用触发任务执行，支持 Webhook 接入。"
      icon-bg="linear-gradient(135deg, rgba(234,179,8,0.12), rgba(202,138,4,0.12))"
      icon-color="var(--icon-yellow)"
    >
      <template #icon><icon-thunderbolt /></template>
      <template #actions>
        <ui-button type="primary" class="create-btn" @click="router.push(to.triggers.create())">
          <template #icon><icon-plus /></template>
          新建触发器
        </ui-button>
      </template>
    </PageHeader>

    <ui-card class="data-grid-card triggers-table-card">
      <DataTable
        :data="items"
        :loading="loading"
        :empty="{
          title: '暂无触发器',
          description: '点击右上角新建。',
          actionText: '新建触发器',
        }"
        @empty-action="router.push(to.triggers.create())"
      >
        <template #columns>
          <ui-table-column title="名称 / Key">
            <template #cell="{ record }">
              <div class="identifier-text">{{ record.name }}</div>
              <code class="key-badge trigger-key-badge">{{ record.key }}</code>
            </template>
          </ui-table-column>

          <ui-table-column title="执行模式">
            <template #cell="{ record }">
              <span class="mode-tag" :class="record.mode === 'sync' ? 'sync' : 'async'">
                {{ record.mode }}
              </span>
            </template>
          </ui-table-column>

          <ui-table-column title="Token 前缀">
            <template #cell="{ record }">
              <span class="text-muted mono token-prefix">{{ record.token_prefix }}••••••</span>
            </template>
          </ui-table-column>

          <ui-table-column title="操作" :width="160" align="right">
            <template #cell="{ record }">
              <div class="action-cell">
                <ui-button
                  size="mini"
                  type="text"
                  class="action-btn-text fire-btn"
                  :loading="fireTrigger.loading.value"
                  @click="handleFire(record.id)"
                >
                  <template #icon><icon-thunderbolt /></template>
                  测试
                </ui-button>
                <ui-button
                  size="mini"
                  type="text"
                  class="action-btn-text"
                  @click="router.push(to.triggers.edit(record.id))"
                >编辑</ui-button>
                <ui-popconfirm content="确定要删除此触发器吗？" position="left" type="warning" @ok="handleDelete(record.id)">
                  <ui-button size="mini" type="text" class="action-btn-text action-btn--danger" :loading="deleteTrigger.loading.value">
                    删除
                  </ui-button>
                </ui-popconfirm>
              </div>
            </template>
          </ui-table-column>
        </template>
      </DataTable>
    </ui-card>

    <!-- Test panel (only shown when there are triggers) -->
    <ui-card v-if="items.length" class="premium-card config-card">
      <template #title>
        <div class="card-title-row">
          <div class="title-icon icon-amber"><icon-thunderbolt /></div>
          测试发送
        </div>
      </template>
      <ui-form layout="vertical">
        <ui-form-item label="请求参数（JSON）">
          <ui-textarea
            v-model="payloadText"
            placeholder="{}"
            :auto-size="{ minRows: 3 }"
            class="mono-field"
          />
          <div class="field-hint">点击列表中的「测试」按钮时，此参数会一起发送。</div>
        </ui-form-item>
        <ui-form-item v-if="resultText" label="返回结果">
          <ui-textarea :model-value="resultText" :auto-size="{ minRows: 4 }" readonly class="mono-field" />
        </ui-form-item>
      </ui-form>
    </ui-card>
  </div>
</template>
