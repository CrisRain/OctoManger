<script setup lang="ts">
import { useRouter } from "vue-router";
import {
  IconEmail, IconImport, IconEye, IconEdit, IconDelete, IconPlus
} from "@/lib/icons";

import { PageHeader, SmartListBar, RowActionsMenu } from "@/components/index";
import { useEmailAccountsList } from "@/composables/useEmailAccountsList";
import { to } from "@/router/registry";

const router = useRouter();

const columns = [
  { title: "邮箱地址", dataIndex: "address", width: 300, slotName: "address" },
  { title: "服务商", dataIndex: "provider", slotName: "provider" },
  { title: "状态", dataIndex: "status", slotName: "status" },
  { title: "操作", width: 80, align: "right", slotName: "actions" },
];

const {
  loading,
  refresh,
  filteredAccounts,
  providers,
  statusFilter,
  providerFilter,
  searchKeyword,
  showBulkImport,
  bulkText,
  bulkResult,
  bulkImportLoading,
  openBulkImport,
  handleBulkImport,
  handleQuickAction,
  handleBatchDelete,
  handleBatchExport,
} = useEmailAccountsList();
</script>

<template>
  <div class="page-container email-accounts-list-page">
    <PageHeader
      title="邮箱账号"
      subtitle="管理所有已接入的邮箱账号"
      icon-bg="linear-gradient(135deg, rgba(234,88,12,0.12), rgba(249,115,22,0.12))"
      icon-color="var(--icon-orange)"
    >
      <template #icon><icon-email /></template>
      <template #actions>
        <ui-button @click="openBulkImport">
          <template #icon><icon-import /></template>
          批量导入
        </ui-button>
        <ui-button type="primary" @click="router.push(to.emailAccounts.create())">
          <template #icon><icon-plus /></template>
          添加邮箱
        </ui-button>
      </template>
    </PageHeader>

    <!-- 智能工具栏 -->
    <SmartListBar
      :data="filteredAccounts"
      :loading="loading"
      v-model:search="searchKeyword"
      @refresh="refresh"
      @batch-delete="handleBatchDelete"
      @batch-export="handleBatchExport"
    >
      <template #filters>
        <!-- 状态筛选 -->
        <div class="filter-group">
          <span class="filter-label">状态：</span>
          <div class="filter-options">
            <button type="button"
              class="filter-option"
              :class="{ 'filter-option--active': !statusFilter }"
              @click="statusFilter = ''"
            >
              全部
            </button>
            <button type="button"
              class="filter-option"
              :class="{ 'filter-option--active': statusFilter === 'active' }"
              @click="statusFilter = 'active'"
            >
              已激活
            </button>
            <button type="button"
              class="filter-option"
              :class="{ 'filter-option--active': statusFilter === 'pending' }"
              @click="statusFilter = 'pending'"
            >
              待验证
            </button>
            <button type="button"
              class="filter-option"
              :class="{ 'filter-option--active': statusFilter === 'inactive' }"
              @click="statusFilter = 'inactive'"
            >
              已停用
            </button>
          </div>
        </div>

        <!-- 服务商筛选 -->
        <div class="filter-group" v-if="providers.length">
          <span class="filter-label">服务商：</span>
          <div class="filter-options">
            <button type="button"
              class="filter-option"
              :class="{ 'filter-option--active': !providerFilter }"
              @click="providerFilter = ''"
            >
              全部
            </button>
            <button type="button"
              v-for="provider in providers"
              :key="provider"
              class="filter-option"
              :class="{ 'filter-option--active': providerFilter === provider }"
              @click="providerFilter = provider"
            >
              {{ provider }}
            </button>
          </div>
        </div>
      </template>
    </SmartListBar>

    <!-- 数据表格 -->
    <ui-card class="data-grid-card table-desktop-only">
      <ui-table
        class="premium-table"
        :data="filteredAccounts"
        :loading="loading"
        :columns="columns"
        :pagination="{
          showTotal: true,
          pageSizeOptions: [10, 20, 50],
          defaultPageSize: 20,
        }"
        :bordered="false"
        row-key="id"
        :row-selection="{ type: 'checkbox' }"
      >
        <!-- 邮箱地址 -->
        <template #address="{ record }">
          <div class="identifier-cell">
            <div class="icon-box icon-orange">
              <icon-email />
            </div>
            <span class="identifier-text">{{ record.address }}</span>
          </div>
        </template>

        <!-- 服务商 -->
        <template #provider="{ record }">
          <code class="action-tag">{{ record.provider }}</code>
        </template>

        <!-- 状态 -->
        <template #status="{ record }">
          <span
            class="status-badge"
            :class="{
              'status-badge--active': record.status === 'active',
              'status-badge--pending': record.status === 'pending',
              'status-badge--inactive': record.status === 'inactive',
            }"
          >
            {{ record.status === 'active' ? '已激活' : record.status === 'pending' ? '待验证' : '已停用' }}
          </span>
        </template>

        <!-- 快速操作 -->
        <template #actions="{ record }">
          <RowActionsMenu
            :item="record"
            :actions="[
              { key: 'view', label: '预览邮件', icon: IconEye },
              { key: 'edit', label: '配置', icon: IconEdit },
              { key: 'delete-divider', divider: true },
              { key: 'delete', label: '删除', icon: IconDelete, danger: true },
            ]"
            @action="handleQuickAction"
          />
        </template>

        <!-- 空状态 -->
        <template #empty>
          <ui-empty description="暂无邮箱账号">
            <ui-button type="primary" @click="router.push(to.emailAccounts.create())">
              添加第一个邮箱
            </ui-button>
          </ui-empty>
        </template>
      </ui-table>
    </ui-card>

    <div class="mobile-list">
      <ui-card
        v-for="record in filteredAccounts"
        :key="record.id"
        class="mobile-list-card"
      >
        <div class="mobile-list-header">
          <div class="icon-box icon-orange">
            <icon-email />
          </div>
          <div class="mobile-list-title-group">
            <div class="mobile-list-title">{{ record.address }}</div>
            <div class="mobile-list-subtitle">
              <code class="action-tag">{{ record.provider }}</code>
            </div>
          </div>
          <RowActionsMenu
            :item="record"
            :actions="[
              { key: 'view', label: '预览邮件', icon: IconEye },
              { key: 'edit', label: '配置', icon: IconEdit },
              { key: 'delete-divider', divider: true },
              { key: 'delete', label: '删除', icon: IconDelete, danger: true },
            ]"
            @action="handleQuickAction"
          />
        </div>

        <div class="mobile-list-meta">
          <div class="mobile-list-meta-row">
            <span class="mobile-list-label">状态</span>
            <div class="mobile-list-value">
              <span
                class="status-badge"
                :class="{
                  'status-badge--active': record.status === 'active',
                  'status-badge--pending': record.status === 'pending',
                  'status-badge--inactive': record.status === 'inactive',
                }"
              >
                {{ record.status === 'active' ? '已激活' : record.status === 'pending' ? '待验证' : '已停用' }}
              </span>
            </div>
          </div>
        </div>
      </ui-card>

      <ui-card
        v-if="!loading && !filteredAccounts.length"
        class="mobile-list-card mobile-list-empty-card"
      >
        <ui-empty description="暂无邮箱账号">
          <ui-button type="primary" @click="router.push(to.emailAccounts.create())">
            添加第一个邮箱
          </ui-button>
        </ui-empty>
      </ui-card>
    </div>

    <!-- 批量导入对话框 -->
    <ui-modal
      v-model:visible="showBulkImport"
      title="批量导入邮箱"
      :width="640"
      :footer="bulkResult === null"
      ok-text="开始导入"
      :ok-loading="bulkImportLoading"
      @ok="handleBulkImport"
    >
      <div v-if="bulkResult === null">
        <p class="bulk-hint">
          每行一条，格式：<code>邮箱----密码----clientid----refresh_token</code>
        </p>
        <ui-textarea
          v-model="bulkText"
          :auto-size="{ minRows: 8, maxRows: 16 }"
          placeholder="user@example.com----password----xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx----0.AXXXXX..."
          class="mono-textarea"
        />
      </div>
      <div v-else>
        <div class="bulk-summary">
          <span>共 {{ bulkResult.length }} 条</span>
          <span class="success-count">成功 {{ bulkResult.filter(r => r.ok).length }}</span>
          <span class="error-count">失败 {{ bulkResult.filter(r => !r.ok).length }}</span>
        </div>
        <div class="bulk-result-list">
          <div v-for="(item, i) in bulkResult" :key="i" class="bulk-result-item" :class="item.ok ? 'result-ok' : 'result-fail'">
            <span class="result-dot" :class="item.ok ? 'online' : 'offline'" />
            <span class="result-address">{{ item.address || item.line }}</span>
            <span v-if="!item.ok" class="result-error">{{ item.error }}</span>
          </div>
        </div>
        <div class="bulk-footer-actions">
          <ui-button @click="() => { bulkResult = null; bulkText = ''; }">继续导入</ui-button>
          <ui-button type="primary" @click="showBulkImport = false">关闭</ui-button>
        </div>
      </div>
    </ui-modal>
  </div>
</template>
