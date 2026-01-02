import { afterEach, describe, expect, it } from 'vitest';

import { clearSessionCookie, createSessionCookie } from './session';

const ORIGINAL_ENV = { ...process.env };

function resetEnv() {
  for (const key of Object.keys(process.env)) {
    delete process.env[key];
  }
  Object.assign(process.env, ORIGINAL_ENV);
}

afterEach(() => {
  resetEnv();
});

describe('session cookies', () => {
  it('omits Secure for http requests (local dev)', () => {
    process.env.NODE_ENV = 'production';
    delete process.env.AUTH_COOKIE_SECURE;

    const request = new Request('http://localhost/api/auth/login');
    const cookie = createSessionCookie('token', request);
    expect(cookie).not.toContain('Secure');
  });

  it('sets Secure when x-forwarded-proto is https', () => {
    process.env.NODE_ENV = 'production';
    delete process.env.AUTH_COOKIE_SECURE;

    const request = new Request('http://localhost/api/auth/login', {
      headers: { 'x-forwarded-proto': 'https' }
    });
    const cookie = createSessionCookie('token', request);
    expect(cookie).toContain('Secure');
  });

  it('allows forcing Secure via AUTH_COOKIE_SECURE=true', () => {
    process.env.NODE_ENV = 'production';
    process.env.AUTH_COOKIE_SECURE = 'true';

    const request = new Request('http://localhost/api/auth/login', {
      headers: { 'x-forwarded-proto': 'http' }
    });
    const cookie = createSessionCookie('token', request);
    expect(cookie).toContain('Secure');
  });

  it('allows disabling Secure via AUTH_COOKIE_SECURE=false', () => {
    process.env.NODE_ENV = 'production';
    process.env.AUTH_COOKIE_SECURE = 'false';

    const request = new Request('https://example.test/api/auth/login', {
      headers: { 'x-forwarded-proto': 'https' }
    });
    const cookie = createSessionCookie('token', request);
    expect(cookie).not.toContain('Secure');
  });

  it('matches Secure behaviour when clearing cookies', () => {
    process.env.NODE_ENV = 'production';
    delete process.env.AUTH_COOKIE_SECURE;

    const request = new Request('https://example.test/api/auth/logout', {
      headers: { 'x-forwarded-proto': 'https' }
    });
    const cookie = clearSessionCookie(request);
    expect(cookie).toContain('Max-Age=0');
    expect(cookie).toContain('Secure');
  });
});

