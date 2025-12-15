'use client';

import { useRouter } from 'next/navigation';
import { useCallback, useState } from 'react';

export function LogoutButton() {
  const router = useRouter();
  const [isPending, setIsPending] = useState(false);

  const handleLogout = useCallback(async () => {
    setIsPending(true);
    await fetch('/api/auth/logout', { method: 'POST' });
    router.replace('/auth/login');
  }, [router]);

  return (
    <button
      type="button"
      onClick={handleLogout}
      disabled={isPending}
      style={{
        borderRadius: 8,
        padding: '6px 12px',
        border: '1px solid #d1d5db',
        background: '#fff',
        color: '#111827',
        fontWeight: 600,
        cursor: 'pointer'
      }}
    >
      {isPending ? 'Signing out…' : 'Sign out'}
    </button>
  );
}