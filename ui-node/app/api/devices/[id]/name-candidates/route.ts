import { NextResponse } from 'next/server';
import { headers } from 'next/headers';
import { randomUUID } from 'crypto';

function apiBase() {
  return process.env.CORE_GO_BASE_URL ?? 'http://localhost:8081';
}

export async function GET(_request: Request, context: { params: Promise<{ id: string }> }) {
  const { id } = await context.params;
  const reqId = (await headers()).get('x-request-id') ?? randomUUID();

  const res = await fetch(`${apiBase()}/api/v1/devices/${id}/name-candidates`, {
    cache: 'no-store',
    headers: {
      Accept: 'application/json',
      'X-Request-ID': reqId
    }
  });

  const body = await res.text();
  return new NextResponse(body, {
    status: res.status,
    headers: {
      'Content-Type': res.headers.get('content-type') ?? 'application/json',
      'X-Request-ID': res.headers.get('x-request-id') ?? reqId
    }
  });
}

