import * as React from "react";

export interface CheckboxProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, "size" | "type"> {
  label?: React.ReactNode;
}

/** Checkbox with label; controlled or uncontrolled. */
export function Checkbox(props: CheckboxProps): JSX.Element;
