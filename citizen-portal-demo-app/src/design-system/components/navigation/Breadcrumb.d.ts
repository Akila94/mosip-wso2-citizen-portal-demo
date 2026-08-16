import * as React from "react";

export interface BreadcrumbItem { label: string; href?: string; }

export interface BreadcrumbProps extends React.HTMLAttributes<HTMLElement> {
  /** Trail items as strings or {label,href}; last renders as current. */
  items?: Array<string | BreadcrumbItem>;
}

/** Slash-separated breadcrumb trail. */
export function Breadcrumb(props: BreadcrumbProps): JSX.Element;
