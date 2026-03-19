import { ref, onMounted } from "vue";
import { storeToRefs } from "pinia";
import { useAccountsStore } from "@/store";
import type { AccountExecuteInput, AccountExecuteResult, AccountPatchInput } from "@/types";

export function useAccounts() {
  const store = useAccountsStore();
  const { accounts, loading, error } = storeToRefs(store);

  async function refresh() {
    await store.fetchAccounts();
  }

  onMounted(() => { void refresh(); });

  return { data: accounts, loading, error, refresh };
}

export function useCreateAccount() {
  const loading = ref(false);
  const error = ref<string | null>(null);
  const store = useAccountsStore();

  async function execute(payload: Parameters<typeof store.create>[0]) {
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

export function usePatchAccount() {
  const loading = ref(false);
  const error = ref<string | null>(null);
  const store = useAccountsStore();

  async function execute(id: number, payload: AccountPatchInput) {
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

export function useExecuteAccount() {
  const loading = ref(false);
  const error = ref<string | null>(null);
  const store = useAccountsStore();

  async function execute(
    id: number,
    action: string,
    params: Record<string, unknown> = {},
  ): Promise<AccountExecuteResult> {
    loading.value = true;
    error.value = null;
    try {
      const result = await store.execute(id, { action, params } as AccountExecuteInput);
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

export function useDeleteAccount() {
  const loading = ref(false);
  const error = ref<string | null>(null);
  const store = useAccountsStore();

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
