import { ref, onMounted } from "vue";
import { storeToRefs } from "pinia";
import { useAccountTypesStore } from "@/store";
import type { AccountTypeCreateInput } from "@/types";

export function useAccountTypes() {
  const store = useAccountTypesStore();
  const { accountTypes, loading, error } = storeToRefs(store);

  async function refresh() {
    await store.fetchAccountTypes();
  }

  onMounted(() => { void refresh(); });

  return { data: accountTypes, loading, error, refresh };
}

export function useCreateAccountType() {
  const loading = ref(false);
  const error = ref<string | null>(null);
  const store = useAccountTypesStore();

  async function execute(payload: AccountTypeCreateInput) {
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

export function useDeleteAccountType() {
  const loading = ref(false);
  const error = ref<string | null>(null);
  const store = useAccountTypesStore();

  async function execute(key: string) {
    loading.value = true;
    error.value = null;
    try {
      await store.remove(key);
    } catch (e) {
      error.value = e instanceof Error ? e.message : "请求失败";
      throw e;
    } finally {
      loading.value = false;
    }
  }

  return { loading, error, execute };
}
