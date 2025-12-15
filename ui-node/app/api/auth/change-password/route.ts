import { NextResponse } from 'next/server';

import { createSessionCookie, createSessionToken, getSessionUser } from '../../../../lib/auth/session';
import { changePassword } from '../../../../lib/auth/users-store';

export async function POST(request: Request) {
  const session = await getSessionUser();
  if (!session) {
    return NextResponse.json({ error: { code: 'unauthorized', message: 'Authentication required.' } }, { status: 401 });
  }

  let body: { current_password?: string; new_password?: string };
  try {
    body = await request.json();
  } catch {
    return NextResponse.json(
      { error: { code: 'validation_failed', message: 'Request body must be JSON.' } },
      { status: 400 }
    );
  }

  const currentPassword = body.current_password ?? '';
  const newPassword = body.new_password ?? '';
  if (!currentPassword || !newPassword) {
    return NextResponse.json(
      { error: { code: 'validation_failed', message: 'current_password and new_password are required.' } },
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
    await changePassword(session.username, currentPassword, newPassword);
  } catch (error) {
    const message = (error as Error)?.message ?? 'password_change_failed';
    if (message === 'password_change_not_supported') {
      return NextResponse.json(
        {
          error: {
            code: 'not_supported',
            message: 'Password changes are disabled unless AUTH_USERS_FILE is configured.'
          }
        },
        { status: 409 }
      );
    }
    if (message === 'invalid_password') {
      return NextResponse.json({ error: { code: 'unauthorized', message: 'Current password is incorrect.' } }, { status: 401 });
    }
    return NextResponse.json({ error: { code: 'internal_error', message: 'Password change failed.' } }, { status: 500 });
  }

  const token = createSessionToken({ username: session.username, role: session.role });
  const response = NextResponse.json({ ok: true });
  response.headers.append('Set-Cookie', createSessionCookie(token));
  return response;
}
