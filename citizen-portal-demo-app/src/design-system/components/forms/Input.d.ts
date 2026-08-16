import * as React from "react";

export interface InputProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, "size"> {
  label?: string;
  hint?: string;
  /** Error message; switches border + helper text to danger. */
  error?: string;
  size?: "sm" | "md" | "lg";
  iconLeft?: React.ReactNode;
  containerStyle?: React.CSSProperties;
}

/** Text input with label, leading icon, hint and error states. */
export function Input(props: InputProps): JSX.Element;
