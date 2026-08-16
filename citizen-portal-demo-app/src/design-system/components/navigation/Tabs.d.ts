import * as React from "react";

export interface TabItem { value: string; label: string; }

export interface TabsProps extends Omit<React.HTMLAttributes<HTMLDivElement>, "onChange"> {
  /** Tabs as strings or {value,label}. */
  tabs?: Array<string | TabItem>;
  /** Controlled active value. */
  value?: string;
  defaultValue?: string;
  onChange?: (value: string) => void;
}

/** Underline tab bar with orange active indicator. */
export function Tabs(props: TabsProps): JSX.Element;
