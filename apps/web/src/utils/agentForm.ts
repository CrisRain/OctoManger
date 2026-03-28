type JsonRecord = Record<string, unknown>;

function asRecord(value: unknown): JsonRecord {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return {};
  }
  return value as JsonRecord;
}

export function parseAgentParamsJSON(raw: string): JsonRecord {
  const trimmed = raw.trim();
  if (!trimmed) {
    return {};
  }

  const parsed = JSON.parse(trimmed) as unknown;
  if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
    throw new Error("额外参数必须是 JSON 对象");
  }
  return parsed as JsonRecord;
}

export function buildAgentInput(params: JsonRecord): JsonRecord {
  const input: JsonRecord = {};

  if (Object.keys(params).length > 0) {
    input.params = params;
  }

  return input;
}

export function splitAgentInput(input: Record<string, unknown> | null | undefined): {
  params: JsonRecord;
} {
  const rawInput = asRecord(input);
  const rawParams = asRecord(rawInput.params);

  const extraParams: JsonRecord = {};
  for (const [key, value] of Object.entries(rawInput)) {
    if (key === "params") {
      continue;
    }
    extraParams[key] = value;
  }

  const params = Object.keys(rawParams).length > 0 ? rawParams : extraParams;

  return { params };
}

export function stringifyAgentParams(params: JsonRecord): string {
  return JSON.stringify(params, null, 2);
}
