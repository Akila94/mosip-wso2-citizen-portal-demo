import React from "react";

const SIZES = {
  sm: { height: 34, fontSize: 13, padding: "0 10px" },
  md: { height: 40, fontSize: 14, padding: "0 12px" },
  lg: { height: 48, fontSize: 16, padding: "0 14px" },
};

/**
 * Text input with optional label, leading icon, and error state.
 */
export function Input({
  label,
  hint,
  error,
  size = "md",
  iconLeft,
  id,
  style = {},
  containerStyle = {},
  disabled = false,
  ...rest
}) {
  const [focus, setFocus] = React.useState(false);
  const s = SIZES[size] || SIZES.md;
  const inputId = id || React.useId();
  const borderColor = error
    ? "var(--danger)"
    : focus
    ? "var(--accent)"
    : "var(--border-strong)";
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 6, ...containerStyle }}>
      {label && (
        <label htmlFor={inputId} style={{ fontFamily: "var(--font-sans)", fontSize: 13, fontWeight: 600, color: "var(--text-primary)" }}>
          {label}
        </label>
      )}
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: 8,
          height: s.height,
          padding: s.padding,
          background: disabled ? "var(--gray-50)" : "var(--surface-card)",
          border: `1px solid ${borderColor}`,
          borderRadius: "var(--radius-md)",
          boxShadow: focus && !error ? "0 0 0 3px var(--focus-ring)" : "none",
          transition: "border-color var(--dur-fast) var(--ease-standard), box-shadow var(--dur-fast) var(--ease-standard)",
          opacity: disabled ? 0.6 : 1,
        }}
      >
        {iconLeft && <span style={{ color: "var(--text-muted)", display: "inline-flex" }}>{iconLeft}</span>}
        <input
          id={inputId}
          disabled={disabled}
          onFocus={() => setFocus(true)}
          onBlur={() => setFocus(false)}
          style={{
            flex: 1,
            border: "none",
            outline: "none",
            background: "transparent",
            fontFamily: "var(--font-sans)",
            fontSize: s.fontSize,
            color: "var(--text-primary)",
            minWidth: 0,
            ...style,
          }}
          {...rest}
        />
      </div>
      {(hint || error) && (
        <span style={{ fontFamily: "var(--font-sans)", fontSize: 12, color: error ? "var(--danger)" : "var(--text-secondary)" }}>
          {error || hint}
        </span>
      )}
    </div>
  );
}
