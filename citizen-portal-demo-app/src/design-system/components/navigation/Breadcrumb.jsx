import React from "react";

/**
 * Breadcrumb trail. `items` = array of { label, href } or strings.
 * Last item renders as current (non-link).
 */
export function Breadcrumb({ items = [], style = {}, ...rest }) {
  const norm = items.map((i) => (typeof i === "string" ? { label: i } : i));
  return (
    <nav
      aria-label="Breadcrumb"
      style={{ display: "flex", alignItems: "center", flexWrap: "wrap", gap: 8, fontFamily: "var(--font-sans)", fontSize: 13, ...style }}
      {...rest}
    >
      {norm.map((item, i) => {
        const last = i === norm.length - 1;
        return (
          <React.Fragment key={i}>
            {last ? (
              <span style={{ color: "var(--text-primary)", fontWeight: 600 }}>{item.label}</span>
            ) : (
              <a href={item.href || "#"} style={{ color: "var(--text-secondary)", textDecoration: "none" }}>{item.label}</a>
            )}
            {!last && <span style={{ color: "var(--text-muted)" }}>/</span>}
          </React.Fragment>
        );
      })}
    </nav>
  );
}
