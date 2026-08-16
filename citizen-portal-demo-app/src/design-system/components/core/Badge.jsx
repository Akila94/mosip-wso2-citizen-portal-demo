import React from "react";

const TONES = {
  neutral: { bg: "var(--gray-100)", fg: "var(--gray-700)" },
  brand:   { bg: "var(--orange-50)", fg: "var(--orange-700)" },
  success: { bg: "var(--success-bg)", fg: "var(--success)" },
  warning: { bg: "var(--warning-bg)", fg: "var(--amber-500)" },
  danger:  { bg: "var(--danger-bg)", fg: "var(--danger)" },
  info:    { bg: "var(--info-bg)", fg: "var(--blue-600)" },
  solid:   { bg: "var(--accent)", fg: "#fff" },
};

/**
 * Small status / category label. Pill or rounded-rect.
 */
export function Badge({ children, tone = "neutral", dot = false, style = {}, ...rest }) {
  const t = TONES[tone] || TONES.neutral;
  return (
    <span
      style={{
        display: "inline-flex",
        alignItems: "center",
        gap: 6,
        height: 22,
        padding: "0 9px",
        background: t.bg,
        color: t.fg,
        fontFamily: "var(--font-sans)",
        fontSize: 12,
        fontWeight: 600,
        letterSpacing: "0.01em",
        borderRadius: "var(--radius-pill)",
        whiteSpace: "nowrap",
        ...style,
      }}
      {...rest}
    >
      {dot && (
        <span style={{ width: 6, height: 6, borderRadius: "50%", background: "currentColor" }} />
      )}
      {children}
    </span>
  );
}
