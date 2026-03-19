export function formatJSON(value: unknown): string {
  return JSON.stringify(value, null, 2);
}

export function formatJSONObject(value: Record<string, unknown> | null | undefined): string {
  return formatJSON(value ?? {});
}

export function parseJSONObject(input: string): Record<string, unknown> {
  const parsed = JSON.parse(input);
  if (parsed === null || Array.isArray(parsed) || typeof parsed !== "object") {
    throw new Error("JSON input must be an object.");
  }

  return parsed as Record<string, unknown>;
}

export function getErrorMessage(error: unknown): string {
  if (error instanceof Error) {
    return error.message;
  }
  if (typeof error === "string") {
    return error;
  }
  return "";
}
