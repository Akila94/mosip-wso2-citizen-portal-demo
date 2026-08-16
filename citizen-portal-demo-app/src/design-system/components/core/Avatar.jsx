import React from "react";

const SIZES = { sm: 28, md: 36, lg: 48 };
const PALETTE = ["#FF7300", "#1B58F4", "#1F9D57", "#595856", "#E0950B"];

function initials(name = "") {
  return name.trim().split(/\s+/).slice(0, 2).map((w) => w[0]).join("").toUpperCase();
}

/**
 * Circular avatar — image, or colored initials fallback.
 */
export function Avatar({ name = "", src, size = "md", style = {}, ...rest }) {
  const dim = SIZES[size] || SIZES.md;
  const color = PALETTE[(name.charCodeAt(0) || 0) % PALETTE.length];
  return (
    <span
      style={{
        width: dim,
        height: dim,
        borderRadius: "50%",
        display: "inline-flex",
        alignItems: "center",
        justifyContent: "center",
        overflow: "hidden",
        background: src ? "var(--gray-100)" : color,
        color: "#fff",
        fontFamily: "var(--font-sans)",
        fontSize: dim * 0.4,
        fontWeight: 600,
        flex: "none",
        ...style,
      }}
      {...rest}
    >
      {src ? (
        <img src={src} alt={name} style={{ width: "100%", height: "100%", objectFit: "cover" }} />
      ) : (
        initials(name)
      )}
    </span>
  );
}
