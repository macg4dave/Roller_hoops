import { NextResponse } from 'next/server';

import { getSessionUser } from '../../../../../lib/auth/session';
import { proxyRequestToCore } from '../../../../../lib/core-api';

export async function GET(request: Request, context: { params: Promise<{ id: string }> }) {
  const session = await getSessionUser();
  if (!session) {
    return NextResponse.json({ error: { code: 'unauthorized', message: 'Authentication required.' } }, { status: 401 });
  }
  const { id } = await context.params;
  return proxyRequestToCore(request, `/api/v1/devices/${id}/tags`);
}

export async function PUT(request: Request, context: { params: Promise<{ id: string }> }) {
  const session = await getSessionUser();
  if (!session) {
    return NextResponse.json({ error: { code: 'unauthorized', message: 'Authentication required.' } }, { status: 401 });
  }
  const { id } = await context.params;
  return proxyRequestToCore(request, `/api/v1/devices/${id}/tags`);
}

