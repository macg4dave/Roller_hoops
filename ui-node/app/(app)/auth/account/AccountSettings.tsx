'use client';

import { FormEvent, useMemo, useState, useTransition } from 'react';

type Props = {
  username: string;
  role: string;
};

type PanelState = { status: 'idle' | 'loading' | 'success' | 'error'; message?: string };

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
    <section className="accountSection">
      <div className="detailMeta">
        <div>Signed in as <strong>{username}</strong></div>
        <div>Role: {role}</div>
        <div className="hint">{supportedNote}</div>
      </div>

      <form onSubmit={submitChangePassword} className="accountForm">
        <p className="accountFormTitle">Change password</p>
        <label className="accountField">
          Current password
          <input
            type="password"
            value={currentPassword}
            onChange={(e) => setCurrentPassword(e.target.value)}
            autoComplete="current-password"
            required
          />
        </label>
        <label className="accountField">
          New password
          <input
            type="password"
            value={newPassword}
            onChange={(e) => setNewPassword(e.target.value)}
            autoComplete="new-password"
            required
          />
        </label>
        <button type="submit" className="btn btnPrimary" disabled={isPending}>
          Update password
        </button>
      </form>

      {canAdmin ? (
        <form onSubmit={submitAdminReset} className="accountForm">
          <p className="accountFormTitle">Admin reset</p>
          <p className="hint">
            Reset a user password (or create the user if missing). Requires <code>AUTH_USERS_FILE</code>.
          </p>
          <label className="accountField">
            Username
            <input
              type="text"
              value={adminUsername}
              onChange={(e) => setAdminUsername(e.target.value)}
              required
            />
          </label>
          <label className="accountField">
            New password
            <input
              type="password"
              value={adminPassword}
              onChange={(e) => setAdminPassword(e.target.value)}
              required
            />
          </label>
          <label className="accountField">
            Role
            <select
              value={adminRole}
              onChange={(e) => setAdminRole(e.target.value)}
            >
              <option value="read-only">read-only</option>
              <option value="admin">admin</option>
            </select>
          </label>
          <button type="submit" className="btn btnPrimary" disabled={isPending}>
            Reset password
          </button>
        </form>
      ) : null}

      {state.message ? (
        <p className={`accountMessage ${state.status === 'error' ? 'accountMessageError' : 'accountMessageSuccess'}`}>
          {state.message}
        </p>
      ) : null}
    </section>
  );
}
