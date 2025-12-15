"use client";

import type { ReactNode } from 'react';

export function EmptyState({ title, children }: { title: string; children?: ReactNode }) {
  return (
    <div className="emptyState">
      <div className="emptyStateTitle">{title}</div>
      {children ? <div className="emptyStateBody">{children}</div> : null}
    </div>
  );
}
