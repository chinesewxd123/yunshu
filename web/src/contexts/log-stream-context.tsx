import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { getToken } from "../services/storage";
import { clearLogStreamSession, readLogStreamSession, writeLogStreamSession } from "../lib/log-stream-session";

export type StreamForm = {
  project_id?: number;
  server_id?: number;
  service_id?: number;
  log_source_id?: number;
  file_path?: string;
  tail_lines?: number;
  include?: string;
  exclude?: string;
  highlight?: string;
};

type LogStreamState = {
  streaming: boolean;
  paused: boolean;
  streamModeHint: string;
  lineCount: number;
  linesPerSec: number;
  lastEventId: number;
  form: StreamForm;
};

type LogStreamContextValue = LogStreamState & {
  setForm: (f: StreamForm) => void;
  start: (override?: StreamForm) => Promise<void>;
  stop: () => void;
  togglePause: () => void;
  attachWriter: (write: (line: string) => void) => () => void;
  registerTerminalClear: (fn: () => void) => () => void;
};

const LogStreamContext = createContext<LogStreamContextValue | null>(null);

function buildSseUrl(projectId: number, params: Record<string, string>) {
  const qs = new URLSearchParams(params);
  return `/api/v1/projects/${projectId}/logs/stream?${qs.toString()}`;
}

