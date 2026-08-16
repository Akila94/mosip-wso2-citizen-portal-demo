import React from "react";

/**
 * Native select styled to match the WSO2 input system.
 */
export function Select({ label, options = [], size = "md", id, containerStyle = {}, disabled = false, style = {}, ...rest }) {
  const [focus, setFocus] = React.useState(false);
  const heights = { sm: 34, md: 40, lg: 48 };
  const h = heights[size] || 40;
  const selId = id || React.useId();
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 6, ...containerStyle }}>
      {label && (
        <label htmlFor={selId} style={{ fontFamily: "var(--font-sans)", fontSize: 13, fontWeight: 600, color: "var(--text-primary)" }}>
          {label}
        </label>
      )}
      <div style={{ position: "relative", display: "flex", alignItems: "center" }}>
        <select
          id={selId}
          disabled={disabled}
          onFocus={() => setFocus(true)}
          onBlur={() => setFocus(false)}
          style={{
            appearance: "none",
            width: "100%",
            height: h,
            padding: "0 36px 0 12px",
            background: disabled ? "var(--gray-50)" : "var(--surface-card)",
            border: `1px solid ${focus ? "var(--accent)" : "var(--border-strong)"}`,
            borderRadius: "var(--radius-md)",
            boxShadow: focus ? "0 0 0 3px var(--focus-ring)" : "none",
            fontFamily: "var(--font-sans)",
            fontSize: size === "sm" ? 13 : 14,
            color: "var(--text-primary)",
            cursor: disabled ? "not-allowed" : "pointer",
            outline: "none",
            transition: "border-color var(--dur-fast) var(--ease-standard), box-shadow var(--dur-fast) var(--ease-standard)",
            ...style,
          }}
          {...rest}
        >
          {options.map((o) => {
            const value = typeof o === "string" ? o : o.value;
            const labelTxt = typeof o === "string" ? o : o.label;
            return <option key={value} value={value}>{labelTxt}</option>;
          })}
        </select>
        <span style={{ position: "absolute", right: 12, pointerEvents: "none", color: "var(--text-muted)", fontSize: 12 }}>▾</span>
      </div>
    </div>
  );
}
