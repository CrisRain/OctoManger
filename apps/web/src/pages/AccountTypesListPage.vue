<script setup lang="ts">
import { computed, ref } from "vue";
import { useRouter } from "vue-router";
import { IconLayers, IconPlus, IconEdit, IconDelete } from "@/lib/icons";

import { PageHeader, SmartListBar, DetailDrawer } from "@/components/index";
import { useAccountTypes } from "@/composables/useAccountTypes";
import { useMessage, useConfirm, useErrorHandler } from "@/composables";
import { to } from "@/router/registry";

const router = useRouter();
const message = useMessage();
const confirm = useConfirm();
const { withErrorHandler } = useErrorHandler();

const { data: items, loading, refresh } = useAccountTypes();

// 分类筛选
const categoryFilter = ref<string>();
const searchKeyword = ref("");
const categoryOptions = [
  { label: '全部', value: '' },
  { label: '通用', value: 'generic' },
  { label: '邮箱', value: 'email' },
  { label: '系统', value: 'system' },
];

// 详情抽屉
const drawerVisible = ref(false);
const selectedItem = ref<any>(null);
const drawerLoading = ref(false);

const filteredItems = computed(() => {
  let result = items.value;
  if (categoryFilter.value) {
    result = result.filter(item => item.category === categoryFilter.value);
  }
  if (searchKeyword.value) {
    const keyword = searchKeyword.value.toLowerCase();
    result = result.filter(item =>
      item.name?.toLowerCase().includes(keyword) ||
      item.key?.toLowerCase().includes(keyword)
    );
  }
  return result;
});

// 打开详情
function openDetail(item: any) {
  selectedItem.value = item;
  drawerVisible.value = true;
}

// 关闭详情
function closeDetail() {
  drawerVisible.value = false;
  selectedItem.value = null;
}

// 快速操作
async function handleQuickAction(key: string, item: any) {
  switch (key) {
    case "view":
      openDetail(item);
      break;
    case "edit":
      router.push(to.accountTypes.edit(item.key));
      break;
    case "delete":
      await deleteAccountType(item);
      break;
  }
}

// 删除账号类型
async function deleteAccountType(item: any) {
  const confirmed = await confirm.confirmDanger(
    `删除账号类型"${item.name}"后，所有使用该类型的账号将无法正常使用。`,
    "确认删除"
  );

  if (!confirmed) return;

  await withErrorHandler(
    async () => {
      // TODO: 调用删除API
      message.success(`已删除账号类型: ${item.name}`);
      await refresh();
    },
    { action: "删除", showSuccess: true }
  );
}

// 批量删除
async function handleBatchDelete(items: any[]) {
  const confirmed = await confirm.confirm(
    `确定要删除选中的 ${items.length} 个账号类型吗？`
  );
  if (!confirmed) return;

  await withErrorHandler(
    async () => {
      // TODO: 调用批量删除API
      message.success(`已删除 ${items.length} 个账号类型`);
      await refresh();
    },
    { action: "批量删除", showSuccess: true }
  );
}
</script>

