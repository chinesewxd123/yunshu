import React, { useEffect, useMemo, useRef, useState } from "react";

export interface LineSeries {
  name?: string;
  data: number[];
  color?: string;
}

interface LineChartProps {
  /** Backward-compatible single-series input */
  data?: number[];
  /** Multi-series input (preferred) */
  series?: LineSeries[];
  labels?: string[];
  height?: number;
  showLegend?: boolean;
}

export function LineChart({ data, series, labels, height = 220, showLegend = true }: LineChartProps) {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const [viewWidth, setViewWidth] = useState(700);
  const w = viewWidth;
  const h = height;
  const padding = 24;
  const svgRef = useRef<SVGSVGElement | null>(null);
  const [hoverIndex, setHoverIndex] = useState<number | null>(null);
  const [hoverX, setHoverX] = useState<number>(0);
  const [hoverClient, setHoverClient] = useState<{ x: number; y: number } | null>(null);
  const normalizedSeries: LineSeries[] =
    series && series.length > 0
      ? series
      : [
          {
            name: undefined,
            data: data ?? [],
            color: "#2563eb",
          },
        ];

  const allValues = normalizedSeries.flatMap((s) => s.data);
  const max = Math.max(...allValues, 1);
  const min = Math.min(...allValues, 0);

  const nPoints = useMemo(() => {
    const maxLen = Math.max(...normalizedSeries.map((s) => s.data.length), 0);
    return maxLen;
  }, [normalizedSeries]);

  const safeLabels = useMemo(() => {
    if (!labels || labels.length === 0) return undefined;
    if (nPoints <= 0) return labels;
    if (labels.length === nPoints) return labels;
    // If labels mismatch, still use provided labels but clamp lookup.
    return labels;
  }, [labels, nPoints]);

  useEffect(() => {
    const el = containerRef.current;
    if (!el) return;
    const resize = () => {
      const next = Math.max(360, Math.floor(el.clientWidth || 700));
      setViewWidth(next);
    };
    resize();
    const observer = new ResizeObserver(() => resize());
    observer.observe(el);
    return () => observer.disconnect();
  }, []);

  function toXY(v: number, i: number, n: number) {
    const x = padding + (i * (w - padding * 2)) / Math.max(1, n - 1);
    const y = padding + ((max - v) * (h - padding * 2)) / Math.max(1, max - min);
    return { x, y };
  }

  function indexToX(i: number, n: number) {
    return padding + (i * (w - padding * 2)) / Math.max(1, n - 1);
  }

  function nearestIndexFromX(x: number, n: number) {
    if (n <= 1) return 0;
    const clamped = Math.max(padding, Math.min(w - padding, x));
    const t = (clamped - padding) / (w - padding * 2);
    return Math.max(0, Math.min(n - 1, Math.round(t * (n - 1))));
  }

  return (
    <div ref={containerRef} style={{ width: "100%", overflow: "hidden", position: "relative" }}>
      {showLegend && normalizedSeries.length > 1 ? (
        <div style={{ display: "flex", gap: 12, flexWrap: "wrap", padding: "0 2px 10px" }}>
          {normalizedSeries.map((s, idx) => {
            const color = s.color || ["#2563eb", "#10b981", "#f59e0b", "#ef4444"][idx % 4];
            return (
              <div key={idx} style={{ display: "flex", alignItems: "center", gap: 8, fontSize: 12, color: "#475569" }}>
                <span style={{ width: 10, height: 10, borderRadius: 999, background: color, boxShadow: "0 0 0 2px rgba(37,99,235,0.08)" }} />
                <span>{s.name || `Series ${idx + 1}`}</span>
              </div>
            );
          })}
        </div>
      ) : null}
      {hoverIndex !== null && hoverClient ? (
        <div
          style={{
            position: "absolute",
            left: Math.min(hoverClient.x + 12, (typeof window !== "undefined" ? window.innerWidth : 99999) - 320),
            top: hoverClient.y - 8,
            transform: "translateY(-100%)",
            pointerEvents: "none",
            background: "rgba(255,255,255,0.95)",
            border: "1px solid #e6eefc",
            borderRadius: 10,
            padding: "10px 12px",
            boxShadow: "0 10px 30px rgba(15, 23, 42, 0.12)",
            minWidth: 220,
            backdropFilter: "blur(6px)",
            WebkitBackdropFilter: "blur(6px)",
          }}
        >
          <div style={{ fontSize: 12, color: "#475569", marginBottom: 6, fontWeight: 600 }}>
            {safeLabels?.[Math.min(hoverIndex, (safeLabels?.length ?? 1) - 1)] ?? `#${hoverIndex + 1}`}
          </div>
          <div style={{ display: "grid", gap: 6 }}>
            {normalizedSeries.map((s, sIdx) => {
              const color = s.color || ["#2563eb", "#10b981", "#f59e0b", "#ef4444"][sIdx % 4];
              const v = s.data[hoverIndex];
              return (
                <div key={sIdx} style={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: 12 }}>
                  <div style={{ display: "flex", alignItems: "center", gap: 8, minWidth: 120 }}>
                    <span style={{ width: 8, height: 8, borderRadius: 999, background: color }} />
                    <span style={{ fontSize: 12, color: "#334155" }}>{s.name || `Series ${sIdx + 1}`}</span>
                  </div>
                  <span style={{ fontSize: 12, color: "#0f172a", fontWeight: 700 }}>{typeof v === "number" ? v : "-"}</span>
                </div>
              );
            })}
          </div>
        </div>
      ) : null}

      <svg
        ref={svgRef}
        viewBox={`0 0 ${w} ${h}`}
        width="100%"
        height={h}
        preserveAspectRatio="xMinYMid meet"
        style={{ cursor: nPoints > 0 ? "crosshair" : "default" }}
        onMouseLeave={() => {
          setHoverIndex(null);
          setHoverClient(null);
        }}
        onMouseMove={(e) => {
          if (!svgRef.current || nPoints <= 0) return;
          const rect = svgRef.current.getBoundingClientRect();
          const px = e.clientX - rect.left;
          const py = e.clientY - rect.top;
          // Convert to viewBox coordinates
          const vx = (px / Math.max(1, rect.width)) * w;
          const idx = nearestIndexFromX(vx, nPoints);
          setHoverIndex(idx);
          setHoverX(indexToX(idx, nPoints));
          setHoverClient({ x: e.clientX - rect.left, y: py + (showLegend && normalizedSeries.length > 1 ? 26 : 0) });
        }}
      >
        {/* grid lines */}
        {[0, 0.25, 0.5, 0.75, 1].map((t) => (
          <line
            key={t}
            x1={padding}
            x2={w - padding}
            y1={padding + t * (h - padding * 2)}
            y2={padding + t * (h - padding * 2)}
            stroke="#eef3fb"
            strokeWidth={1}
          />
        ))}

        {hoverIndex !== null && nPoints > 0 ? (
          <g>
            <line
              x1={hoverX}
              x2={hoverX}
              y1={padding}
              y2={h - padding}
              stroke="#c7d7f7"
              strokeWidth={1}
              strokeDasharray="4 4"
            />
          </g>
        ) : null}

        {normalizedSeries.map((s, sIdx) => {
          const color = s.color || ["#2563eb", "#10b981", "#f59e0b", "#ef4444"][sIdx % 4];
          const pts = s.data
            .map((v, i) => {
              const p = toXY(v, i, s.data.length);
              return `${p.x},${p.y}`;
            })
            .join(" ");

          const strokes = s.data.map((v, i) => ({ ...toXY(v, i, s.data.length), v }));

          return (
            <g key={sIdx}>
              <polyline
                fill="none"
                stroke={color}
                strokeWidth={2}
                strokeLinejoin="round"
                strokeLinecap="round"
                points={pts}
              />
              {strokes.map((p, idx) => {
                const isHover = hoverIndex === idx;
                return (
                  <g key={idx}>
                    <circle cx={p.x} cy={p.y} r={3.5} fill={color} opacity={isHover ? 1 : 0.85} />
                    {isHover ? (
                      <circle cx={p.x} cy={p.y} r={7} fill={color} opacity={0.18} />
                    ) : null}
                  </g>
                );
              })}
            </g>
          );
        })}
      </svg>
    </div>
  );
}

export default LineChart;
