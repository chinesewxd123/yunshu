import type { StreamForm } from "../contexts/log-stream-context";

const STORAGE_KEY = "yunshu:log-stream:v1";

export type LogStreamPersisted = {
  form: Partial<StreamForm>;
  streaming?: boolean;
  lastEventId?: number;
};

export function readLogStreamSession(): LogStreamPersisted | null {
  try {
    const raw = sessionStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    return JSON.parse(raw) as LogStreamPersisted;
  } catch {
    return null;
  }
}

export function writeLogStreamSession(data: LogStreamPersisted) {
  try {
    sessionStorage.setItem(STORAGE_KEY, JSON.stringify(data));
  } catch {
    /* quota */
  }
}

export function clearLogStreamSession() {
  try {
    sessionStorage.removeItem(STORAGE_KEY);
  } catch {
    /* ignore */
  }
}

export function syncLogStreamSearchParams(form: Partial<StreamForm>, streaming: boolean) {
  const url = new URL(window.location.href);
  const set = (k: string, v: string | number | undefined) => {
    if (v === undefined || v === "" || v === 0) url.searchParams.delete(k);
    else url.searchParams.set(k, String(v));
  };
  set("project_id", form.project_id);
  set("server_id", form.server_id);
  set("service_id", form.service_id);
  set("log_source_id", form.log_source_id);
  set("tail_lines", form.tail_lines ?? 200);
  if (form.file_path) set("file_path", form.file_path);
  if (form.include) set("include", form.include);
  if (form.exclude) set("exclude", form.exclude);
  if (form.highlight) set("highlight", form.highlight);
  if (streaming) url.searchParams.set("autostart", "1");
  else url.searchParams.delete("autostart");
  window.history.replaceState(null, "", url.pathname + url.search);
}

export function parseLogStreamFromSearch(): Partial<StreamForm> & { autostart?: boolean } {
  const p = new URLSearchParams(window.location.search);
  const num = (k: string) => {
    const v = Number(p.get(k));
    return Number.isFinite(v) && v > 0 ? v : undefined;
  };
  return {
    project_id: num("project_id"),
    server_id: num("server_id"),
    service_id: num("service_id"),
    log_source_id: num("log_source_id"),
    tail_lines: Number(p.get("tail_lines")) || 200,
    file_path: p.get("file_path") || undefined,
    include: p.get("include") || undefined,
    exclude: p.get("exclude") || undefined,
    highlight: p.get("highlight") || undefined,
    autostart: p.get("autostart") === "1",
  };
}
