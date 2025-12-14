import { randomUUID } from 'crypto';

const CORE_BASE = process.env.CORE_GO_BASE_URL ?? 'http://localhost:8081';

export async function proxyToCore(request: Request, path: string, method: string, body?: string) {
  const requestId = request.headers.get('x-request-id') ?? randomUUID();
  const headers = new Headers(request.headers);
  headers.delete('content-length');
  headers.set('X-Request-ID', requestId);
  headers.set('Accept', 'application/json');
  if (body != null) {
    headers.set('Content-Type', 'application/json');
  }

  const coreResponse = await fetch(`${CORE_BASE}${path}`, {
    method,
    headers,
    body
  });

  const payload = await coreResponse.arrayBuffer();
  const responseHeaders = new Headers(coreResponse.headers);
  responseHeaders.set('X-Request-ID', requestId);

  return new Response(payload, {
    status: coreResponse.status,
    headers: responseHeaders
  });
}
