import * as React from "react";

export interface SelectOption { value: string; label: string; }

export interface SelectProps extends Omit<React.SelectHTMLAttributes<HTMLSelectElement>, "size"> {
  label?: string;
  /** Options as strings or {value,label}. */
  options?: Array<string | SelectOption>;
  size?: "sm" | "md" | "lg";
  containerStyle?: React.CSSProperties;
}

/** Styled native select matching the input system. */
export function Select(props: SelectProps): JSX.Element;
