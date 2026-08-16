import * as React from "react";

/**
 * @startingPoint section="Core" subtitle="Primary, secondary, ghost & danger buttons" viewport="700x150"
 */
export interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  /** Visual style. Default "primary". */
  variant?: "primary" | "secondary" | "ghost" | "inverse" | "danger";
  /** Size. Default "md". */
  size?: "sm" | "md" | "lg";
  /** Icon node rendered before the label. */
  iconLeft?: React.ReactNode;
  /** Icon node rendered after the label. */
  iconRight?: React.ReactNode;
  /** Stretch to container width. */
  fullWidth?: boolean;
  /** Show a spinner and block interaction. */
  loading?: boolean;
}

/**
 * WSO2 button. Primary = signature orange with glow; secondary = outlined;
 * ghost = quiet; inverse = white on dark; danger = red.
 */
export function Button(props: ButtonProps): JSX.Element;
