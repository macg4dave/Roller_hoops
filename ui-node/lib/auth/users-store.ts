import { readFile, writeFile } from 'fs/promises';

import { hashPasswordScrypt, verifyPasswordScrypt } from './password';

export type AuthUser = {
  username: string;
  role: string;
  password?: string;
  password_scrypt?: string;
};

const AUTH_USERS_FILE = (process.env.AUTH_USERS_FILE ?? '').trim();

export async function loadAuthUsers(): Promise<{ users: AuthUser[]; writable: boolean }> {
  if (AUTH_USERS_FILE) {
    const raw = await readFile(AUTH_USERS_FILE, 'utf8').catch(() => '');
    if (raw.trim() === '') {
      return { users: envUsers(), writable: true };
    }
    const parsed = JSON.parse(raw) as unknown;
    if (!Array.isArray(parsed)) {
      throw new Error('AUTH_USERS_FILE must contain a JSON array');
    }
    const users = parsed
      .map((u) => normalizeUser(u))
      .filter((u): u is AuthUser => u != null);
    return { users, writable: true };
  }
  return { users: envUsers(), writable: false };
}

export async function verifyCredentials(username: string, password: string): Promise<AuthUser | null> {
  const normalized = username.trim();
  const { users } = await loadAuthUsers();
  const found = users.find((u) => u.username === normalized);
  if (!found) {
    return null;
  }
  if (typeof found.password_scrypt === 'string' && found.password_scrypt.startsWith('scrypt$')) {
    return verifyPasswordScrypt(password, found.password_scrypt) ? found : null;
  }
  return found.password === password ? found : null;
}

export async function changePassword(username: string, currentPassword: string, newPassword: string) {
  const { users, writable } = await loadAuthUsers();
  if (!writable) {
    throw new Error('password_change_not_supported');
  }
  const normalized = username.trim();
  const idx = users.findIndex((u) => u.username === normalized);
  if (idx < 0) {
    throw new Error('not_found');
  }
  const user = users[idx];
  const verified = await verifyCredentials(username, currentPassword);
  if (!verified) {
    throw new Error('invalid_password');
  }
  users[idx] = { ...user, password: undefined, password_scrypt: hashPasswordScrypt(newPassword) };
  await persistUsers(users);
}

export async function adminResetPassword(username: string, newPassword: string, role?: string) {
  const { users, writable } = await loadAuthUsers();
  if (!writable) {
    throw new Error('password_change_not_supported');
  }
  const normalized = username.trim();
  const idx = users.findIndex((u) => u.username === normalized);
  const nextRole = (role ?? '').trim();
  if (idx < 0) {
    const created: AuthUser = {
      username: normalized,
      role: nextRole || 'read-only',
      password_scrypt: hashPasswordScrypt(newPassword)
    };
    users.push(created);
  } else {
    users[idx] = {
      ...users[idx],
      role: nextRole || users[idx].role || 'read-only',
      password: undefined,
      password_scrypt: hashPasswordScrypt(newPassword)
    };
  }
  await persistUsers(users);
}

async function persistUsers(users: AuthUser[]) {
  if (!AUTH_USERS_FILE) {
    throw new Error('password_change_not_supported');
  }
  const payload = JSON.stringify(users, null, 2);
  await writeFile(AUTH_USERS_FILE, payload, 'utf8');
}

function envUsers(): AuthUser[] {
  const rawUsers = (process.env.AUTH_USERS ?? '').trim();
  if (rawUsers) {
    return rawUsers
      .split(',')
      .map((raw) => raw.trim())
      .filter(Boolean)
      .map((raw) => {
        const [username, password, role] = raw.split(':');
        return {
          username: (username ?? '').trim(),
          password: (password ?? '').trim(),
          role: (role ?? 'admin').trim() || 'admin',
          password_scrypt: undefined
        } satisfies AuthUser;
      })
      .filter((u) => u.username && (u.password || u.password_scrypt));
  }

  const username = (process.env.AUTH_USERNAME ?? 'admin').trim();
  const password = (process.env.AUTH_PASSWORD ?? 'admin').trim();
  const role = (process.env.AUTH_ROLE ?? 'admin').trim() || 'admin';
  return [{ username, password, role }];
}

function normalizeUser(value: unknown): AuthUser | null {
  if (!value || typeof value !== 'object') {
    return null;
  }
  const candidate = value as Partial<AuthUser>;
  const username = String(candidate.username ?? '').trim();
  const role = String(candidate.role ?? 'read-only').trim() || 'read-only';
  const password = candidate.password != null ? String(candidate.password) : undefined;
  const passwordScrypt = candidate.password_scrypt != null ? String(candidate.password_scrypt) : undefined;
  if (!username) {
    return null;
  }
  if (!password && !passwordScrypt) {
    return null;
  }
  return { username, role, password, password_scrypt: passwordScrypt };
}
