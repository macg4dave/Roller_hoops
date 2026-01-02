import { NextResponse } from 'next/server';
import { createSessionCookie, createSessionToken, verifyCredentials } from '../../../../lib/auth/session';

export async function POST(request: Request) {
  let body: { username?: string; password?: string };
  try {
    body = await request.json();
  } catch (error) {
    return NextResponse.json(
      {
        error: {
          code: 'validation_failed',
          message: 'Request body must be JSON with username and password.'
        }
      },
      { status: 400 }
    );
  }

  const username = body?.username?.trim() ?? '';
  const password = body?.password ?? '';

  if (!username || !password) {
    return NextResponse.json(
      {
        error: {
          code: 'validation_failed',
          message: 'Both username and password are required.'
        }
      },
      { status: 400 }
    );
  }

  const user = await verifyCredentials(username, password);
  if (!user) {
    return NextResponse.json(
      {
        error: {
          code: 'unauthorized',
          message: 'Invalid username or password.'
        }
      },
      { status: 401 }
    );
  }

  const token = createSessionToken(user);
  const response = NextResponse.json({ ok: true });
  response.headers.append('Set-Cookie', createSessionCookie(token, request));
  return response;
}
