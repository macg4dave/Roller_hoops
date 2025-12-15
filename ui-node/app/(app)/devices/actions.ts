'use server';

import { revalidatePath } from 'next/cache';
import { headers } from 'next/headers';
import { randomUUID } from 'crypto';

import { Device, DiscoveryRun } from './types';

import { CreateDeviceState, DiscoveryRunState, DeviceMetadataState, DeviceDisplayNameState } from './state';
import { getSessionUser } from '../../lib/auth/session';
import { writeAuditEvent } from '../../lib/audit';

function apiBase() {
  return process.env.CORE_GO_BASE_URL ?? 'http://localhost:8081';
}

async function extractErrorMessage(res: Response) {
  try {
    const body = (await res.json()) as { error?: { message?: string } };
    return body?.error?.message;
  } catch {
    return undefined;
  }
}

export async function createDevice(
  _prevState: CreateDeviceState,
  formData: FormData
): Promise<CreateDeviceState> {
  const session = await getSessionUser();
  if (!session) {
    return { status: 'error', message: 'Authentication required.' };
  }
  if (session.role === 'read-only') {
    return { status: 'error', message: 'Read-only users cannot create devices.' };
  }

  const displayNameRaw = formData.get('display_name');
  const payload: Record<string, unknown> = {};

  if (typeof displayNameRaw === 'string' && displayNameRaw.trim().length > 0) {
    payload.display_name = displayNameRaw.trim();
  }

  const ownerRaw = formData.get('owner');
  const locationRaw = formData.get('location');
  const notesRaw = formData.get('notes');

  const metadata: Record<string, string> = {};
  if (typeof ownerRaw === 'string' && ownerRaw.trim().length > 0) {
    metadata.owner = ownerRaw.trim();
  }
  if (typeof locationRaw === 'string' && locationRaw.trim().length > 0) {
    metadata.location = locationRaw.trim();
  }
  if (typeof notesRaw === 'string' && notesRaw.trim().length > 0) {
    metadata.notes = notesRaw.trim();
  }

  if (Object.keys(metadata).length > 0) {
    payload.metadata = metadata;
  }

  const reqId = (await headers()).get('x-request-id') ?? randomUUID();

  const res = await fetch(`${apiBase()}/api/v1/devices`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json',
      'X-Request-ID': reqId
    },
    cache: 'no-store',
    body: JSON.stringify(payload)
  });

  if (!res.ok) {
    const message = (await extractErrorMessage(res)) ?? `Request failed (${res.status})`;
    return { status: 'error', message };
  }

  const device = (await res.json()) as Device;
  await writeAuditEvent(
    {
      actor: session.username,
      actor_role: session.role,
      action: 'device.create',
      target_type: 'device',
      target_id: device.id,
      details: { display_name: device.display_name ?? null }
    },
    reqId
  );
  revalidatePath('/devices');
  const label = device.display_name?.trim() || 'device';
  return { status: 'success', message: `Created ${label} (${device.id})` };
}

export async function triggerDiscovery(
  _prevState: DiscoveryRunState,
  formData: FormData
): Promise<DiscoveryRunState> {
  const session = await getSessionUser();
  if (!session) {
    return { status: 'error', message: 'Authentication required.' };
  }
  if (session.role === 'read-only') {
    return { status: 'error', message: 'Read-only users cannot trigger discoveries.' };
  }

  const scopeRaw = formData.get('scope');
  const payload: Record<string, unknown> = {};
  if (typeof scopeRaw === 'string' && scopeRaw.trim().length > 0) {
    payload.scope = scopeRaw.trim();
  }

  const reqId = (await headers()).get('x-request-id') ?? randomUUID();
  const res = await fetch(`${apiBase()}/api/v1/discovery/run`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json',
      'X-Request-ID': reqId
    },
    cache: 'no-store',
    body: JSON.stringify(payload)
  });

  if (!res.ok) {
    const message = (await extractErrorMessage(res)) ?? `Request failed (${res.status})`;
    return { status: 'error', message };
  }

  const run = (await res.json()) as DiscoveryRun;
  await writeAuditEvent(
    {
      actor: session.username,
      actor_role: session.role,
      action: 'discovery.trigger',
      target_type: 'discovery_run',
      target_id: run.id,
      details: { scope: run.scope ?? null }
    },
    reqId
  );
  revalidatePath('/devices');
  return {
    status: 'success',
    message: `Discovery run ${run.id} queued (${run.status})`
  };
}

