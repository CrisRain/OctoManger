<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { Message } from "@/lib/feedback";
import {
  useEmailAccounts,
  useEmailMailboxes,
  useEmailMessages,
  useEmailMessage,
  usePreviewMailboxes,
  usePreviewLatestMessage,
} from "@/composables/useEmailAccounts";
import { PageHeader } from "@/components/index";
import { to } from "@/router/registry";

const route = useRoute();
const accountId = Number(route.params.id);

const { data: accounts, loading } = useEmailAccounts();
const account = computed(() => accounts.value.find((item) => item.id === accountId));

const mailboxPattern = ref("");
const selectedMailbox = ref("");
const selectedMessageId = ref("");
const previewOutput = ref("");
const previewLabel = ref("");
const previewGeneratedAt = ref("");
const readerTab = ref("body");

const {
  data: mailboxesData,
  loading: mailboxesLoading,
  error: mailboxesError,
  refresh: refreshMailboxes,
} = useEmailMailboxes(accountId, () => mailboxPattern.value);

const {
  data: messagesData,
  loading: messagesLoading,
  error: messagesError,
  refresh: refreshMessages,
} = useEmailMessages(accountId, () => selectedMailbox.value);

const {
  data: messageData,
  loading: messageLoading,
  error: messageError,
  refresh: refreshMessage,
} = useEmailMessage(accountId, () => selectedMessageId.value);

const previewMailboxes = usePreviewMailboxes();
const previewLatest = usePreviewLatestMessage();

const mailboxItems = computed(() => mailboxesData.value?.items ?? []);
const messageItems = computed(() => messagesData.value?.items ?? []);
const messageDetail = computed(() => messageData.value);
const selectedMailboxItem = computed(
  () => mailboxItems.value.find((item) => item.id === selectedMailbox.value) ?? null,
);
const selectedMessageSummary = computed(
  () => messageItems.value.find((item) => item.id === selectedMessageId.value) ?? null,
);
const selectedFlags = computed(() => {
  if (messageDetail.value?.flags?.length) return messageDetail.value.flags;
  return selectedMessageSummary.value?.flags ?? [];
});
const totalMessages = computed(() => messagesData.value?.total ?? messageItems.value.length);
const headerEntries = computed(() => Object.entries(messageDetail.value?.headers ?? {}));
const readerBody = computed(() => {
  const detail = messageDetail.value;
  if (!detail) return "";
  if (detail.text_body?.trim()) return detail.text_body.trim();
  if (detail.html_body?.trim()) return htmlToText(detail.html_body);
  return "No body content.";
});
const accountStatusText = computed(() => {
  switch (account.value?.status) {
    case "active":
      return "已连接";
    case "pending":
      return "待验证";
    case "inactive":
      return "已停用";
    default:
      return "未知状态";
  }
});
const accountStatusColor = computed(() => {
  switch (account.value?.status) {
    case "active":
      return "green";
    case "pending":
      return "orange";
    case "inactive":
      return "gray";
    default:
      return "gray";
  }
});

watch(selectedMailbox, () => {
  selectedMessageId.value = "";
  readerTab.value = "body";
  if (selectedMailbox.value) void refreshMessages();
});

watch(selectedMessageId, () => {
  readerTab.value = "body";
  if (selectedMessageId.value) void refreshMessage();
});

watch(mailboxItems, (items) => {
  if (!items.length) {
    selectedMailbox.value = "";
    return;
  }
  if (!items.some((item) => item.id === selectedMailbox.value)) {
    selectedMailbox.value = items[0].id;
  }
}, { immediate: true });

watch(messageItems, (items) => {
  if (!items.length) {
    selectedMessageId.value = "";
    return;
  }
  if (!items.some((item) => item.id === selectedMessageId.value)) {
    selectedMessageId.value = items[0].id;
  }
}, { immediate: true });

async function handleSearchMailboxes() {
  await refreshMailboxes();
}

async function handleRefreshWorkspace() {
  await refreshMailboxes();
  if (selectedMailbox.value) await refreshMessages();
  if (selectedMessageId.value) await refreshMessage();
}

function setPreviewPayload(label: string, payload: unknown) {
  previewLabel.value = label;
  previewGeneratedAt.value = new Date().toLocaleString("zh-CN");
  previewOutput.value = JSON.stringify(payload, null, 2);
  readerTab.value = "preview";
}

