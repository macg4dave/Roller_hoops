"use client";

import type { ReactNode } from 'react';

export function Field({ children }: { children: ReactNode }) {
  return <div className="field">{children}</div>;
}

export function Label({ htmlFor, children }: { htmlFor?: string; children: ReactNode }) {
  return (
    <label className="label" htmlFor={htmlFor}>
      {children}
    </label>
  );
}

export function Hint({ children }: { children: ReactNode }) {
  return <div className="hint">{children}</div>;
}
