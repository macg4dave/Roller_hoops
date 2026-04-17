import { NextResponse } from 'next/server';

import { getSessionUser } from '../../../../../lib/auth/session';
import { proxyRequestToCore } from '../../../../../lib/core-api';
import { auditEventForProxy, writeAuditEvent } from '../../../../../lib/audit';

export async function GET(request: Request, context: { params: Promise<{ id: string }> }) {
  const session = await getSessionUser();
  if (!session) {
    return NextResponse.json({ error: { code: 'unauthorized', message: 'Authentication required.' } }, { status: 401 });
  }
  const { id } = await context.params;
  return proxyRequestToCore(request, `/api/v1/devices/${encodeURIComponent(id)}/tags`);
}

export async function PUT(request: Request, context: { params: Promise<{ id: string }> }) {
  const session = await getSessionUser();
  if (!session) {
    return NextResponse.json({ error: { code: 'unauthorized', message: 'Authentication required.' } }, { status: 401 });
  }
  if (session.role === 'read-only') {
    return NextResponse.json({ error: { code: 'forbidden', message: 'Read-only users cannot modify data.' } }, { status: 403 });
  }
  const { id } = await context.params;
  const path = `/api/v1/devices/${encodeURIComponent(id)}/tags`;
  const proxied = await proxyRequestToCore(request, path);
  const reqId = proxied.headers.get('x-request-id') ?? undefined;
  await writeAuditEvent(auditEventForProxy(session, 'PUT', path, proxied.status), reqId);
  return proxied;
}