async function handlePreviewMailboxes() {
  if (!account.value) return;
  try {
    const result = await previewMailboxes.execute({
      config: account.value.config as Record<string, unknown>,
      pattern: mailboxPattern.value,
    });
    setPreviewPayload("文件夹快速预览", result);
  } catch (e) {
    Message.error(e instanceof Error ? e.message : "预览失败");
  }
}

async function handlePreviewLatest() {
  if (!account.value) return;
  try {
    const result = await previewLatest.execute({
      config: account.value.config as Record<string, unknown>,
      mailbox: selectedMailbox.value || undefined,
    });
    setPreviewPayload(
      selectedMailboxItem.value
        ? `${selectedMailboxItem.value.name} 的最新邮件预览`
        : "最新邮件预览",
      result,
    );
  } catch (e) {
    Message.error(e instanceof Error ? e.message : "预览失败");
  }
}

function formatDate(value?: string, withYear = false) {
  if (!value) return "未知时间";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  const options: Intl.DateTimeFormatOptions = withYear
    ? {
        year: "numeric",
        month: "2-digit",
        day: "2-digit",
        hour: "2-digit",
        minute: "2-digit",
      }
    : {
        month: "2-digit",
        day: "2-digit",
        hour: "2-digit",
        minute: "2-digit",
      };
  return date.toLocaleString("zh-CN", options);
}

