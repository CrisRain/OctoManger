import { computed, ref, onMounted } from "vue";
import { storeToRefs } from "pinia";
import { useEmailAccountsStore } from "@/store";
import type {
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

export function useEmailAccounts() {
  const store = useEmailAccountsStore();
  const { emailAccounts, loading, error } = storeToRefs(store);

  async function refresh() {
    await store.fetchEmailAccounts();
  }

  onMounted(() => { void refresh(); });

  return { data: emailAccounts, loading, error, refresh };
}

export function useBulkImportEmailAccounts() {
  const loading = ref(false);
  const error = ref<string | null>(null);
  const store = useEmailAccountsStore();

  async function execute(lines: string[]): Promise<EmailBulkImportResult> {
    loading.value = true;
    error.value = null;
    try {
      const result = await store.bulkImport(lines);
      return result;
    } catch (e) {
      error.value = e instanceof Error ? e.message : "请求失败";
      throw e;
    } finally {
      loading.value = false;
    }
  }

  return { loading, error, execute };
}

export function useCreateEmailAccount() {
  const loading = ref(false);
  const error = ref<string | null>(null);
  const store = useEmailAccountsStore();

  async function execute(payload: EmailAccountCreateInput) {
    loading.value = true;
    error.value = null;
    try {
      const result = await store.create(payload);
      return result;
    } catch (e) {
      error.value = e instanceof Error ? e.message : "请求失败";
      throw e;
    } finally {
      loading.value = false;
    }
  }

  return { loading, error, execute };
}

export function usePatchEmailAccount() {
  const loading = ref(false);
  const error = ref<string | null>(null);
  const store = useEmailAccountsStore();

  async function execute(id: number, payload: EmailAccountPatchInput) {
    loading.value = true;
    error.value = null;
    try {
      const result = await store.update(id, payload);
      return result;
    } catch (e) {
      error.value = e instanceof Error ? e.message : "请求失败";
      throw e;
    } finally {
      loading.value = false;
    }
  }

  return { loading, error, execute };
}

export function useDeleteEmailAccount() {
  const loading = ref(false);
  const error = ref<string | null>(null);
  const store = useEmailAccountsStore();

  async function execute(id: number) {
    loading.value = true;
    error.value = null;
    try {
      await store.remove(id);
    } catch (e) {
      error.value = e instanceof Error ? e.message : "请求失败";
      throw e;
    } finally {
      loading.value = false;
    }
  }

  return { loading, error, execute };
}

export function useBuildAuthorizeURL() {
  const loading = ref(false);
  const error = ref<string | null>(null);
  const store = useEmailAccountsStore();

  async function execute(id: number): Promise<OutlookAuthorizeURLResult> {
    loading.value = true;
    error.value = null;
    try {
      const result = await store.buildAuthorizeUrl(id);
      return result;
    } catch (e) {
      error.value = e instanceof Error ? e.message : "请求失败";
      throw e;
    } finally {
      loading.value = false;
    }
  }

  return { loading, error, execute };
}

export function useExchangeCode() {
  const loading = ref(false);
  const error = ref<string | null>(null);
  const store = useEmailAccountsStore();

  async function execute(id: number, payload: OutlookExchangeCodeInput) {
    loading.value = true;
    error.value = null;
    try {
      const result = await store.exchangeCode(id, payload);
      return result;
    } catch (e) {
      error.value = e instanceof Error ? e.message : "请求失败";
      throw e;
    } finally {
      loading.value = false;
    }
  }

  return { loading, error, execute };
}

export function useEmailMailboxes(accountId: number | null, pattern: () => string) {
  const store = useEmailAccountsStore();
  const { mailboxesByAccount } = storeToRefs(store);
  const data = computed<EmailMailboxListResult | null>(() => {
    if (!accountId) return null;
    return mailboxesByAccount.value[accountId] ?? null;
  });
  const loading = ref(false);
  const error = ref<string | null>(null);

  async function refresh() {
    if (!accountId) return;
    loading.value = true;
    try {
      await store.fetchMailboxes(accountId, pattern());
      error.value = null;
    } catch (e) {
      error.value = e instanceof Error ? e.message : "请求失败";
    } finally {
      loading.value = false;
    }
  }

  onMounted(() => { void refresh(); });

  return { data, loading, error, refresh };
}

export function useEmailMessages(accountId: number | null, mailbox: () => string) {
  const store = useEmailAccountsStore();
  const { messagesByKey } = storeToRefs(store);
  const data = computed<EmailMessageListResult | null>(() => {
    if (!accountId || !mailbox()) return null;
    const key = `${accountId}:${mailbox()}`;
    return messagesByKey.value[key] ?? null;
  });
  const loading = ref(false);
  const error = ref<string | null>(null);

  async function refresh() {
    if (!accountId || !mailbox()) return;
    loading.value = true;
    try {
      await store.fetchMessages(accountId, {
        mailbox: mailbox(),
        limit: 20,
        offset: 0,
      });
      error.value = null;
    } catch (e) {
      error.value = e instanceof Error ? e.message : "请求失败";
    } finally {
      loading.value = false;
    }
  }

  onMounted(() => { void refresh(); });

  return { data, loading, error, refresh };
}

export function useEmailMessage(accountId: number | null, messageId: () => string) {
  const store = useEmailAccountsStore();
  const { messageDetailsByKey } = storeToRefs(store);
  const data = computed<EmailMessageDetail | null>(() => {
    if (!accountId || !messageId()) return null;
    const key = `${accountId}:${messageId()}`;
    return messageDetailsByKey.value[key] ?? null;
  });
  const loading = ref(false);
  const error = ref<string | null>(null);

  async function refresh() {
    if (!accountId || !messageId()) return;
    loading.value = true;
    try {
      await store.fetchMessage(accountId, messageId());
      error.value = null;
    } catch (e) {
      error.value = e instanceof Error ? e.message : "请求失败";
    } finally {
      loading.value = false;
    }
  }

  onMounted(() => { void refresh(); });

  return { data, loading, error, refresh };
}

export function usePreviewMailboxes() {
  const loading = ref(false);
  const error = ref<string | null>(null);
  const store = useEmailAccountsStore();

  async function execute(payload: EmailPreviewInput) {
    loading.value = true;
    error.value = null;
    try {
      const result = await store.previewMailboxes(payload);
      return result;
    } catch (e) {
      error.value = e instanceof Error ? e.message : "请求失败";
      throw e;
    } finally {
      loading.value = false;
    }
  }

  return { loading, error, execute };
}

export function usePreviewLatestMessage() {
  const loading = ref(false);
  const error = ref<string | null>(null);
  const store = useEmailAccountsStore();

  async function execute(payload: EmailPreviewInput): Promise<EmailLatestMessageResult> {
    loading.value = true;
    error.value = null;
    try {
      const result = await store.previewLatestMessage(payload);
      return result;
    } catch (e) {
      error.value = e instanceof Error ? e.message : "请求失败";
      throw e;
    } finally {
      loading.value = false;
    }
  }

  return { loading, error, execute };
}
