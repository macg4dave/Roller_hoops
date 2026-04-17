import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('../../../../../lib/auth/session', () => ({
  getSessionUser: vi.fn()
}));

vi.mock('../../../../../lib/auth/users-store', async () => {
  const actual = await vi.importActual<typeof import('../../../../../lib/auth/users-store')>(
    '../../../../../lib/auth/users-store'
  );
  return {
    ...actual,
    adminResetPassword: vi.fn()
  };
});

import { getSessionUser } from '../../../../../lib/auth/session';
import { adminResetPassword } from '../../../../../lib/auth/users-store';
import { POST } from './route';

describe('admin reset-password API route', () => {
  beforeEach(() => {
    vi.mocked(getSessionUser).mockReset();
    vi.mocked(adminResetPassword).mockReset();
  });

  it('rejects unknown roles instead of creating writable non-admin users', async () => {
    vi.mocked(getSessionUser).mockResolvedValue({ username: 'admin', role: 'admin' });

    const response = await POST(
      new Request('http://localhost/api/auth/admin/reset-password', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username: 'alice', new_password: 'password123', role: 'owner' })
      })
    );

    expect(response.status).toBe(400);
    await expect(response.json()).resolves.toEqual({
      error: { code: 'validation_failed', message: 'role must be admin or read-only.' }
    });
    expect(adminResetPassword).not.toHaveBeenCalled();
  });
});
