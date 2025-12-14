import { proxyToCore } from '../../../../lib/core-api';

export async function POST(request: Request) {
  const payload = await request.text();
  return proxyToCore(request, '/api/v1/devices/import', 'POST', payload);
}
