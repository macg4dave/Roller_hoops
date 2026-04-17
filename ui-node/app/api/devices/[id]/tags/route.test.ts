import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('../../../../../lib/auth/session', () => ({
  getSessionUser: vi.fn()
}));

vi.mock('../../../../../lib/core-api', () => ({
  proxyRequestToCore: vi.fn()
}));

vi.mock('../../../../../lib/audit', () => ({
  auditEventForProxy: vi.fn(),
  writeAuditEvent: vi.fn()
}));

import { auditEventForProxy, writeAuditEvent } from '../../../../../lib/audit';
import { proxyRequestToCore } from '../../../../../lib/core-api';
import { getSessionUser } from '../../../../../lib/auth/session';
import { PUT } from './route';

const context = {
  params: Promise.resolve({ id: 'device-1' })
};

describe('device tags API route', () => {
  beforeEach(() => {
    vi.mocked(getSessionUser).mockReset();
    vi.mocked(proxyRequestToCore).mockReset();
    vi.mocked(auditEventForProxy).mockReset();
    vi.mocked(writeAuditEvent).mockReset();
  });

  it('rejects read-only tag updates before proxying to core', async () => {
    vi.mocked(getSessionUser).mockResolvedValue({ username: 'reader', role: 'read-only' });

    const response = await PUT(
      new Request('http://localhost/api/devices/device-1/tags', { method: 'PUT' }),
      context
    );

    expect(response.status).toBe(403);
    await expect(response.json()).resolves.toEqual({
      error: { code: 'forbidden', message: 'Read-only users cannot modify data.' }
    });
    expect(proxyRequestToCore).not.toHaveBeenCalled();
    expect(writeAuditEvent).not.toHaveBeenCalled();
  });

  it('audits writable tag updates after proxying to core', async () => {
    const session = { username: 'admin', role: 'admin' };
    const auditEvent = {
      actor: 'admin',
      actor_role: 'admin',
      action: 'device.update',
      target_type: 'device',
      target_id: 'device-1',
      details: { status: 200 }
    };
    const proxied = new Response(JSON.stringify([]), {
      status: 200,
      headers: { 'x-request-id': 'req-1' }
    });
    const request = new Request('http://localhost/api/devices/device-1/tags', { method: 'PUT' });

    vi.mocked(getSessionUser).mockResolvedValue(session);
    vi.mocked(proxyRequestToCore).mockResolvedValue(proxied);
    vi.mocked(auditEventForProxy).mockReturnValue(auditEvent);

    const response = await PUT(request, context);

    expect(response).toBe(proxied);
    expect(proxyRequestToCore).toHaveBeenCalledWith(request, '/api/v1/devices/device-1/tags');
    expect(auditEventForProxy).toHaveBeenCalledWith(session, 'PUT', '/api/v1/devices/device-1/tags', 200);
    expect(writeAuditEvent).toHaveBeenCalledWith(auditEvent, 'req-1');
  });
});
