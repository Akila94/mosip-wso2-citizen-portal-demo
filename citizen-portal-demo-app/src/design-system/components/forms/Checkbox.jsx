import React from "react";

/**
 * Checkbox with label. Controlled via `checked`/`onChange` or uncontrolled.
 */
export function Checkbox({ label, checked, defaultChecked, onChange, disabled = false, id, style = {}, ...rest }) {
  const [internal, setInternal] = React.useState(!!defaultChecked);
  const isControlled = checked !== undefined;
  const on = isControlled ? checked : internal;
  const cbId = id || React.useId();
  const toggle = (e) => {
    if (!isControlled) setInternal(e.target.checked);
    onChange && onChange(e);
  };
  return (
    <label htmlFor={cbId} style={{ display: "inline-flex", alignItems: "center", gap: 10, cursor: disabled ? "not-allowed" : "pointer", opacity: disabled ? 0.5 : 1, ...style }}>
      <span
        style={{
          width: 18,
          height: 18,
          borderRadius: "var(--radius-xs)",
          border: on ? "1px solid var(--accent)" : "1px solid var(--border-strong)",
          background: on ? "var(--accent)" : "var(--surface-card)",
          display: "inline-flex",
          alignItems: "center",
          justifyContent: "center",
          flex: "none",
          transition: "background var(--dur-fast) var(--ease-standard), border-color var(--dur-fast) var(--ease-standard)",
        }}
      >
        {on && (
          <svg width="12" height="12" viewBox="0 0 12 12" fill="none">
            <path d="M2.5 6.2 5 8.6l4.5-5" stroke="#fff" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
          </svg>
        )}
      </span>
      <input id={cbId} type="checkbox" checked={on} disabled={disabled} onChange={toggle} style={{ position: "absolute", opacity: 0, width: 0, height: 0 }} {...rest} />
      {label && <span style={{ fontFamily: "var(--font-sans)", fontSize: 14, color: "var(--text-primary)" }}>{label}</span>}
    </label>
  );
}
