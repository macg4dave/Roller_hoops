'use client';

import { FormEvent, useMemo, useState, useTransition } from 'react';

type Props = {
  username: string;
  role: string;
};

type PanelState = { status: 'idle' | 'loading' | 'success' | 'error'; message?: string };

function messageStyle(state: PanelState) {
  if (state.status === 'error') {
    return { background: '#f9d7da', color: '#b00020' };
  }
  if (state.status === 'success') {
    return { background: '#d1e7dd', color: '#0f5132' };
  }
  return { background: '#eef2ff', color: '#1e3a8a' };
}

export function AccountSettings({ username, role }: Props) {
  const [state, setState] = useState<PanelState>({ status: 'idle' });
  const [isPending, startTransition] = useTransition();

  const [currentPassword, setCurrentPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [adminUsername, setAdminUsername] = useState('');
  const [adminPassword, setAdminPassword] = useState('');
  const [adminRole, setAdminRole] = useState('read-only');

  const canAdmin = role === 'admin';

  const supportedNote = useMemo(() => {
    return 'Password changes require `AUTH_USERS_FILE` to be configured for the UI service.';
  }, []);

  const submitChangePassword = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setState({ status: 'loading', message: 'Updating password…' });
    startTransition(async () => {
      const res = await fetch('/api/auth/change-password', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ current_password: currentPassword, new_password: newPassword })
      });
      if (!res.ok) {
        const payload = (await res.json().catch(() => null)) as { error?: { message?: string } } | null;
        setState({ status: 'error', message: payload?.error?.message ?? `Password change failed (${res.status})` });
        return;
      }
      setCurrentPassword('');
      setNewPassword('');
      setState({ status: 'success', message: 'Password updated.' });
    });
  };

  const submitAdminReset = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setState({ status: 'loading', message: 'Resetting password…' });
    startTransition(async () => {
      const res = await fetch('/api/auth/admin/reset-password', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username: adminUsername, new_password: adminPassword, role: adminRole })
      });
      if (!res.ok) {
        const payload = (await res.json().catch(() => null)) as { error?: { message?: string } } | null;
        setState({ status: 'error', message: payload?.error?.message ?? `Password reset failed (${res.status})` });
        return;
      }
      setAdminPassword('');
      setState({ status: 'success', message: 'Password reset.' });
    });
  };

  return (
    <section style={{ display: 'grid', gap: 18, maxWidth: 520 }}>
      <div style={{ display: 'grid', gap: 4 }}>
        <div style={{ fontSize: 14, color: '#111827' }}>Signed in as {username}</div>
        <div style={{ fontSize: 12, color: '#4b5563' }}>Role: {role}</div>
        <div style={{ fontSize: 12, color: '#6b7280' }}>{supportedNote}</div>
      </div>

      <form onSubmit={submitChangePassword} style={{ border: '1px solid #e5e7eb', borderRadius: 10, padding: 16, display: 'grid', gap: 10 }}>
        <div style={{ fontSize: 12, letterSpacing: '0.05em', textTransform: 'uppercase', color: '#4b5563' }}>
          Change password
        </div>
        <label style={{ display: 'grid', gap: 6, fontWeight: 600, fontSize: 13 }}>
          Current password
          <input
            type="password"
            value={currentPassword}
            onChange={(e) => setCurrentPassword(e.target.value)}
            autoComplete="current-password"
            style={{ padding: '10px 12px', borderRadius: 8, border: '1px solid #d1d5db' }}
            required
          />
        </label>
        <label style={{ display: 'grid', gap: 6, fontWeight: 600, fontSize: 13 }}>
          New password
          <input
            type="password"
            value={newPassword}
            onChange={(e) => setNewPassword(e.target.value)}
            autoComplete="new-password"
            style={{ padding: '10px 12px', borderRadius: 8, border: '1px solid #d1d5db' }}
            required
          />
        </label>
        <button
          type="submit"
          disabled={isPending}
          style={{ borderRadius: 8, padding: '10px 14px', border: 'none', background: '#111827', color: '#fff', fontWeight: 700, cursor: 'pointer', width: 'fit-content' }}
        >
          Update password
        </button>
      </form>

      {canAdmin ? (
        <form onSubmit={submitAdminReset} style={{ border: '1px solid #e5e7eb', borderRadius: 10, padding: 16, display: 'grid', gap: 10 }}>
          <div style={{ fontSize: 12, letterSpacing: '0.05em', textTransform: 'uppercase', color: '#4b5563' }}>
            Admin reset
          </div>
          <div style={{ fontSize: 13, color: '#374151' }}>
            Reset a user password (or create the user if missing). Requires `AUTH_USERS_FILE`.
          </div>
          <label style={{ display: 'grid', gap: 6, fontWeight: 600, fontSize: 13 }}>
            Username
            <input
              type="text"
              value={adminUsername}
              onChange={(e) => setAdminUsername(e.target.value)}
              style={{ padding: '10px 12px', borderRadius: 8, border: '1px solid #d1d5db' }}
              required
            />
          </label>
          <label style={{ display: 'grid', gap: 6, fontWeight: 600, fontSize: 13 }}>
            New password
            <input
              type="password"
              value={adminPassword}
              onChange={(e) => setAdminPassword(e.target.value)}
              style={{ padding: '10px 12px', borderRadius: 8, border: '1px solid #d1d5db' }}
              required
            />
          </label>
          <label style={{ display: 'grid', gap: 6, fontWeight: 600, fontSize: 13 }}>
            Role
            <select
              value={adminRole}
              onChange={(e) => setAdminRole(e.target.value)}
              style={{ padding: '10px 12px', borderRadius: 8, border: '1px solid #d1d5db' }}
            >
              <option value="read-only">read-only</option>
              <option value="admin">admin</option>
            </select>
          </label>
          <button
            type="submit"
            disabled={isPending}
            style={{ borderRadius: 8, padding: '10px 14px', border: 'none', background: '#111827', color: '#fff', fontWeight: 700, cursor: 'pointer', width: 'fit-content' }}
          >
            Reset password
          </button>
        </form>
      ) : null}

      {state.message ? (
        <p style={{ margin: 0, padding: '8px 10px', borderRadius: 6, fontWeight: 600, ...messageStyle(state) }}>
          {state.message}
        </p>
      ) : null}
    </section>
  );
}

