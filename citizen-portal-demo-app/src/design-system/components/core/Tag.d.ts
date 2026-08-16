import * as React from "react";

export interface TagProps extends React.HTMLAttributes<HTMLSpanElement> {
  /** Optional leading icon node. */
  icon?: React.ReactNode;
  /** When provided, shows a removable × button. */
  onRemove?: () => void;
}

/** Outlined chip for filters, metadata, removable selections. */
export function Tag(props: TagProps): JSX.Element;
