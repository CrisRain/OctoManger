import { client } from "@/shared/api/generated/client";
import type { AccountTypeCreateInput, ListAccountTypesResponse } from "@/types";

export const listAccountTypes = (): Promise<ListAccountTypesResponse> =>
  client.listAccountTypes();

export const createAccountType = (payload: AccountTypeCreateInput) =>
  client.createAccountType({ body: payload });

export const deleteAccountType = (key: string) =>
  client.deleteAccountType({ path: { key } });
