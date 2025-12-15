import { createHmac, randomUUID, timingSafeEqual } from 'crypto';
import { cookies } from 'next/headers';

import { verifyCredentials as verifyCredentialsFromStore, type AuthUser } from './users-store';

export type SessionUser = {
  username: string;
  role: string;
};

type SessionPayload = SessionUser & { expiresAt: number };

const SESSION_COOKIE_NAME = 'roller_session';
const SESSION_DURATION_SECONDS = 60 * 60 * 24; // 24 hours
const SESSION_SECRET = process.env.AUTH_SESSION_SECRET ?? 'dev-session-secret';

function makeSignature(payload: string) {
  return createHmac('sha256', SESSION_SECRET).update(payload).digest();
}

function tokenFromPayload(payload: string, signature: Buffer) {
  const encoded = `${payload}:${signature.toString('hex')}`;
  return Buffer.from(encoded, 'utf8').toString('base64');
}

function decodeToken(token: string): SessionPayload | null {
  const decoded = Buffer.from(token, 'base64').toString('utf8');
  const parts = decoded.split(':');
  if (parts.length !== 5) {
    return null;
  }
  const [username, role, expiresStr, nonce, signatureHex] = parts;
  const expiresAt = Number(expiresStr);
  if (!username || !role || Number.isNaN(expiresAt)) {
    return null;
  }
  const payload = `${username}:${role}:${expiresAt}:${nonce}`;
  const expected = makeSignature(payload);
  const actual = Buffer.from(signatureHex, 'hex');
  if (expected.length !== actual.length || !timingSafeEqual(expected, actual)) {
    return null;
  }
  return { username, role, expiresAt };
}

export function createSessionToken(user: AuthUser) {
  const expiresAt = Math.floor(Date.now() / 1000) + SESSION_DURATION_SECONDS;
  const nonce = randomUUID();
  const payload = `${user.username}:${user.role}:${expiresAt}:${nonce}`;
  const signature = makeSignature(payload);
  return tokenFromPayload(payload, signature);
}

export function verifySessionToken(token: string) {
  try {
    return decodeToken(token);
  } catch {
    return null;
  }
}

export async function getSessionUser(): Promise<SessionUser | null> {
  const store = await cookies();
  const sessionCookie = store.get(SESSION_COOKIE_NAME);
  if (!sessionCookie) {
    return null;
  }
  const token = decodeURIComponent(sessionCookie.value);
  const session = verifySessionToken(token);
  if (!session) {
    return null;
  }
  if (session.expiresAt < Math.floor(Date.now() / 1000)) {
    return null;
  }
  return { username: session.username, role: session.role };
}

function makeCookie(value: string, options: { maxAge: number; expires: string }) {
  const parts = [`${SESSION_COOKIE_NAME}=${encodeURIComponent(value)}`, `Path=/`, `HttpOnly`, `SameSite=Lax`, `Max-Age=${options.maxAge}`, `Expires=${options.expires}`];
  if (process.env.NODE_ENV === 'production') {
    parts.push('Secure');
  }
  return parts.join('; ');
}

function makeExpiredCookie() {
  const expires = new Date(0).toUTCString();
  const parts = [`${SESSION_COOKIE_NAME}=`, `Path=/`, `HttpOnly`, `SameSite=Lax`, `Max-Age=0`, `Expires=${expires}`];
  if (process.env.NODE_ENV === 'production') {
    parts.push('Secure');
  }
  return parts.join('; ');
}

export function createSessionCookie(token: string) {
  const expires = new Date(Date.now() + SESSION_DURATION_SECONDS * 1000).toUTCString();
  return makeCookie(token, { maxAge: SESSION_DURATION_SECONDS, expires });
}

export function clearSessionCookie() {
  return makeExpiredCookie();
}

export { SESSION_COOKIE_NAME, SESSION_DURATION_SECONDS };

export async function verifyCredentials(username: string, password: string): Promise<AuthUser | null> {
  return verifyCredentialsFromStore(username, password);
}
