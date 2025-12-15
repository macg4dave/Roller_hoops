import type { CSSProperties } from 'react';

type SkeletonProps = {
  className?: string;
  style?: CSSProperties;
  ariaLabel?: string;
};

export function Skeleton({ className, style, ariaLabel }: SkeletonProps) {
  const label = ariaLabel ?? 'Loading';
  return <div aria-label={label} aria-busy="true" className={`skeleton${className ? ` ${className}` : ''}`} style={style} />;
}

type SkeletonLineProps = {
  width?: string;
  className?: string;
};

export function SkeletonLine({ width, className }: SkeletonLineProps) {
  return <Skeleton className={`skeletonLine${className ? ` ${className}` : ''}`} style={width ? { width } : undefined} />;
}
