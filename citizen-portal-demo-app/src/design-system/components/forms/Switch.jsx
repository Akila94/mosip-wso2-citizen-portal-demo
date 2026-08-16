import React from "react";

/**
 * Toggle switch. Controlled (`checked`/`onChange`) or uncontrolled.
 */
export function Switch({ checked, defaultChecked, onChange, label, disabled = false, id, style = {}, ...rest }) {
  const [internal, setInternal] = React.useState(!!defaultChecked);
  const isControlled = checked !== undefined;
  const on = isControlled ? checked : internal;
  const swId = id || React.useId();
  const toggle = (e) => {
    if (!isControlled) setInternal(e.target.checked);
    onChange && onChange(e);
  };
  return (
    <label htmlFor={swId} style={{ display: "inline-flex", alignItems: "center", gap: 10, cursor: disabled ? "not-allowed" : "pointer", opacity: disabled ? 0.5 : 1, ...style }}>
      <span
        style={{
          width: 38,
          height: 22,
          borderRadius: "var(--radius-pill)",
          background: on ? "var(--accent)" : "var(--gray-300)",
          position: "relative",
          flex: "none",
          transition: "background var(--dur-base) var(--ease-standard)",
        }}
      >
        <span
          style={{
            position: "absolute",
            top: 2,
            left: on ? 18 : 2,
            width: 18,
            height: 18,
            borderRadius: "50%",
            background: "#fff",
            boxShadow: "var(--shadow-sm)",
            transition: "left var(--dur-base) var(--ease-emphasized)",
          }}
        />
      </span>
      <input id={swId} type="checkbox" checked={on} disabled={disabled} onChange={toggle} style={{ position: "absolute", opacity: 0, width: 0, height: 0 }} {...rest} />
      {label && <span style={{ fontFamily: "var(--font-sans)", fontSize: 14, color: "var(--text-primary)" }}>{label}</span>}
    </label>
  );
}
