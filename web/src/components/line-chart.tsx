import React from "react";

interface LineChartProps {
  data: number[];
  labels?: string[];
  height?: number;
}

export function LineChart({ data, labels, height = 220 }: LineChartProps) {
  const w = 700;
  const h = height;
  const padding = 24;
  const max = Math.max(...data, 1);
  const min = Math.min(...data, 0);
  const points = data
    .map((v, i) => {
      const x = padding + (i * (w - padding * 2)) / Math.max(1, data.length - 1);
      const y = padding + ((max - v) * (h - padding * 2)) / Math.max(1, max - min);
      return `${x},${y}`;
    })
    .join(" ");

  const strokes = data.map((v, i) => {
    const x = padding + (i * (w - padding * 2)) / Math.max(1, data.length - 1);
    const y = padding + ((max - v) * (h - padding * 2)) / Math.max(1, max - min);
    return { x, y, v };
  });

  return (
    <div style={{ width: "100%", overflow: "hidden" }}>
      <svg viewBox={`0 0 ${w} ${h}`} width="100%" height={h} preserveAspectRatio="xMidYMid meet">
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

        {/* area / line */}
        <polyline
          fill="none"
          stroke="#2563eb"
          strokeWidth={2}
          strokeLinejoin="round"
          strokeLinecap="round"
          points={points}
        />

        {/* points */}
        {strokes.map((p, idx) => (
          <circle key={idx} cx={p.x} cy={p.y} r={3.5} fill="#2563eb" />
        ))}
      </svg>
    </div>
  );
}

export default LineChart;
