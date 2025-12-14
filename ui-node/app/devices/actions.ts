'use server';

import { revalidatePath } from 'next/cache';

import { Device } from './types';

export type CreateDeviceState = {
  status: 'idle' | 'success' | 'error';
  message?: string;
};

const defaultState: CreateDeviceState = { status: 'idle' };

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
  const displayNameRaw = formData.get('display_name');
  const payload: Record<string, string> = {};

  if (typeof displayNameRaw === 'string' && displayNameRaw.trim().length > 0) {
    payload.display_name = displayNameRaw.trim();
  }

  const res = await fetch(`${apiBase()}/api/v1/devices`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
    cache: 'no-store',
    body: JSON.stringify(payload)
  });

  if (!res.ok) {
    const message = (await extractErrorMessage(res)) ?? `Request failed (${res.status})`;
    return { status: 'error', message };
  }

  const device = (await res.json()) as Device;
  revalidatePath('/devices');
  const label = device.display_name?.trim() || 'device';
  return { status: 'success', message: `Created ${label} (${device.id})` };
}

export function initialCreateDeviceState(): CreateDeviceState {
  return { ...defaultState };
}
