import * as React from "react";

export interface SwitchProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, "size" | "type"> {
  label?: React.ReactNode;
}

/** Toggle switch; controlled or uncontrolled. */
export function Switch(props: SwitchProps): JSX.Element;
