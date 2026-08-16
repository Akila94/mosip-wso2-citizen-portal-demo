import React from "react";

/**
 * Tooltip on hover/focus. Wraps a single child.
 */
export function Tooltip({ label, placement = "top", children, style = {} }) {
  const [show, setShow] = React.useState(false);
  const pos = {
    top:    { bottom: "calc(100% + 8px)", left: "50%", transform: "translateX(-50%)" },
    bottom: { top: "calc(100% + 8px)", left: "50%", transform: "translateX(-50%)" },
    left:   { right: "calc(100% + 8px)", top: "50%", transform: "translateY(-50%)" },
    right:  { left: "calc(100% + 8px)", top: "50%", transform: "translateY(-50%)" },
  };
  return (
    <span
      style={{ position: "relative", display: "inline-flex" }}
      onMouseEnter={() => setShow(true)}
      onMouseLeave={() => setShow(false)}
      onFocus={() => setShow(true)}
      onBlur={() => setShow(false)}
    >
      {children}
      {show && (
        <span
          role="tooltip"
          style={{
            position: "absolute",
            zIndex: 50,
            whiteSpace: "nowrap",
            background: "var(--gray-950)",
            color: "#fff",
            fontFamily: "var(--font-sans)",
            fontSize: 12,
            fontWeight: 500,
            padding: "6px 9px",
            borderRadius: "var(--radius-sm)",
            boxShadow: "var(--shadow-lg)",
            pointerEvents: "none",
            ...pos[placement],
            ...style,
          }}
        >
          {label}
        </span>
      )}
    </span>
  );
}
