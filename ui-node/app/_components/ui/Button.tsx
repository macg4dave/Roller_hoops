'use client';

import type { ButtonHTMLAttributes, ReactNode } from 'react';

type Variant = 'default' | 'primary' | 'danger';

type Props = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: Variant;
  children: ReactNode;
};

export function Button({ variant = 'default', className, ...props }: Props) {
  const variantClass =
    variant === 'primary' ? 'btn btnPrimary' : variant === 'danger' ? 'btn btnDanger' : 'btn';

  const merged = className ? `${variantClass} ${className}` : variantClass;

  return <button {...props} className={merged} />;
}
