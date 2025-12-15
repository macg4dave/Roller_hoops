import { NextResponse } from 'next/server';
import { getSessionUser } from '../../../../lib/auth/session';
import { proxyRequestToCore } from '../../../../lib/core-api';
import { auditEventForProxy, writeAuditEvent } from '../../../../lib/audit';

export async function POST(request: Request) {
  const session = await getSessionUser();
  if (!session) {
    return NextResponse.json({ error: { code: 'unauthorized', message: 'Authentication required.' } }, { status: 401 });
  }
  if (session.role === 'read-only') {
    return NextResponse.json({ error: { code: 'forbidden', message: 'Read-only users cannot modify data.' } }, { status: 403 });
  }
  const proxied = await proxyRequestToCore(request, '/api/v1/devices/import');
  const reqId = proxied.headers.get('x-request-id') ?? undefined;
  await writeAuditEvent(auditEventForProxy(session, 'POST', '/api/v1/devices/import', proxied.status), reqId);
  return proxied;
}
