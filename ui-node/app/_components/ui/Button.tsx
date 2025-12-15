
'use client';

import { forwardRef } from 'react';
import type { ButtonHTMLAttributes, ReactNode } from 'react';

type Variant = 'default' | 'primary' | 'danger';

type Props = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: Variant;
  children: ReactNode;
};

export const Button = forwardRef<HTMLButtonElement, Props>(function Button(
  { variant = 'default', className, ...props },
  ref
) {
  const variantClass =
    variant === 'primary' ? 'btn btnPrimary' : variant === 'danger' ? 'btn btnDanger' : 'btn';

  const merged = className ? `${variantClass} ${className}` : variantClass;

  return <button ref={ref} {...props} className={merged} />;
});
