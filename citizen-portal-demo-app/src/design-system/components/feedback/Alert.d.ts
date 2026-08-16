import * as React from "react";

export interface AlertProps extends React.HTMLAttributes<HTMLDivElement> {
  tone?: "info" | "success" | "warning" | "danger";
  title?: React.ReactNode;
  /** Optional dismiss button. */
  onClose?: () => void;
}

/** Inline alert / callout banner. */
export function Alert(props: AlertProps): JSX.Element;
