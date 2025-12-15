"use client";

import type { ReactNode } from 'react';

type Tone = 'neutral' | 'info' | 'success' | 'warning' | 'danger';

type Props = {
  tone?: Tone;
  children: ReactNode;
};

export function Badge({ tone = 'neutral', children }: Props) {
  const toneClass =
    tone === 'info'
      ? 'badge badgeInfo'
      : tone === 'success'
        ? 'badge badgeSuccess'
        : tone === 'warning'
          ? 'badge badgeWarning'
          : tone === 'danger'
            ? 'badge badgeDanger'
            : 'badge';

  return <span className={toneClass}>{children}</span>;
}