<template>
  <div class="page-shell smart-list-page account-types-list-page">
    <PageHeader
      title="账号类型"
      subtitle="定义系统中使用的账号类型及其字段结构"
      icon-bg="linear-gradient(135deg, rgba(2,132,199,0.12), rgba(14,165,233,0.12))"
      icon-color="#0284c7"
    >
      <template #icon><icon-layers /></template>
      <template #actions>
        <ui-button type="primary" @click="router.push(to.accountTypes.create())">
          <template #icon><icon-plus /></template>
          新建类型
        </ui-button>
      </template>
    </PageHeader>

    <!-- 智能工具栏 -->
    <SmartListBar
      :data="filteredItems"
      :loading="loading"
      v-model:search="searchKeyword"
      @refresh="refresh"
      @batch-delete="handleBatchDelete"
    >
      <template #filters>
        <div class="flex flex-wrap items-center gap-2">
          <span class="text-xs font-medium text-slate-500">分类：</span>
          <div class="flex flex-wrap items-center gap-1">
            <button type="button"
              v-for="option in categoryOptions"
              :key="option.value"
              class="filter-chip"
              :class="{ active: categoryFilter === option.value }"
              @click="categoryFilter = option.value"
            >
              {{ option.label }}
            </button>
          </div>
        </div>
      </template>
    </SmartListBar>

    <!-- 数据卡片网格 -->
    <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
      <ui-card
        v-for="item in filteredItems"
        :key="item.key"
        class="cursor-pointer"
        hoverable
        @click="openDetail(item)"
      >
        <div class="mb-3 flex items-center gap-3">
          <div
            class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-xl text-sm"
            :class="{
              'bg-blue-50 text-blue-500': item.category === 'generic',
              'bg-orange-50 text-orange-500': item.category === 'email',
              'bg-slate-50 text-slate-500': item.category === 'system'
            }"
          >
            <icon-layers />
          </div>
          <div class="min-w-0 flex-1">
            <code class="block truncate font-mono text-[11px] text-slate-400">{{ item.key }}</code>
            <span class="inline-flex items-center rounded-full border border-slate-200 bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-600">
              {{ item.category === 'generic' ? '通用' : item.category === 'email' ? '邮箱' : '系统' }}
            </span>
          </div>
        </div>
        <div class="mb-3 text-sm font-semibold text-slate-900">{{ item.name }}</div>
        <div class="flex items-center justify-between gap-2 border-t border-slate-100 pt-3">
          <span class="text-xs text-slate-500">{{ Object.keys(item.schema || {}).length }} 个字段</span>
          <div class="flex items-center gap-1" @click.stop>
            <ui-button size="small" type="text" @click="handleQuickAction('edit', item)">
              <template #icon><icon-edit /></template>
              编辑
            </ui-button>
            <ui-button
              size="small"
              type="text"
              status="danger"
              @click="handleQuickAction('delete', item)"
            >
              <template #icon><icon-delete /></template>
              删除
            </ui-button>
          </div>
        </div>
      </ui-card>

      <!-- 空状态 -->
      <ui-card v-if="!loading && !filteredItems.length" class="col-span-full empty-state-block">
        <ui-empty description="暂无账号类型">
          <ui-button type="primary" @click="router.push(to.accountTypes.create())">
            创建第一个账号类型
          </ui-button>
        </ui-empty>
      </ui-card>
    </div>

    <!-- 详情抽屉 -->
    <DetailDrawer
      v-model:open="drawerVisible"
      :title="selectedItem?.name"
      :loading="drawerLoading"
      @close="closeDetail"
    >
      <template v-if="selectedItem" #detail>
        <div class="rounded-xl border p-4 border-slate-200 bg-white/[56%]">
          <div class="flex items-start justify-between gap-4 border-b border-slate-100 py-3 first:pt-0 last:border-b-0 last:pb-0 max-md:flex-col max-md:items-start">
            <span class="text-xs font-semibold uppercase tracking-[0.08em] text-slate-500">类型标识</span>
            <code class="text-sm font-medium text-slate-900 mono">{{ selectedItem.key }}</code>
          </div>
          <div class="flex items-start justify-between gap-4 border-b border-slate-100 py-3 first:pt-0 last:border-b-0 last:pb-0 max-md:flex-col max-md:items-start">
            <span class="text-xs font-semibold uppercase tracking-[0.08em] text-slate-500">分类</span>
            <span class="text-sm font-medium text-slate-900">
              {{ selectedItem.category === 'generic' ? '通用' : selectedItem.category === 'email' ? '邮箱' : '系统' }}
            </span>
          </div>
          <div class="flex items-start justify-between gap-4 border-b border-slate-100 py-3 first:pt-0 last:border-b-0 last:pb-0 max-md:flex-col max-md:items-start">
            <span class="text-xs font-semibold uppercase tracking-[0.08em] text-slate-500">字段数量</span>
            <span class="text-sm font-medium text-slate-900">{{ Object.keys(selectedItem.schema || {}).length }} 个</span>
          </div>
        </div>

        <div v-if="selectedItem.description" class="rounded-xl border p-4 border-slate-200 bg-white/[56%]">
          <h4 class="text-[15px] font-semibold text-slate-900">描述</h4>
          <p class="mt-2 text-sm leading-6 text-slate-600">{{ selectedItem.description }}</p>
        </div>

        <div v-if="selectedItem.schema && Object.keys(selectedItem.schema).length" class="rounded-xl border p-4 border-slate-200 bg-white/[56%]">
          <h4 class="text-[15px] font-semibold text-slate-900">字段定义</h4>
          <div class="mt-3 flex flex-col gap-2">
            <div v-for="(field, key) in selectedItem.schema" :key="key" class="rounded-lg border border-slate-200 bg-slate-50/70 p-3">
              <div class="flex flex-wrap items-center gap-2">
                <code class="text-xs font-mono font-semibold text-slate-700">{{ key }}</code>
                <span class="inline-flex items-center rounded-full border border-slate-200 bg-white px-2 py-0.5 text-[11px] text-slate-500">{{ field.type || 'string' }}</span>
              </div>
              <p v-if="field.description" class="mt-1.5 text-xs leading-relaxed text-slate-500">{{ field.description }}</p>
            </div>
          </div>
        </div>
      </template>

      <template #footer>
        <ui-button @click="closeDetail">关闭</ui-button>
        <ui-button
          type="primary"
          @click="selectedItem && router.push(to.accountTypes.edit(selectedItem.key))"
        >
          <template #icon><icon-edit /></template>
          编辑
        </ui-button>
      </template>
    </DetailDrawer>
  </div>
</template>
