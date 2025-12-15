'use client';

import type { InputHTMLAttributes, SelectHTMLAttributes, TextareaHTMLAttributes } from 'react';

export function Input({ className, ...props }: InputHTMLAttributes<HTMLInputElement>) {
  const merged = className ? `input ${className}` : 'input';
  return <input {...props} className={merged} />;
}

export function Select({ className, ...props }: SelectHTMLAttributes<HTMLSelectElement>) {
  const merged = className ? `select ${className}` : 'select';
  return <select {...props} className={merged} />;
}

export function Textarea({ className, ...props }: TextareaHTMLAttributes<HTMLTextAreaElement>) {
  const merged = className ? `textarea ${className}` : 'textarea';
  return <textarea {...props} className={merged} />;
}
