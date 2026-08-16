import * as React from "react";

export interface LogoProps extends React.HTMLAttributes<HTMLSpanElement> {
  /** Pixel height of the wordmark. Default 28. */
  size?: number;
  /** Color treatment. Default "auto". */
  tone?: "auto" | "light" | "mono";
  /** Render the "2" in signature orange. Default true. */
  accent?: boolean;
  /** Show the trailing orange accent dot. Default false. */
  showAccentDot?: boolean;
}

/**
 * WSO2 wordmark (type-recreated).
 */
export function Logo(props: LogoProps): JSX.Element;
