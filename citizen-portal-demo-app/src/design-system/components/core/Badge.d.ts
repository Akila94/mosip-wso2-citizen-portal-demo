import * as React from "react";

export interface BadgeProps extends React.HTMLAttributes<HTMLSpanElement> {
  tone?: "neutral" | "brand" | "success" | "warning" | "danger" | "info" | "solid";
  /** Show a leading status dot. */
  dot?: boolean;
}

/** Small status / category pill. */
export function Badge(props: BadgeProps): JSX.Element;
