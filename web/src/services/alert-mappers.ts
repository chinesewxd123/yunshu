export type PagedResult<T> = {
  items: T[];
  list: T[];
  total: number;
  page: number;
  page_size: number;
};

export function parseStringArray(raw?: string): string[] {
  const s = String(raw || "").trim();
  if (!s) return [];
  try {
    const parsed = JSON.parse(s) as unknown;
    if (!Array.isArray(parsed)) return [];
    return parsed.map((it) => String(it ?? "").trim()).filter(Boolean);
  } catch {
    return [];
  }
}

export function parseCommaSeparatedList(raw?: string): string[] {
  return String(raw || "")
    .split(",")
    .map((it) => it.trim())
    .filter(Boolean);
}

export function parseCommaSeparatedNumbers(raw?: string): number[] {
  return parseCommaSeparatedList(raw)
    .map((it) => Number(it))
    .filter((it) => Number.isFinite(it) && it > 0);
}

export function parseNumberArray(raw?: string): number[] {
  return parseStringArray(raw)
    .map((it) => Number(it))
    .filter((it) => Number.isFinite(it) && it > 0);
}

export function parseStringMap(raw?: string): Record<string, string> {
  const s = String(raw || "").trim();
  if (!s) return {};
  try {
    const parsed = JSON.parse(s) as Record<string, unknown>;
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) return {};
    return Object.fromEntries(
      Object.entries(parsed)
        .map(([key, value]) => [String(key).trim(), String(value ?? "").trim()])
        .filter(([key, value]) => key && value),
    );
  } catch {
    return {};
  }
}

export function stringifyPrettyJSON(value: unknown, fallback = "{}"): string {
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return fallback;
  }
}

export function normalizePagedPayload<TInput, TOutput = TInput>(
  payload: { items?: TInput[]; list?: TInput[]; total?: number; page?: number; page_size?: number },
  mapItem?: (item: TInput) => TOutput,
): PagedResult<TOutput> {
  const source = Array.isArray(payload.items) ? payload.items : Array.isArray(payload.list) ? payload.list : [];
  const items = mapItem ? source.map(mapItem) : (source as unknown as TOutput[]);
  return {
    items,
    list: items,
    total: Number(payload.total || 0),
    page: Number(payload.page || 1),
    page_size: Number(payload.page_size || items.length || 10),
  };
}
