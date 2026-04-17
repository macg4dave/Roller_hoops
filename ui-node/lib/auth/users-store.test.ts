import { afterEach, describe, expect, it, vi } from 'vitest';

const ORIGINAL_ENV = { ...process.env };

function resetEnv() {
  for (const key of Object.keys(process.env)) {
    delete process.env[key];
  }
  Object.assign(process.env, ORIGINAL_ENV);
  delete process.env.AUTH_USERS_FILE;
}

async function loadAuthUsersWithEnv(env: Record<string, string | undefined>) {
  resetEnv();
  Object.assign(process.env, env);
  vi.resetModules();
  const { loadAuthUsers } = await import('./users-store');
  return loadAuthUsers();
}

afterEach(() => {
  resetEnv();
  vi.resetModules();
});

describe('auth user role parsing', () => {
  it('defaults missing AUTH_USERS roles to admin for quickstart compatibility', async () => {
    const { users } = await loadAuthUsersWithEnv({ AUTH_USERS: 'alice:secret' });

    expect(users).toMatchObject([{ username: 'alice', role: 'admin' }]);
  });

  it('normalizes unknown explicit AUTH_USERS roles to read-only', async () => {
    const { users } = await loadAuthUsersWithEnv({
      AUTH_USERS: 'alice:secret:owner,bob:secret:read-only,carol:secret:admin'
    });

    expect(users.map((user) => ({ username: user.username, role: user.role }))).toEqual([
      { username: 'alice', role: 'read-only' },
      { username: 'bob', role: 'read-only' },
      { username: 'carol', role: 'admin' }
    ]);
  });

  it('normalizes unknown AUTH_ROLE values to read-only', async () => {
    const { users } = await loadAuthUsersWithEnv({
      AUTH_USERNAME: 'alice',
      AUTH_PASSWORD: 'secret',
      AUTH_ROLE: 'owner'
    });

    expect(users).toMatchObject([{ username: 'alice', role: 'read-only' }]);
  });
});
