import * as React from "react";

export interface CardProps extends React.HTMLAttributes<HTMLDivElement> {
  /** Hover lift + stronger border + shadow. */
  interactive?: boolean;
  /** Toggle inner padding (default true). */
  padded?: boolean;
}

/** White surface container with subtle border and soft shadow. */
export function Card(props: CardProps): JSX.Element;
