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
  <div class="page-container smart-list-page account-types-list-page">
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
        <div class="filter-group">
          <span class="filter-label">分类：</span>
          <div class="filter-options">
            <button type="button"
              v-for="option in categoryOptions"
              :key="option.value"
              class="filter-option"
              :class="{ 'filter-option--active': categoryFilter === option.value }"
              @click="categoryFilter = option.value"
            >
              {{ option.label }}
            </button>
          </div>
        </div>
      </template>
    </SmartListBar>

    <!-- 数据卡片网格 -->
    <div class="types-grid">
      <ui-card
        v-for="item in filteredItems"
        :key="item.key"
        class="type-card"
        hoverable
        @click="openDetail(item)"
      >
        <div class="type-header">
          <div class="type-icon" :class="`type-icon--${item.category}`">
            <icon-layers />
          </div>
          <div class="type-info">
            <code class="type-key">{{ item.key }}</code>
            <span class="type-count">
              {{ item.category === 'generic' ? '通用' : item.category === 'email' ? '邮箱' : '系统' }}
            </span>
          </div>
        </div>
        <div class="type-name">{{ item.name }}</div>
        <div class="type-meta">
          <span class="type-field-count">{{ Object.keys(item.schema || {}).length }} 个字段</span>
        </div>

        <!-- 快速操作 -->
        <div class="type-actions" @click.stop>
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
      </ui-card>

      <!-- 空状态 -->
      <ui-card v-if="!loading && !filteredItems.length" class="empty-card">
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
        <div class="detail-section">
          <div class="detail-row">
            <span class="detail-label">类型标识</span>
            <code class="detail-value mono">{{ selectedItem.key }}</code>
          </div>
          <div class="detail-row">
            <span class="detail-label">分类</span>
            <span class="detail-value">
              {{ selectedItem.category === 'generic' ? '通用' : selectedItem.category === 'email' ? '邮箱' : '系统' }}
            </span>
          </div>
          <div class="detail-row">
            <span class="detail-label">字段数量</span>
            <span class="detail-value">{{ Object.keys(selectedItem.schema || {}).length }} 个</span>
          </div>
        </div>

        <div v-if="selectedItem.description" class="detail-section">
          <h4 class="section-title">描述</h4>
          <p class="section-text">{{ selectedItem.description }}</p>
        </div>

        <div v-if="selectedItem.schema && Object.keys(selectedItem.schema).length" class="detail-section">
          <h4 class="section-title">字段定义</h4>
          <div class="schema-list">
            <div v-for="(field, key) in selectedItem.schema" :key="key" class="schema-item">
              <div class="schema-header">
                <code class="schema-key">{{ key }}</code>
                <span class="schema-type">{{ field.type || 'string' }}</span>
              </div>
              <p v-if="field.description" class="schema-desc">{{ field.description }}</p>
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
