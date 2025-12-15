'use client';

import { useRouter } from 'next/navigation';
import { FormEvent, useState, useTransition } from 'react';

export function LoginForm() {
  const router = useRouter();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [isPending, startTransition] = useTransition();

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError(null);
    startTransition(async () => {
      const response = await fetch('/api/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password })
      });
      if (response.ok) {
        router.replace('/devices');
        return;
      }
      const payload = (await response.json().catch(() => null)) as { error?: { message?: string } } | null;
      setError(payload?.error?.message ?? 'Invalid username or password.');
    });
  };

  return (
    <form onSubmit={handleSubmit} style={{ display: 'grid', gap: 12 }}>
      <label style={{ display: 'grid', gap: 4, fontWeight: 600, fontSize: 14 }} htmlFor="auth-username">
        Username
        <input
          id="auth-username"
          type="text"
          value={username}
          onChange={(event) => setUsername(event.target.value)}
          placeholder="admin"
          autoComplete="username"
          style={{ borderRadius: 6, border: '1px solid #d1d5db', padding: '10px 12px' }}
          required
        />
      </label>
      <label style={{ display: 'grid', gap: 4, fontWeight: 600, fontSize: 14 }} htmlFor="auth-password">
        Password
        <input
          id="auth-password"
          type="password"
          value={password}
          onChange={(event) => setPassword(event.target.value)}
          autoComplete="current-password"
          style={{ borderRadius: 6, border: '1px solid #d1d5db', padding: '10px 12px' }}
          required
        />
      </label>
      {error ? <p style={{ color: '#b91c1c', margin: 0 }}>{error}</p> : null}
      <button
        type="submit"
        disabled={isPending}
        style={{
          borderRadius: 8,
          padding: '10px 14px',
          border: 'none',
          background: '#111827',
          color: '#fff',
          fontWeight: 600,
          cursor: 'pointer'
        }}
      >
        {isPending ? 'Signing in…' : 'Sign in'}
      </button>
    </form>
  );
}