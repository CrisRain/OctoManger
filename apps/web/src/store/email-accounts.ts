import { defineStore } from "pinia";
import { ref } from "vue";
import {
  bulkImportEmailAccounts,
  buildOutlookAuthorizeURL,
  createEmailAccount,
  deleteEmailAccount,
  exchangeOutlookCode,
  getEmailMessage,
  listEmailAccounts,
  listEmailMailboxes,
  listEmailMessages,
  patchEmailAccount,
  previewEmailMailboxes,
  previewLatestEmailMessage,
} from "@/api";
import type {
  EmailAccount,
  EmailAccountCreateInput,
  EmailAccountPatchInput,
  EmailBulkImportResult,
  EmailLatestMessageResult,
  EmailMailboxListResult,
  EmailMessageDetail,
  EmailMessageListResult,
  EmailPreviewInput,
  OutlookAuthorizeURLResult,
  OutlookExchangeCodeInput,
} from "@/types";

type EmailAccountsResponseLike = {
  items?: unknown;
  data?: unknown;
};

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function toEmailAccount(item: unknown): EmailAccount | null {
  if (!isRecord(item)) {
    return null;
  }
  const id = item.id;
  const address = item.address;
  const provider = item.provider;
  const status = item.status;
  const config = item.config;
  if (typeof id !== "number" || typeof address !== "string") {
    return null;
  }
  return {
    id,
    address,
    provider: typeof provider === "string" ? provider : "",
    status: typeof status === "string" ? status : "",
    config: isRecord(config) ? config : {},
  };
}

function normalizeEmailAccountsResponse(payload: unknown): EmailAccount[] {
  const source = payload as EmailAccountsResponseLike;
  const firstLevelItems = Array.isArray(source?.items) ? source.items : null;
  if (firstLevelItems) {
    return firstLevelItems.map(toEmailAccount).filter((item): item is EmailAccount => item !== null);
  }
  if (Array.isArray(source?.data)) {
    return source.data.map(toEmailAccount).filter((item): item is EmailAccount => item !== null);
  }
  if (isRecord(source?.data) && Array.isArray((source.data as EmailAccountsResponseLike).items)) {
    const nestedItems = (source.data as EmailAccountsResponseLike).items as unknown[];
    return nestedItems.map(toEmailAccount).filter((item): item is EmailAccount => item !== null);
  }
  return [];
}

export const useEmailAccountsStore = defineStore("emailAccounts", () => {
  const emailAccounts = ref<EmailAccount[]>([]);
  const mailboxesByAccount = ref<Record<number, EmailMailboxListResult>>({});
  const messagesByKey = ref<Record<string, EmailMessageListResult>>({});
  const messageDetailsByKey = ref<Record<string, EmailMessageDetail>>({});
  const loading = ref(false);
  const error = ref<string | null>(null);

  async function fetchEmailAccounts() {
    loading.value = true;
    error.value = null;
    try {
      const response = await listEmailAccounts();
      emailAccounts.value = normalizeEmailAccountsResponse(response);
    } catch (e) {
      error.value = e instanceof Error ? e.message : "请求失败";
    } finally {
      loading.value = false;
    }
  }

  async function bulkImport(lines: string[]): Promise<EmailBulkImportResult> {
    return bulkImportEmailAccounts(lines);
  }

  async function create(payload: EmailAccountCreateInput) {
    const result = await createEmailAccount(payload);
    emailAccounts.value = [result, ...emailAccounts.value];
    return result;
  }

  async function update(id: number, payload: EmailAccountPatchInput) {
    const result = await patchEmailAccount(id, payload);
    emailAccounts.value = emailAccounts.value.map((item) => (item.id === id ? result : item));
    return result;
  }

  async function remove(id: number) {
    await deleteEmailAccount(id);
    emailAccounts.value = emailAccounts.value.filter((item) => item.id !== id);
  }

  async function buildAuthorizeUrl(id: number): Promise<OutlookAuthorizeURLResult> {
    return buildOutlookAuthorizeURL(id);
  }

  async function exchangeCode(id: number, payload: OutlookExchangeCodeInput) {
    return exchangeOutlookCode(id, payload);
  }

  async function fetchMailboxes(id: number, pattern?: string): Promise<EmailMailboxListResult> {
    const result = await listEmailMailboxes(id, pattern);
    mailboxesByAccount.value = { ...mailboxesByAccount.value, [id]: result };
    return result;
  }

  async function fetchMessages(
    id: number,
    options: { mailbox?: string; limit?: number; offset?: number },
  ): Promise<EmailMessageListResult> {
    const result = await listEmailMessages(id, options);
    const mailbox = options.mailbox ?? "";
    const key = `${id}:${mailbox}`;
    messagesByKey.value = { ...messagesByKey.value, [key]: result };
    return result;
  }

  async function fetchMessage(id: number, messageId: string): Promise<EmailMessageDetail> {
    const result = await getEmailMessage(id, messageId);
    const key = `${id}:${messageId}`;
    messageDetailsByKey.value = { ...messageDetailsByKey.value, [key]: result };
    return result;
  }

  async function previewMailboxes(payload: EmailPreviewInput): Promise<EmailMailboxListResult> {
    return previewEmailMailboxes(payload);
  }

  async function previewLatestMessage(payload: EmailPreviewInput): Promise<EmailLatestMessageResult> {
    return previewLatestEmailMessage(payload);
  }

  return {
    emailAccounts,
    mailboxesByAccount,
    messagesByKey,
    messageDetailsByKey,
    loading,
    error,
    fetchEmailAccounts,
    bulkImport,
    create,
    update,
    remove,
    buildAuthorizeUrl,
    exchangeCode,
    fetchMailboxes,
    fetchMessages,
    fetchMessage,
    previewMailboxes,
    previewLatestMessage,
  };
});
