import { client } from "@/shared/api/generated/client";
import type {
  AccountExecuteInput,
  AccountExecuteResult,
  AccountCreateInput,
  AccountPatchInput,
  ListAccountsResponse,
} from "@/types";

export const listAccounts = (): Promise<ListAccountsResponse> => client.listAccounts();

export const createAccount = (payload: AccountCreateInput) =>
  client.createAccount({ body: payload });

export const patchAccount = (id: number, payload: AccountPatchInput) =>
  client.patchAccount({ path: { id }, body: payload });

export const executeAccount = (
  id: number,
  payload: AccountExecuteInput,
): Promise<AccountExecuteResult> =>
  client.executeAccount({ path: { id }, body: payload });

export const deleteAccount = (id: number) =>
  client.deleteAccount({ path: { id } });
