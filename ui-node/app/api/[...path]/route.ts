import { NextResponse } from 'next/server';
import { getSessionUser } from '../../../lib/auth/session';
import { proxyRequestToCore } from '../../../lib/core-api';
import { auditEventForProxy, writeAuditEvent } from '../../../lib/audit';

type RouteContext = {
  params: Promise<{ path: string[] }>;
};

async function proxy(request: Request, context: RouteContext) {
  const { path } = await context.params;
  const url = new URL(request.url);
  const suffix = path.length > 0 ? `/${path.join('/')}` : '';
  const method = request.method.toUpperCase();
  const session = await getSessionUser();

  if (!session) {
    return NextResponse.json(
      { error: { code: 'unauthorized', message: 'Authentication required.' } },
      { status: 401 }
    );
  }

  const mutatingMethods = new Set(['POST', 'PUT', 'PATCH', 'DELETE']);
  if (mutatingMethods.has(method) && session.role === 'read-only') {
    return NextResponse.json(
      { error: { code: 'forbidden', message: 'Read-only users cannot modify data.' } },
      { status: 403 }
    );
  }

  const proxied = await proxyRequestToCore(request, `/api${suffix}${url.search}`);
  if (mutatingMethods.has(method)) {
    const reqId = proxied.headers.get('x-request-id') ?? undefined;
    await writeAuditEvent(auditEventForProxy(session, method, `/api${suffix}${url.search}`, proxied.status), reqId);
  }
  return proxied;
}

export async function GET(request: Request, context: RouteContext) {
  return proxy(request, context);
}

export async function POST(request: Request, context: RouteContext) {
  return proxy(request, context);
}

export async function PUT(request: Request, context: RouteContext) {
  return proxy(request, context);
}

export async function PATCH(request: Request, context: RouteContext) {
  return proxy(request, context);
}

export async function DELETE(request: Request, context: RouteContext) {
  return proxy(request, context);
}

export async function HEAD(request: Request, context: RouteContext) {
  return proxy(request, context);
}

export async function OPTIONS(request: Request, context: RouteContext) {
  return proxy(request, context);
}