function formatSize(size?: number) {
  if (typeof size !== "number" || Number.isNaN(size)) return "大小未知";
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${(size / (1024 * 1024)).toFixed(1)} MB`;
}

function isUnread(flags: string[] = []) {
  return !flags.some((flag) => {
    const normalized = flag.toLowerCase();
    return normalized.includes("seen") || normalized.includes("read");
  });
}

function formatFlag(flag: string) {
  return flag.replace(/^[\\]+/, "");
}

function htmlToText(html: string) {
  return html
    .replace(/<style[\s\S]*?<\/style>/gi, " ")
    .replace(/<script[\s\S]*?<\/script>/gi, " ")
    .replace(/<\/(p|div|li|tr|h1|h2|h3|h4|h5|h6|br)>/gi, "\n")
    .replace(/<[^>]+>/g, " ")
    .replace(/&nbsp;/gi, " ")
    .replace(/&amp;/gi, "&")
    .replace(/&lt;/gi, "<")
    .replace(/&gt;/gi, ">")
    .replace(/\r/g, "")
    .replace(/\n{3,}/g, "\n\n")
    .replace(/[ \t]{2,}/g, " ")
    .trim();
}
</script>

<template>
  <div class="page-container email-preview-page">
    <PageHeader
      title="预览邮箱数据"
      icon-bg="linear-gradient(135deg, rgba(234,88,12,0.12), rgba(249,115,22,0.12))"
      icon-color="var(--icon-orange)"
      :back-to="to.emailAccounts.list()"
      back-label="返回邮箱账号"
    >
      <template #icon><icon-email /></template>
      <template #subtitle>
        <span v-if="account">{{ account.address }}</span>
      </template>
      <template #actions>
        <div v-if="account" class="preview-header-actions">
          <ui-tag class="toolbar-tag">{{ account.provider || "outlook" }}</ui-tag>
          <ui-tag :color="accountStatusColor" class="toolbar-tag">{{ accountStatusText }}</ui-tag>
          <ui-button :loading="mailboxesLoading" @click="handleRefreshWorkspace">
            <template #icon><icon-refresh /></template>
            刷新邮箱
          </ui-button>
          <ui-button :loading="previewLatest.loading.value" type="primary" @click="handlePreviewLatest">
            <template #icon><icon-eye /></template>
            预览最新
          </ui-button>
        </div>
      </template>
    </PageHeader>

    <div v-if="loading" class="center-empty"><ui-spin :size="36" /></div>
    <div v-else-if="!account" class="center-empty"><p class="muted-copy">未找到该邮箱账号。</p></div>

    <div v-else class="mail-preview-page">
      <section class="mail-toolbar">
        <div class="mail-toolbar-main">
          <div class="mail-toolbar-icon">
            <icon-email />
          </div>
          <div class="mail-toolbar-eyebrow">Mailbox Workspace</div>
          <div class="mail-toolbar-title">{{ selectedMailboxItem?.name || "全部邮箱文件夹" }}</div>
          <p class="mail-toolbar-subtitle">
            已加载 {{ mailboxItems.length }} 个文件夹，当前文件夹共 {{ totalMessages }} 封邮件
          </p>
        </div>

        <div class="mail-toolbar-actions">
          <ui-button :loading="previewMailboxes.loading.value" @click="handlePreviewMailboxes">
            文件夹预览
          </ui-button>
          <ui-button :loading="messagesLoading" :disabled="!selectedMailbox" @click="refreshMessages">
            刷新列表
          </ui-button>
        </div>
      </section>

      <div class="mail-workspace">
        <aside class="mail-pane folders-pane">
          <div class="pane-header">
            <div class="pane-eyebrow">Folders</div>
            <h3 class="pane-title">邮箱文件夹</h3>
            <span class="pane-count">{{ mailboxItems.length }}</span>
          </div>

          <div class="pane-toolbar">
            <ui-input
              v-model="mailboxPattern"
              allow-clear
              placeholder="筛选文件夹，例如 *alert*"
              @press-enter="handleSearchMailboxes"
            >
              <template #prefix><icon-search /></template>
            </ui-input>
            <ui-button type="secondary" :loading="mailboxesLoading" @click="handleSearchMailboxes">
              筛选
            </ui-button>
          </div>

          <div class="pane-body folder-list">
            <div v-if="mailboxesLoading" class="pane-empty">
              <ui-spin />
              <span>正在读取文件夹…</span>
            </div>
            <div v-else-if="mailboxesError" class="pane-empty pane-empty--error">
              <p>{{ mailboxesError }}</p>
              <ui-button size="small" @click="handleSearchMailboxes">重试</ui-button>
            </div>
            <div v-else-if="!mailboxItems.length" class="pane-empty">
              <icon-folder />
              <span>{{ mailboxPattern ? "没有匹配的文件夹" : "暂无文件夹数据" }}</span>
            </div>
            <button
              v-for="item in mailboxItems"
              :key="item.id"
              type="button"
              class="folder-item"
              :class="{ 'folder-item--active': item.id === selectedMailbox }"
              @click="selectedMailbox = item.id"
            >
              <span class="folder-item-icon"><icon-folder /></span>
              <span class="folder-item-name">{{ item.name }}</span>
              <span class="folder-item-meta">
                {{ item.id === selectedMailbox ? "正在查看这个文件夹" : "点击查看邮件列表" }}
              </span>
            </button>
          </div>
        </aside>

        <section class="mail-pane messages-pane">
          <div class="pane-header">
            <div class="pane-eyebrow">Messages</div>
            <h3 class="pane-title">{{ selectedMailboxItem?.name || "邮件列表" }}</h3>
            <span class="pane-count">{{ totalMessages }}</span>
          </div>

          <div class="message-summary-bar">
            <span>{{ selectedMailboxItem ? `已打开 ${selectedMailboxItem.name}` : "先选择一个文件夹" }}</span>
            <span>{{ messageItems.length }} / {{ totalMessages }} 封已载入</span>
          </div>

          <div class="pane-body message-list">
            <div v-if="!selectedMailbox" class="pane-empty">
              <icon-email />
              <span>选择左侧文件夹后即可查看邮件</span>
            </div>
            <div v-else-if="messagesLoading" class="pane-empty">
              <ui-spin />
              <span>正在拉取邮件列表…</span>
            </div>
            <div v-else-if="messagesError" class="pane-empty pane-empty--error">
              <p>{{ messagesError }}</p>
              <ui-button size="small" @click="refreshMessages">重试</ui-button>
            </div>
            <div v-else-if="!messageItems.length" class="pane-empty">
              <icon-email />
              <span>这个文件夹当前没有邮件</span>
            </div>
            <button
              v-for="item in messageItems"
              :key="item.id"
              type="button"
              class="message-row"
              :class="{
                'message-row--active': item.id === selectedMessageId,
                'message-row--unread': isUnread(item.flags),
              }"
              @click="selectedMessageId = item.id"
            >
              <span class="message-row-indicator" />
              <span class="message-row-from">{{ item.from || "未知发件人" }}</span>
              <span class="message-row-date">{{ formatDate(item.date) }}</span>
              <div class="message-row-subject">{{ item.subject || "(no subject)" }}</div>
              <div class="message-row-meta">
                <span class="message-row-to">{{ item.to || "未提供收件人" }}</span>
                <span>{{ formatSize(item.size) }}</span>
              </div>
            </button>
          </div>
        </section>

        <section class="mail-pane reader-pane">
          <div class="reader-header" :class="{ 'reader-header--empty': !selectedMessageSummary && !messageDetail }">
            <template v-if="selectedMessageSummary || messageDetail">
              <div class="reader-eyebrow-row">
                <span class="reader-mailbox-pill">
                  <icon-folder />
                  {{ selectedMailboxItem?.name || "当前文件夹" }}
                </span>
                <span class="reader-date">
                  {{ formatDate(messageDetail?.date || selectedMessageSummary?.date, true) }}
                </span>
              </div>
              <h2 class="reader-subject">
                {{ messageDetail?.subject || selectedMessageSummary?.subject || "(no subject)" }}
              </h2>

              <div class="reader-meta-grid">
                <div class="reader-meta-card">
                  <span class="reader-meta-label">发件人</span>
                  <span class="reader-meta-value">{{ messageDetail?.from || selectedMessageSummary?.from || "未知" }}</span>
                </div>
                <div class="reader-meta-card">
                  <span class="reader-meta-label">收件人</span>
                  <span class="reader-meta-value">{{ messageDetail?.to || selectedMessageSummary?.to || "未知" }}</span>
                </div>
                <div class="reader-meta-card">
                  <span class="reader-meta-label">抄送</span>
                  <span class="reader-meta-value">{{ messageDetail?.cc || "无" }}</span>
                </div>
                <div class="reader-meta-card">
                  <span class="reader-meta-label">大小</span>
                  <span class="reader-meta-value">{{ formatSize(messageDetail?.size || selectedMessageSummary?.size) }}</span>
                </div>
              </div>

              <div v-if="selectedFlags.length" class="reader-flags">
                <ui-tag v-for="flag in selectedFlags" :key="flag" class="reader-flag">
                  {{ formatFlag(flag) }}
                </ui-tag>
              </div>
            </template>

            <template v-else>
              <div class="pane-empty reader-empty">
                <icon-eye />
                <span>从中间邮件列表选择一封邮件开始阅读</span>
              </div>
            </template>
          </div>

          <ui-tabs v-model:active-key="readerTab" class="reader-tabs">
            <ui-tab-pane key="body" title="阅读视图">
              <div class="reader-tab-panel">
                <div v-if="messageLoading" class="pane-empty">
                  <ui-spin />
                  <span>正在加载邮件正文…</span>
                </div>
                <div v-else-if="messageError" class="pane-empty pane-empty--error">
                  <p>{{ messageError }}</p>
                  <ui-button size="small" @click="refreshMessage">重新拉取</ui-button>
                </div>
                <div v-else-if="messageDetail" class="reader-paper">
                  <pre class="reader-paper-content">{{ readerBody }}</pre>
                </div>
                <div v-else class="pane-empty">
                  <icon-email />
                  <span>选中邮件后可在这里查看正文</span>
                </div>
              </div>
            </ui-tab-pane>

            <ui-tab-pane key="html" title="HTML">
              <div class="reader-tab-panel">
                <pre v-if="messageDetail?.html_body" class="code-view">{{ messageDetail.html_body }}</pre>
                <div v-else class="pane-empty">
                  <icon-file />
                  <span>这封邮件没有 HTML 内容</span>
                </div>
              </div>
            </ui-tab-pane>

            <ui-tab-pane key="headers" title="Headers">
              <div class="reader-tab-panel">
                <div v-if="headerEntries.length" class="headers-list">
                  <div v-for="[key, value] in headerEntries" :key="key" class="header-row">
                    <span class="header-key">{{ key }}</span>
                    <span class="header-value">{{ value }}</span>
                  </div>
                </div>
                <div v-else class="pane-empty">
                  <icon-info-circle />
                  <span>暂无可展示的邮件头</span>
                </div>
              </div>
            </ui-tab-pane>

            <ui-tab-pane key="preview" title="快速预览">
              <div class="reader-tab-panel">
                <div v-if="previewOutput" class="preview-output-card">
                  <div class="preview-output-head">
                    <div class="pane-eyebrow">Developer Snapshot</div>
                    <h4 class="preview-output-title">{{ previewLabel || "快速预览输出" }}</h4>
                    <span class="preview-output-time">{{ previewGeneratedAt }}</span>
                  </div>
                  <pre class="code-view">{{ previewOutput }}</pre>
                </div>
                <div v-else class="pane-empty">
                  <icon-code-block />
                  <span>点击顶部的预览按钮后，这里会显示原始输出</span>
                </div>
              </div>
            </ui-tab-pane>
          </ui-tabs>
        </section>
      </div>
    </div>
  </div>
</template>

