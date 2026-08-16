import * as React from "react";

export interface IconButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  /** Required for accessibility. */
  "aria-label": string;
  variant?: "primary" | "secondary" | "ghost";
  size?: "sm" | "md" | "lg";
}

/** Square icon-only button (pass a Lucide icon as children). */
export function IconButton(props: IconButtonProps): JSX.Element;
