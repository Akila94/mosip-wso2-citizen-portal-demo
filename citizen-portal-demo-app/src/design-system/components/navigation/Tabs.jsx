import React from "react";

/**
 * Underline tabs. Controlled via `value`/`onChange` or uncontrolled.
 * `tabs` is an array of strings or { value, label }.
 */
export function Tabs({ tabs = [], value, defaultValue, onChange, style = {}, ...rest }) {
  const norm = tabs.map((t) => (typeof t === "string" ? { value: t, label: t } : t));
  const [internal, setInternal] = React.useState(defaultValue ?? norm[0]?.value);
  const isControlled = value !== undefined;
  const active = isControlled ? value : internal;
  const [hover, setHover] = React.useState(null);
  const select = (v) => {
    if (!isControlled) setInternal(v);
    onChange && onChange(v);
  };
  return (
    <div
      role="tablist"
      style={{
        display: "flex",
        gap: 4,
        borderBottom: "1px solid var(--border-default)",
        ...style,
      }}
      {...rest}
    >
      {norm.map((t) => {
        const on = t.value === active;
        return (
          <button
            key={t.value}
            role="tab"
            aria-selected={on}
            onClick={() => select(t.value)}
            onMouseEnter={() => setHover(t.value)}
            onMouseLeave={() => setHover(null)}
            style={{
              appearance: "none",
              background: "none",
              border: "none",
              cursor: "pointer",
              padding: "10px 14px",
              marginBottom: -1,
              fontFamily: "var(--font-sans)",
              fontSize: 14,
              fontWeight: on ? 600 : 500,
              color: on ? "var(--text-primary)" : hover === t.value ? "var(--text-primary)" : "var(--text-secondary)",
              borderBottom: `2px solid ${on ? "var(--accent)" : "transparent"}`,
              transition: "color var(--dur-fast) var(--ease-standard), border-color var(--dur-fast) var(--ease-standard)",
            }}
          >
            {t.label}
          </button>
        );
      })}
    </div>
  );
}