export async function updateDeviceMetadata(
  _prevState: DeviceMetadataState,
  formData: FormData
): Promise<DeviceMetadataState> {
  const session = await getSessionUser();
  if (!session) {
    return { status: 'error', message: 'Authentication required.' };
  }
  if (session.role === 'read-only') {
    return { status: 'error', message: 'Read-only users cannot modify metadata.' };
  }

  const deviceId = formData.get('device_id');
  if (typeof deviceId !== 'string' || deviceId.trim().length === 0) {
    return { status: 'error', message: 'missing device id' };
  }

  const metadata: Record<string, string> = {};
  const ownerRaw = formData.get('owner');
  const locationRaw = formData.get('location');
  const notesRaw = formData.get('notes');

  const addField = (key: string, raw: FormDataEntryValue | null) => {
    if (typeof raw !== 'string') return;
    metadata[key] = raw.trim();
  };

  addField('owner', ownerRaw);
  addField('location', locationRaw);
  addField('notes', notesRaw);

  const payload: Record<string, unknown> = {};
  if (Object.keys(metadata).length > 0) {
    payload.metadata = metadata;
  }

  if (Object.keys(payload).length === 0) {
    return { status: 'error', message: 'nothing to update' };
  }

  const reqId = (await headers()).get('x-request-id') ?? randomUUID();
  const res = await fetch(`${apiBase()}/api/v1/devices/${deviceId}`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json',
      'X-Request-ID': reqId
    },
    cache: 'no-store',
    body: JSON.stringify(payload)
  });

  if (!res.ok) {
    const message = (await extractErrorMessage(res)) ?? `Request failed (${res.status})`;
    return { status: 'error', message };
  }

  const device = (await res.json()) as Device;
  await writeAuditEvent(
    {
      actor: session.username,
      actor_role: session.role,
      action: 'device.metadata.update',
      target_type: 'device',
      target_id: device.id,
      details: payload
    },
    reqId
  );
  revalidatePath('/devices');

  return { status: 'success', message: `Metadata updated (${device.id})` };
}

export async function updateDeviceDisplayName(
  _prevState: DeviceDisplayNameState,
  formData: FormData
): Promise<DeviceDisplayNameState> {
  const session = await getSessionUser();
  if (!session) {
    return { status: 'error', message: 'Authentication required.' };
  }
  if (session.role === 'read-only') {
    return { status: 'error', message: 'Read-only users cannot modify devices.' };
  }

  const deviceId = formData.get('device_id');
  if (typeof deviceId !== 'string' || deviceId.trim().length === 0) {
    return { status: 'error', message: 'missing device id' };
  }

  const displayNameRaw = formData.get('display_name');
  if (typeof displayNameRaw !== 'string' || displayNameRaw.trim().length === 0) {
    return { status: 'error', message: 'missing display name' };
  }
  const displayName = displayNameRaw.trim();

  const reqId = (await headers()).get('x-request-id') ?? randomUUID();
  const res = await fetch(`${apiBase()}/api/v1/devices/${deviceId}`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json',
      'X-Request-ID': reqId
    },
    cache: 'no-store',
    body: JSON.stringify({ display_name: displayName })
  });

  if (!res.ok) {
    const message = (await extractErrorMessage(res)) ?? `Request failed (${res.status})`;
    return { status: 'error', message };
  }

  const device = (await res.json()) as Device;
  await writeAuditEvent(
    {
      actor: session.username,
      actor_role: session.role,
      action: 'device.display_name.update',
      target_type: 'device',
      target_id: device.id,
      details: { display_name: device.display_name ?? null }
    },
    reqId
  );
  revalidatePath('/devices');
  const label = device.display_name?.trim() || 'device';
  return { status: 'success', message: `Display name set to ${label}` };
}
