"use client";

import type { ReactNode } from 'react';

type Tone = 'info' | 'success' | 'warning' | 'danger';

type Props = {
  tone: Tone;
  children: ReactNode;
};

export function Alert({ tone, children }: Props) {
  const toneClass =
    tone === 'success'
      ? 'alert alertSuccess'
      : tone === 'warning'
        ? 'alert alertWarning'
        : tone === 'danger'
          ? 'alert alertDanger'
          : 'alert alertInfo';

  return <div className={toneClass}>{children}</div>;
}
