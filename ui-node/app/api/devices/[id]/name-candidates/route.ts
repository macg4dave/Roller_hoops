import { NextResponse } from 'next/server';

import { getSessionUser } from '../../../../../lib/auth/session';
import { proxyRequestToCore } from '../../../../../lib/core-api';

export async function GET(_request: Request, context: { params: Promise<{ id: string }> }) {
  const session = await getSessionUser();
  if (!session) {
    return NextResponse.json({ error: { code: 'unauthorized', message: 'Authentication required.' } }, { status: 401 });
  }
  const { id } = await context.params;
  return proxyRequestToCore(_request, `/api/v1/devices/${id}/name-candidates`);
}
