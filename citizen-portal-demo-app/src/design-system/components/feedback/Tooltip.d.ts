import * as React from "react";

export interface TooltipProps {
  /** Tooltip text. */
  label: React.ReactNode;
  placement?: "top" | "bottom" | "left" | "right";
  children: React.ReactNode;
  style?: React.CSSProperties;
}

/** Hover/focus tooltip wrapping a single child. */
export function Tooltip(props: TooltipProps): JSX.Element;
