import { proxyToCore } from '../../../../lib/core-api';

export async function GET(request: Request) {
  return proxyToCore(request, '/api/v1/devices/export', 'GET');
}
