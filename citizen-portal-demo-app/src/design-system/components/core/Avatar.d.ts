import * as React from "react";

export interface AvatarProps extends React.HTMLAttributes<HTMLSpanElement> {
  /** Used for initials fallback and alt text. */
  name?: string;
  /** Image URL; falls back to colored initials when absent. */
  src?: string;
  size?: "sm" | "md" | "lg";
}

/** Circular avatar with colored-initials fallback. */
export function Avatar(props: AvatarProps): JSX.Element;
