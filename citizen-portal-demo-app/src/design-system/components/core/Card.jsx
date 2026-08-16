import React from "react";

/**
 * Surface container. White, subtle border, soft shadow, 12–16px radius.
 * Set `interactive` for hover lift, `padded` toggles inner padding.
 */
export function Card({
  children,
  interactive = false,
  padded = true,
  style = {},
  onMouseEnter,
  onMouseLeave,
  ...rest
}) {
  const [hover, setHover] = React.useState(false);
  return (
    <div
      onMouseEnter={(e) => { setHover(true); onMouseEnter && onMouseEnter(e); }}
      onMouseLeave={(e) => { setHover(false); onMouseLeave && onMouseLeave(e); }}
      style={{
        background: "var(--surface-card)",
        border: "1px solid var(--border-default)",
        borderRadius: "var(--radius-xl)",
        boxShadow: interactive && hover ? "var(--shadow-md)" : "var(--shadow-sm)",
        borderColor: interactive && hover ? "var(--border-strong)" : "var(--border-default)",
        padding: padded ? "var(--space-6)" : 0,
        transition: "box-shadow var(--dur-base) var(--ease-standard), border-color var(--dur-base) var(--ease-standard), transform var(--dur-base) var(--ease-standard)",
        transform: interactive && hover ? "translateY(-2px)" : "none",
        cursor: interactive ? "pointer" : "default",
        ...style,
      }}
      {...rest}
    >
      {children}
    </div>
  );
}
