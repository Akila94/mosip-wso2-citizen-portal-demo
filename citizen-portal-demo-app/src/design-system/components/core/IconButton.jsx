import React from "react";

const SIZES = { sm: 32, md: 40, lg: 48 };

/**
 * Square icon-only button. Pass a Lucide icon as children.
 */
export function IconButton({
  children,
  variant = "ghost",
  size = "md",
  disabled = false,
  "aria-label": ariaLabel,
  style = {},
  ...rest
}) {
  const [hover, setHover] = React.useState(false);
  const dim = SIZES[size] || SIZES.md;
  const variants = {
    primary: { background: "var(--accent)", color: "#fff", border: "1px solid transparent" },
    secondary: { background: "var(--surface-card)", color: "var(--text-primary)", border: "1px solid var(--border-strong)" },
    ghost: { background: hover ? "var(--surface-hover)" : "transparent", color: "var(--text-secondary)", border: "1px solid transparent" },
  };
  const hoverP = hover && variant === "primary" ? { background: "var(--accent-hover)" } : {};
  return (
    <button
      aria-label={ariaLabel}
      disabled={disabled}
      onMouseEnter={() => setHover(true)}
      onMouseLeave={() => setHover(false)}
      style={{
        width: dim,
        height: dim,
        display: "inline-flex",
        alignItems: "center",
        justifyContent: "center",
        borderRadius: "var(--radius-md)",
        cursor: disabled ? "not-allowed" : "pointer",
        opacity: disabled ? 0.45 : 1,
        transition: "background var(--dur-fast) var(--ease-standard)",
        ...(variants[variant] || variants.ghost),
        ...hoverP,
        ...style,
      }}
      {...rest}
    >
      {children}
    </button>
  );
}
