import React from "react";

const TONES = {
  info:    { bg: "var(--info-bg)", border: "var(--blue-100)", icon: "var(--blue-600)", mark: "ℹ" },
  success: { bg: "var(--success-bg)", border: "#bfe6cf", icon: "var(--success)", mark: "✓" },
  warning: { bg: "var(--warning-bg)", border: "#f6dca6", icon: "var(--amber-500)", mark: "!" },
  danger:  { bg: "var(--danger-bg)", border: "#f3c4c2", icon: "var(--danger)", mark: "!" },
};

/**
 * Inline alert / callout banner.
 */
export function Alert({ tone = "info", title, children, onClose, style = {}, ...rest }) {
  const t = TONES[tone] || TONES.info;
  return (
    <div
      role="status"
      style={{
        display: "flex",
        gap: 12,
        padding: "14px 16px",
        background: t.bg,
        border: `1px solid ${t.border}`,
        borderRadius: "var(--radius-lg)",
        fontFamily: "var(--font-sans)",
        ...style,
      }}
      {...rest}
    >
      <span
        style={{
          width: 20,
          height: 20,
          borderRadius: "50%",
          background: t.icon,
          color: "#fff",
          fontSize: 12,
          fontWeight: 700,
          display: "inline-flex",
          alignItems: "center",
          justifyContent: "center",
          flex: "none",
          marginTop: 1,
        }}
      >
        {t.mark}
      </span>
      <div style={{ flex: 1, minWidth: 0 }}>
        {title && <div style={{ fontSize: 14, fontWeight: 600, color: "var(--text-primary)", marginBottom: children ? 2 : 0 }}>{title}</div>}
        {children && <div style={{ fontSize: 13.5, color: "var(--text-secondary)", lineHeight: 1.5 }}>{children}</div>}
      </div>
      {onClose && (
        <button onClick={onClose} aria-label="Dismiss" style={{ background: "none", border: "none", cursor: "pointer", color: "var(--text-muted)", fontSize: 16, lineHeight: 1, padding: 0, height: 18 }}>×</button>
      )}
    </div>
  );
}
