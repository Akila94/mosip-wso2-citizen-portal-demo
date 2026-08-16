import React from "react";

/**
 * WSO2 wordmark. Type-recreated in Inter Extrabold (lowercase "wso2")
 * since only .ai source artwork was supplied. Swap for official SVG
 * when available.
 */
export function Logo({
  size = 28,
  tone = "auto",      // "auto" (dark text) | "light" (white) | "mono"
  accent = true,      // color the "2" orange
  showAccentDot = false,
  style = {},
  ...rest
}) {
  const color = tone === "light" ? "#ffffff" : "var(--gray-950)";
  const accentColor = tone === "mono" || !accent ? color : "var(--orange-500)";
  return (
    <span
      style={{
        fontFamily: "var(--font-display)",
        fontWeight: 800,
        fontSize: size,
        lineHeight: 1,
        letterSpacing: "-0.04em",
        color,
        display: "inline-flex",
        alignItems: "center",
        userSelect: "none",
        ...style,
      }}
      {...rest}
    >
      wso<span style={{ color: accentColor }}>2</span>
      {showAccentDot && (
        <span
          style={{
            width: size * 0.16,
            height: size * 0.16,
            borderRadius: "50%",
            background: "var(--orange-500)",
            marginLeft: size * 0.12,
            alignSelf: "flex-end",
            marginBottom: size * 0.08,
          }}
        />
      )}
    </span>
  );
}
