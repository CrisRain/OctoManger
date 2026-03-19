import axios, { AxiosHeaders, type AxiosInstance } from "axios";

const ADMIN_KEY_STORAGE_KEY = "octo_admin_key";

const request: AxiosInstance = axios.create({
  baseURL: (import.meta.env.VITE_API_BASE as string | undefined)?.replace(/\/+$/, "") ?? "",
  timeout: 15000,
});

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === "object" && value !== null;

const extractErrorMessage = (data: unknown): string | undefined => {
  if (!isRecord(data)) {
    return undefined;
  }
  const message = data.message;
  if (typeof message === "string" && message.trim().length > 0) {
    return message;
  }
  const detail = data.detail;
  if (typeof detail === "string" && detail.trim().length > 0) {
    return detail;
  }
  return undefined;
};

request.interceptors.request.use((config) => {
  const adminKey = localStorage.getItem(ADMIN_KEY_STORAGE_KEY)?.trim();
  if (adminKey) {
    const headers = AxiosHeaders.from(config.headers ?? {});
    headers.set("X-Admin-Key", adminKey);
    config.headers = headers;
  }
  return config;
});

request.interceptors.response.use(
  (response) => response,
  (error: unknown) => {
    if (axios.isAxiosError(error)) {
      const responseData = error.response?.data;
      const message =
        typeof responseData === "string"
          ? responseData
          : extractErrorMessage(responseData) ?? error.message;
      return Promise.reject(new Error(message));
    }
    return Promise.reject(error);
  },
);

export { request };
