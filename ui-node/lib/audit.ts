import { randomUUID } from 'crypto';
import type { SessionUser } from './auth/session';

function coreBase() {
  return process.env.CORE_GO_BASE_URL ?? 'http://localhost:8081';
}

export type AuditEvent = {
  actor: string;
  actor_role?: string;
  action: string;
  target_type?: string;
  target_id?: string;
  details?: Record<string, unknown>;
};

export async function writeAuditEvent(event: AuditEvent, requestId?: string) {
  try {
    const reqId = requestId ?? randomUUID();
    await fetch(`${coreBase()}/api/v1/audit/events`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Accept: 'application/json',
        'X-Request-ID': reqId
      },
      cache: 'no-store',
      body: JSON.stringify(event)
    });
  } catch {
    // Best-effort by design; do not fail user workflows on audit logging.
  }
}

export function auditEventForProxy(session: SessionUser, method: string, path: string, status: number): AuditEvent {
  const normalizedMethod = method.toUpperCase();
  const normalizedPath = path.startsWith('/api') ? path.slice('/api'.length) : path;

  const event: AuditEvent = {
    actor: session.username,
    actor_role: session.role,
    action: `${normalizedMethod} ${normalizedPath}`,
    details: { status }
  };

  const segments = normalizedPath.split('?')[0].split('/').filter(Boolean);
  if (segments.length >= 2 && segments[0] === 'v1') {
    if (segments[1] === 'devices') {
      if (normalizedMethod === 'POST' && segments.length === 2) {
        event.action = 'device.create';
        event.target_type = 'device';
      } else if (normalizedMethod === 'POST' && segments.length >= 3 && segments[2] === 'import') {
        event.action = 'device.import_snapshot';
        event.target_type = 'device';
      } else if (segments.length >= 3) {
        const id = segments[2];
        event.target_type = 'device';
        event.target_id = id;
        if (normalizedMethod === 'PUT') {
          event.action = 'device.update';
        }
      }
    }
    if (segments[1] === 'discovery' && normalizedMethod === 'POST' && segments[2] === 'run') {
      event.action = 'discovery.trigger';
      event.target_type = 'discovery';
    }
    if (segments[1] === 'inventory' && normalizedMethod === 'POST') {
      event.action = 'inventory.import';
      event.target_type = 'inventory';
    }
  }
  return event;
}
