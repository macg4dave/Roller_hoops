import { NextResponse } from 'next/server';

import { getSessionUser } from '../../../../../lib/auth/session';
import { adminResetPassword } from '../../../../../lib/auth/users-store';

export async function POST(request: Request) {
  const session = await getSessionUser();
  if (!session) {
    return NextResponse.json({ error: { code: 'unauthorized', message: 'Authentication required.' } }, { status: 401 });
  }
  if (session.role !== 'admin') {
    return NextResponse.json({ error: { code: 'forbidden', message: 'Admin role required.' } }, { status: 403 });
  }

  let body: { username?: string; new_password?: string; role?: string };
  try {
    body = await request.json();
  } catch {
    return NextResponse.json(
      { error: { code: 'validation_failed', message: 'Request body must be JSON.' } },
      { status: 400 }
    );
  }

  const username = (body.username ?? '').trim();
  const newPassword = body.new_password ?? '';
  const role = (body.role ?? '').trim();
  if (!username || !newPassword) {
    return NextResponse.json(
      { error: { code: 'validation_failed', message: 'username and new_password are required.' } },
      { status: 400 }
    );
  }
  if (newPassword.length < 8) {
    return NextResponse.json(
      { error: { code: 'validation_failed', message: 'new_password must be at least 8 characters.' } },
      { status: 400 }
    );
  }

  try {
    await adminResetPassword(username, newPassword, role || undefined);
  } catch (error) {
    const message = (error as Error)?.message ?? 'reset_failed';
    if (message === 'password_change_not_supported') {
      return NextResponse.json(
        {
          error: {
            code: 'not_supported',
            message: 'User management is disabled unless AUTH_USERS_FILE is configured.'
          }
        },
        { status: 409 }
      );
    }
    return NextResponse.json({ error: { code: 'internal_error', message: 'Password reset failed.' } }, { status: 500 });
  }

  return NextResponse.json({ ok: true });
}
