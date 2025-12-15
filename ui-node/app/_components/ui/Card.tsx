"use client";

import type { HTMLAttributes, ReactNode } from 'react';

type Props = HTMLAttributes<HTMLDivElement> & {
  children: ReactNode;
};

export function Card({ className, ...props }: Props) {
  const merged = className ? `card ${className}` : 'card';
  return <div {...props} className={merged} />;
}

export function CardBody({ className, ...props }: Props) {
  const merged = className ? `cardPad ${className}` : 'cardPad';
  return <div {...props} className={merged} />;
}
