import React from "react";

const SIZES = {
  sm: { height: 32, padding: "0 14px", fontSize: 13, gap: 6, icon: 16 },
  md: { height: 40, padding: "0 18px", fontSize: 14, gap: 8, icon: 18 },
  lg: { height: 48, padding: "0 24px", fontSize: 16, gap: 8, icon: 20 },
};

const VARIANTS = {
  primary: {
    background: "var(--accent)",
    color: "var(--accent-contrast)",
    border: "1px solid transparent",
    boxShadow: "var(--shadow-accent)",
  },
  secondary: {
    background: "var(--surface-card)",
    color: "var(--text-primary)",
    border: "1px solid var(--border-strong)",
  },
  ghost: {
    background: "transparent",
    color: "var(--text-primary)",
    border: "1px solid transparent",
  },
  inverse: {
    background: "var(--gray-0)",
    color: "var(--gray-950)",
    border: "1px solid transparent",
  },
  danger: {
    background: "var(--danger)",
    color: "#fff",
    border: "1px solid transparent",
  },
};

/**
 * Primary action button. Sentence-case labels, action verbs.
 */
export function Button({
  children,
  variant = "primary",
  size = "md",
  iconLeft,
  iconRight,
  fullWidth = false,
  disabled = false,
  loading = false,
  style = {},
  onMouseEnter,
  onMouseLeave,
  onMouseDown,
  onMouseUp,
  ...rest
}) {
  const [hover, setHover] = React.useState(false);
  const [active, setActive] = React.useState(false);
  const s = SIZES[size] || SIZES.md;
  const base = VARIANTS[variant] || VARIANTS.primary;

  const hoverStyle =
    !disabled && hover
      ? variant === "primary"
        ? { background: "var(--accent-hover)" }
        : variant === "danger"
        ? { background: "#c2302c" }
        : variant === "ghost"
        ? { background: "var(--surface-hover)" }
        : { background: "var(--surface-hover)", borderColor: "var(--border-strong)" }
      : {};

  return (
    <button
      disabled={disabled || loading}
      onMouseEnter={(e) => { setHover(true); onMouseEnter && onMouseEnter(e); }}
      onMouseLeave={(e) => { setHover(false); setActive(false); onMouseLeave && onMouseLeave(e); }}
      onMouseDown={(e) => { setActive(true); onMouseDown && onMouseDown(e); }}
      onMouseUp={(e) => { setActive(false); onMouseUp && onMouseUp(e); }}
      style={{
        display: fullWidth ? "flex" : "inline-flex",
        width: fullWidth ? "100%" : undefined,
        alignItems: "center",
        justifyContent: "center",
        gap: s.gap,
        height: s.height,
        padding: s.padding,
        fontFamily: "var(--font-sans)",
        fontSize: s.fontSize,
        fontWeight: 600,
        letterSpacing: "-0.005em",
        borderRadius: "var(--radius-md)",
        cursor: disabled || loading ? "not-allowed" : "pointer",
        opacity: disabled ? 0.45 : 1,
        transform: active && !disabled ? "translateY(1px) scale(0.99)" : "none",
        transition: "background var(--dur-fast) var(--ease-standard), transform var(--dur-fast) var(--ease-standard), box-shadow var(--dur-base) var(--ease-standard)",
        whiteSpace: "nowrap",
        ...base,
        ...hoverStyle,
        ...style,
      }}
      {...rest}
    >
      {loading && <Spinner size={s.icon} />}
      {!loading && iconLeft}
      {children}
      {!loading && iconRight}
    </button>
  );
}

function Spinner({ size = 16 }) {
  return (
    <span
      style={{
        width: size,
        height: size,
        border: "2px solid currentColor",
        borderTopColor: "transparent",
        borderRadius: "50%",
        display: "inline-block",
        animation: "wso2-spin 0.7s linear infinite",
      }}
    />
  );
}

if (typeof document !== "undefined" && !document.getElementById("wso2-spin-kf")) {
  const st = document.createElement("style");
  st.id = "wso2-spin-kf";
  st.textContent = "@keyframes wso2-spin{to{transform:rotate(360deg)}}";
  document.head.appendChild(st);
}