export function LogStreamProvider({ children }: { children: ReactNode }) {
  const [form, setFormState] = useState<StreamForm>({ tail_lines: 200 });
  const [streaming, setStreaming] = useState(false);
  const [paused, setPaused] = useState(false);
  const [streamModeHint, setStreamModeHint] = useState("未开始");
  const [lineCount, setLineCount] = useState(0);
  const [linesPerSec, setLinesPerSec] = useState(0);
  const [lastEventId, setLastEventId] = useState(0);

  const abortRef = useRef<AbortController | null>(null);
  const writersRef = useRef(new Set<(line: string) => void>());
  const clearRef = useRef<(() => void) | null>(null);
  const pausedRef = useRef(false);
  const lastEventIdRef = useRef(0);
  const lineBucketRef = useRef(0);
  const lineBucketTsRef = useRef(Date.now());

  useEffect(() => {
    const id = window.setInterval(() => {
      const now = Date.now();
      const dt = (now - lineBucketTsRef.current) / 1000;
      if (dt >= 1) {
        setLinesPerSec(Math.round(lineBucketRef.current / dt));
        lineBucketRef.current = 0;
        lineBucketTsRef.current = now;
      }
    }, 1000);
    return () => window.clearInterval(id);
  }, []);

  const attachWriter = useCallback((write: (line: string) => void) => {
    writersRef.current.add(write);
    return () => {
      writersRef.current.delete(write);
    };
  }, []);

  const registerTerminalClear = useCallback((fn: () => void) => {
    clearRef.current = fn;
    return () => {
      if (clearRef.current === fn) clearRef.current = null;
    };
  }, []);

  const emitLine = useCallback((line: string) => {
    if (pausedRef.current) return;
    lineBucketRef.current += 1;
    setLineCount((c) => c + 1);
    writersRef.current.forEach((w) => w(line));
  }, []);

  const stop = useCallback(() => {
    abortRef.current?.abort();
    abortRef.current = null;
    setStreaming(false);
    setPaused(false);
    pausedRef.current = false;
    setStreamModeHint("已停止");
    writeLogStreamSession({ form, streaming: false, lastEventId: lastEventIdRef.current });
  }, [form]);

  const runStreamLoop = useCallback(
    async (values: StreamForm, ac: AbortController) => {
      const projectId = values.project_id!;
      const serverId = values.server_id!;
      const sourceId = values.log_source_id!;

      const baseParams: Record<string, string> = {
        project_id: String(projectId),
        server_id: String(serverId),
        log_source_id: String(sourceId),
        tail_lines: String(values.tail_lines ?? 200),
        source: "agent",
      };
      if (values.include) baseParams.include = values.include;
      if (values.exclude) baseParams.exclude = values.exclude;
      if (values.highlight) baseParams.highlight = values.highlight;
      if (values.file_path?.trim()) baseParams.file_path = values.file_path.trim();

      const token = getToken();
      let retries = 0;
      const maxRetries = 5;

      while (!ac.signal.aborted && retries <= maxRetries) {
        const params = { ...baseParams };
        if (lastEventIdRef.current > 0) {
          params.after_id = String(lastEventIdRef.current);
        }
        const url = buildSseUrl(projectId, params);
        try {
          const resp = await fetch(url, {
            headers: { Authorization: token ? `Bearer ${token}` : "" },
            signal: ac.signal,
          });
          if (!resp.ok || !resp.body) {
            throw new Error(`stream failed: ${resp.status}`);
          }
          setStreamModeHint(retries > 0 ? `运行中（已重连 #${retries}）` : "运行中");
          const reader = resp.body.getReader();
          const decoder = new TextDecoder("utf-8");
          let buf = "";
          while (!ac.signal.aborted) {
            const { done, value } = await reader.read();
            if (done) break;
            buf += decoder.decode(value, { stream: true });
            let idx: number;
            while ((idx = buf.indexOf("\n\n")) >= 0) {
              const rawEvent = buf.slice(0, idx);
              buf = buf.slice(idx + 2);
              const lines = rawEvent.split("\n");
              const eventName = lines.find((l) => l.startsWith("event:"))?.slice("event:".length).trim();
              const dataLine = lines.find((l) => l.startsWith("data:"))?.slice("data:".length).trim();
              if (!dataLine || eventName === "ping") continue;
              if (eventName === "log") {
                try {
                  const payload = JSON.parse(dataLine) as { line?: string; id?: number };
                  if (typeof payload.id === "number" && payload.id > lastEventIdRef.current) {
                    lastEventIdRef.current = payload.id;
                    setLastEventId(payload.id);
                  }
                  if (payload.line != null) emitLine(payload.line);
                } catch {
                  emitLine(dataLine);
                }
              }
            }
          }
          retries++;
          if (ac.signal.aborted) break;
          await new Promise((r) => setTimeout(r, Math.min(1000 * retries, 8000)));
        } catch (e: unknown) {
          if (ac.signal.aborted || (e as { name?: string })?.name === "AbortError") break;
          retries++;
          if (retries > maxRetries) {
            setStreamModeHint("连接失败");
            break;
          }
          await new Promise((r) => setTimeout(r, Math.min(1000 * retries, 8000)));
        }
      }
    },
    [emitLine],
  );

  const start = useCallback(async (override?: StreamForm) => {
    const values = override ?? form;
    if (!values.project_id || !values.server_id || !values.log_source_id) {
      throw new Error("请选择 project / server / log source");
    }
    if (override) setFormState((prev) => ({ ...prev, ...override }));
    stop();
    clearRef.current?.();
    setLineCount(0);
    lineBucketRef.current = 0;
    lastEventIdRef.current = 0;
    setLastEventId(0);

    const ac = new AbortController();
    abortRef.current = ac;
    setStreaming(true);
    setPaused(false);
    pausedRef.current = false;
    setStreamModeHint("连接中…");
    writeLogStreamSession({ form: values, streaming: true, lastEventId: 0 });

    emitLine(`Streaming logs... project=${values.project_id} server=${values.server_id} source=${values.log_source_id}`);
    void runStreamLoop(values, ac).finally(() => {
      if (abortRef.current === ac) {
        setStreaming(false);
        setStreamModeHint("已停止");
        writeLogStreamSession({ form: values, streaming: false, lastEventId: lastEventIdRef.current });
        abortRef.current = null;
      }
    });
  }, [form, stop, emitLine, runStreamLoop]);

  const togglePause = useCallback(() => {
    setPaused((p) => {
      pausedRef.current = !p;
      setStreamModeHint(!p ? "已暂停（后台仍连接）" : streaming ? "运行中" : "已停止");
      return !p;
    });
  }, [streaming]);

  const setForm = useCallback((f: StreamForm) => {
    setFormState((prev) => ({ ...prev, ...f }));
  }, []);

  const value = useMemo(
    () => ({
      form,
      setForm,
      streaming,
      paused,
      streamModeHint,
      lineCount,
      linesPerSec,
      lastEventId,
      start,
      stop,
      togglePause,
      attachWriter,
      registerTerminalClear,
    }),
    [form, streaming, paused, streamModeHint, lineCount, linesPerSec, lastEventId, start, stop, togglePause, attachWriter, registerTerminalClear, setForm],
  );

  return <LogStreamContext.Provider value={value}>{children}</LogStreamContext.Provider>;
}

export function useLogStream() {
  const ctx = useContext(LogStreamContext);
  if (!ctx) throw new Error("useLogStream must be used within LogStreamProvider");
  return ctx;
}

export function useLogStreamOptional() {
  return useContext(LogStreamContext);
}

/** 从 session 恢复「离开页面前仍在推流」的意图（由日志页挂载时调用 autostart）。 */
export function useRestoreLogStreamIntent() {
  const stream = useLogStreamOptional();
  return useCallback(() => {
    const saved = readLogStreamSession();
    if (saved?.streaming && saved.form?.project_id) {
      stream?.setForm(saved.form);
      return true;
    }
    return false;
  }, [stream]);
}
