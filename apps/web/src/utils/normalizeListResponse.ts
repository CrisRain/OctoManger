type ListResponseLike = {
  items?: unknown;
  data?: unknown;
};

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

export function normalizeListResponse<T>(payload: unknown): T[] {
  const source = payload as ListResponseLike;
  if (Array.isArray(source?.items)) {
    return source.items as T[];
  }
  if (Array.isArray(source?.data)) {
    return source.data as T[];
  }
  if (isRecord(source?.data) && Array.isArray((source.data as ListResponseLike).items)) {
    return (source.data as ListResponseLike).items as T[];
  }
  return [];
}
