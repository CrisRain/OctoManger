import { defineStore } from "pinia";
import { ref } from "vue";
import { createAccount, deleteAccount, executeAccount, listAccounts, patchAccount } from "@/api";
import type { Account, AccountCreateInput, AccountExecuteInput, AccountExecuteResult, AccountPatchInput } from "@/types";
import { normalizeListResponse } from "@/utils/normalizeListResponse";

export const useAccountsStore = defineStore("accounts", () => {
  const accounts = ref<Account[]>([]);
  const loading = ref(false);
  const error = ref<string | null>(null);

  async function fetchAccounts() {
    loading.value = true;
    error.value = null;
    try {
      accounts.value = normalizeListResponse<Account>(await listAccounts());
    } catch (e) {
      error.value = e instanceof Error ? e.message : "请求失败";
    } finally {
      loading.value = false;
    }
  }

  async function create(payload: AccountCreateInput) {
    const result = await createAccount(payload);
    accounts.value = [result, ...accounts.value];
    return result;
  }

  async function update(id: number, payload: AccountPatchInput) {
    const result = await patchAccount(id, payload);
    accounts.value = accounts.value.map((item) => (item.id === id ? result : item));
    return result;
  }

  async function remove(id: number) {
    await deleteAccount(id);
    accounts.value = accounts.value.filter((item) => item.id !== id);
  }

  async function execute(id: number, payload: AccountExecuteInput): Promise<AccountExecuteResult> {
    return executeAccount(id, payload);
  }

  return {
    accounts,
    loading,
    error,
    fetchAccounts,
    create,
    update,
    remove,
    execute,
  };
});
