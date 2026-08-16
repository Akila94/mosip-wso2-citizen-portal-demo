import React from "react";

/**
 * Tag / chip — for filters, metadata, removable selections.
 */
export function Tag({ children, onRemove, icon, style = {}, ...rest }) {
  const [hover, setHover] = React.useState(false);
  return (
    <span
      style={{
        display: "inline-flex",
        alignItems: "center",
        gap: 6,
        height: 28,
        padding: onRemove ? "0 6px 0 12px" : "0 12px",
        background: "var(--surface-card)",
        color: "var(--text-primary)",
        border: "1px solid var(--border-default)",
        fontFamily: "var(--font-sans)",
        fontSize: 13,
        fontWeight: 500,
        borderRadius: "var(--radius-pill)",
        whiteSpace: "nowrap",
        ...style,
      }}
      {...rest}
    >
      {icon}
      {children}
      {onRemove && (
        <button
          aria-label="Remove"
          onClick={onRemove}
          onMouseEnter={() => setHover(true)}
          onMouseLeave={() => setHover(false)}
          style={{
            width: 18,
            height: 18,
            display: "inline-flex",
            alignItems: "center",
            justifyContent: "center",
            border: "none",
            cursor: "pointer",
            borderRadius: "50%",
            background: hover ? "var(--gray-200)" : "transparent",
            color: "var(--text-secondary)",
            fontSize: 14,
            lineHeight: 1,
            padding: 0,
          }}
        >
          ×
        </button>
      )}
    </span>
  );
}
